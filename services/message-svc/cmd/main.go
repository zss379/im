package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/shulian-paas/im/message-svc/internal/config"
	"github.com/shulian-paas/im/message-svc/internal/handler"
	"github.com/shulian-paas/im/message-svc/internal/middleware"
	"github.com/shulian-paas/im/message-svc/internal/metrics"
	"github.com/shulian-paas/im/message-svc/internal/mq"
	"github.com/shulian-paas/im/message-svc/internal/repo"
	"github.com/shulian-paas/im/message-svc/internal/service"
)

func main() {
	cfgPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		// 默认配置兜底
		cfg = config.DefaultConfig()
	}

	setupLogger(cfg.Log)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoDB.URI))
	if err != nil {
		log.Fatal().Err(err).Msg("connect mongodb")
	}

	if err := mongoClient.Ping(ctx, nil); err != nil {
		log.Fatal().Err(err).Msg("ping mongodb")
	}

	db := mongoClient.Database(cfg.MongoDB.Database)

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatal().Err(err).Msg("ping redis")
	}

	producer := mq.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.TopicMessagePush, cfg.Kafka.TopicBotTrigger)

	msgRepo := repo.NewMessageRepo(db)
	msgCache := repo.NewMessageCache(rdb)
	msgSvc := service.NewMessageService(msgRepo, msgCache, producer, cfg.Rate.BotMsgPerSec)
	msgHandler := handler.NewMessageHandler(msgSvc, msgCache)

	r := gin.New()
	r.Use(
		gin.Recovery(),
		middleware.LoggerMiddleware(),
		middleware.RecoveryMiddleware(),
		middleware.AuthMiddleware(cfg.OpenIM.Secret),
		middleware.TenantMiddleware(),
	)

	api := r.Group("/api/v1")
	msgHandler.RegisterRoutes(api)

	if cfg.Prometheus.Enabled {
		r.GET(cfg.Prometheus.Path, metrics.Handler())
	}

	srv := &http.Server{
		Addr:    formatAddr(cfg.Server.Port),
		Handler: r,
	}

	go func() {
		log.Info().Int("port", cfg.Server.Port).Msg("server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server listen")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting down server")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal().Err(err).Msg("server shutdown")
	}

	producer.Close()
	mongoClient.Disconnect(context.Background())
	rdb.Close()
	log.Info().Msg("server exited")
}

func setupLogger(cfg config.LogConfig) {
	level, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		level = zerolog.DebugLevel
	}
	zerolog.SetGlobalLevel(level)

	if cfg.JSON {
		log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	} else {
		log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "15:04:05"}).With().Timestamp().Logger()
	}
}

func formatAddr(port int) string {
	if port <= 0 {
		port = 8081
	}
	return ":" + itoa(port)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 10)
	for n > 0 {
		buf = append(buf, byte('0'+n%10))
		n /= 10
	}
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}
