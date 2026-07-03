package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/shulian-paas/im/file-svc/internal/config"
	"github.com/shulian-paas/im/file-svc/internal/handler"
	"github.com/shulian-paas/im/file-svc/internal/middleware"
	"github.com/shulian-paas/im/file-svc/internal/repo"
	"github.com/shulian-paas/im/file-svc/internal/service"
)

func main() {
	cfg := loadConfig()
	setupLogger(cfg.Log)

	db := openDB(cfg.MySQL.DSN)
	rdb := openRedis(cfg.Redis)
	mc := openMinIO(cfg.MinIO)

	mysqlRepo := repo.NewMySQLRepo(db)
	_ = rdb // Redis reserved for future use
	storage := repo.NewStorage(mc, cfg.MinIO.Bucket)

	if err := mysqlRepo.AutoMigrate(); err != nil {
		log.Fatal().Err(err).Msg("failed to auto-migrate database")
	}

	if err := storage.EnsureBucket(context.Background()); err != nil {
		log.Warn().Err(err).Msg("failed to ensure MinIO bucket, may need manual setup")
	}

	svc := service.NewFileService(mysqlRepo, storage)
	r := setupRouter(svc, cfg)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", strconv.Itoa(cfg.Server.Port)),
		Handler: r,
	}

	go func() {
		log.Info().Int("port", cfg.Server.Port).Msg("file-svc starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("forced shutdown")
	}
	log.Info().Msg("server exited")
}

func loadConfig() *config.Config {
	path := "config.yaml"
	if p := os.Getenv("CONFIG_PATH"); p != "" {
		path = p
	}
	cfg, err := config.Load(path)
	if err != nil {
		log.Warn().Err(err).Msg("failed to load config, using defaults")
		return config.DefaultConfig()
	}
	return cfg
}

func setupLogger(cfg config.LogConfig) {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	if cfg.JSON {
		log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	} else {
		log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "2006-01-02 15:04:05"}).With().Timestamp().Logger()
	}
}

func openDB(dsn string) *gorm.DB {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to MySQL")
	}
	return db
}

func openRedis(cfg config.RedisConfig) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatal().Err(err).Msg("failed to connect to Redis")
	}
	return rdb
}

func openMinIO(cfg config.MinIOConfig) *minio.Client {
	mc, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create MinIO client")
	}
	return mc
}

func setupRouter(svc *service.FileService, cfg *config.Config) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.LoggerMiddleware())
	r.Use(middleware.RecoveryMiddleware())
	r.Use(middleware.AuthMiddleware(cfg.Server.JWTSecret))

	if cfg.Prometheus.Enabled {
		r.GET(cfg.Prometheus.Path, gin.WrapH(promhttp.Handler()))
	}

	api := r.Group("/api/v1")
	{
		fileHandler := handler.NewFileHandler(svc)
		fileHandler.RegisterRoutes(api)
	}

	return r
}
