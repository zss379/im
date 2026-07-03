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
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/shulian-paas/im/session-svc/internal/config"
	"github.com/shulian-paas/im/session-svc/internal/handler"
	"github.com/shulian-paas/im/session-svc/internal/middleware"
	"github.com/shulian-paas/im/session-svc/internal/mq"
	"github.com/shulian-paas/im/session-svc/internal/repo"
	"github.com/shulian-paas/im/session-svc/internal/service"
)

func main() {
	cfg := loadConfig()
	setupLogger(cfg.Log)

	db := openDB(cfg.MySQL.DSN)
	rdb := openRedis(cfg.Redis)

	mysqlRepo := repo.NewMySQLRepo(db)
	cache := repo.NewCache(rdb)

	if err := mysqlRepo.AutoMigrate(); err != nil {
		log.Fatal().Err(err).Msg("failed to auto-migrate database")
	}

	svc := service.NewSessionService(mysqlRepo, cache)

	consumer := mq.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.TopicMessagePush, cfg.Kafka.ConsumerGroupID, mysqlRepo, cache)
	producer := mq.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.TopicSessionSync)
	_ = producer // 下阶段发布会话变更事件时使用

	r := setupRouter(svc, cfg)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", strconv.Itoa(cfg.Server.Port)),
		Handler: r,
	}

	go func() {
		log.Info().Int("port", cfg.Server.Port).Msg("session-svc starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	// 启动 Kafka consumer（独立 goroutine）
	consumer.Start(context.Background())

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("shutting down...")

	// 优雅关闭
	consumer.Stop()

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

func setupRouter(svc *service.SessionService, cfg *config.Config) *gin.Engine {
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
		sessionHandler := handler.NewSessionHandler(svc)
		sessionHandler.RegisterRoutes(api)
	}

	return r
}
