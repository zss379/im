package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"
	_ "github.com/go-sql-driver/mysql"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/shulian-paas/im/bot-svc/internal/config"
	"github.com/shulian-paas/im/bot-svc/internal/consumer"
	"github.com/shulian-paas/im/bot-svc/internal/handler"
	"github.com/shulian-paas/im/bot-svc/internal/repo"
	"github.com/shulian-paas/im/bot-svc/internal/service"
	"github.com/shulian-paas/im/bot-svc/internal/sse"
)

func main() {
	// 1. 加载配置
	cfgPath := os.Getenv("BOT_SVC_CONFIG")
	if cfgPath == "" {
		cfgPath = "config.yaml"
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Warn().Err(err).Msg("config load failed, using defaults")
		cfg = config.DefaultConfig()
	}

	// 2. 初始化日志
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	if cfg.Log.JSON {
		log.Logger = log.Output(os.Stdout)
	} else {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	}

	// 3. 连接 MySQL
	db, err := sqlx.Connect("mysql", cfg.MySQL.DSN)
	if err != nil {
		log.Fatal().Err(err).Msg("connect mysql failed")
	}
	defer db.Close()
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	// 4. 连接 Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Warn().Err(err).Msg("redis ping failed, continuing...")
	}

	// 5. 初始化各层
	// Repos
	botRepo := repo.NewBotRepo(db)
	botCache := repo.NewBotCache(rdb)
	openimClient := repo.NewOpenIMClient(&cfg.OpenIM)

	// SSE pool
	ssePool := sse.NewPool(cfg.SSE.MaxConnections)

	// Message client
	msgClient := service.NewMessageClient(fmt.Sprintf("http://localhost:%d", 8081)) // message-svc default port

	// Webhook services
	syncSvc := service.NewSyncWebhookService(&cfg.Webhook)
	asyncSvc := service.NewAsyncWebhookService(&cfg.Webhook, botCache)
	sseSvc := service.NewSSEWebhookService(&cfg.SSE, ssePool, msgClient)

	// Bot orchestrator
	botSvc := service.NewBotService(botCache, botRepo, msgClient, syncSvc, asyncSvc, sseSvc)

	// 6. 启动 Kafka consumer
	botConsumer := consumer.NewBotTriggerConsumer(cfg.Kafka.Brokers, cfg.Kafka.Topic, cfg.Kafka.GroupID, botSvc)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		if err := botConsumer.Start(ctx); err != nil {
			log.Fatal().Err(err).Msg("bot_trigger consumer stopped")
		}
	}()

	// 7. 启动异步回调清理协程 (task 5.5)
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				asyncSvc.CleanupExpired(context.Background())
			case <-ctx.Done():
				return
			}
		}
	}()

	// 8. 初始化 Redis 缓存 (启动时加载活跃机器人)
	go initBotCache(context.Background(), botRepo, botCache)

	// 9. 设置 HTTP 路由
	r := gin.Default()

	// Prometheus metrics
	if cfg.Prometheus.Enabled {
		r.GET(cfg.Prometheus.Path, gin.WrapH(promhttp.Handler()))
	}

	// Bot CRUD handlers
	botHandler := handler.NewBotHandler(botRepo, botCache)
	r.GET("/bots", botHandler.ListBots)
	r.POST("/bots", botHandler.CreateBot)
	r.POST("/bots/:bot_id/toggle", botHandler.ToggleBot)
	r.DELETE("/bots/:bot_id", botHandler.DeleteBot)

	// Callback handler
	cbHandler := handler.NewCallbackHandler(asyncSvc, msgClient, botCache)
	r.POST("/internal/bot/callback", cbHandler.HandleCallback)

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 10. 启动 HTTP server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: r,
	}

	go func() {
		log.Info().Int("port", cfg.Server.Port).Msg("bot-svc starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("http server failed")
		}
	}()

	// 11. 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("shutting down bot-svc...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	_ = botConsumer.Close()
	_ = srv.Shutdown(shutdownCtx)
	log.Info().Msg("bot-svc stopped")
}

// initBotCache 启动时加载所有活跃机器人配置到 Redis
func initBotCache(ctx context.Context, botRepo *repo.BotRepo, cache *repo.BotCache) {
	ids, err := botRepo.GetActiveBotIDs()
	if err != nil {
		log.Err(err).Msg("init bot cache: get active bot IDs failed")
		return
	}
	for _, id := range ids {
		_ = cache.AddUserID(ctx, id)
		bot, err := botRepo.GetByID(id)
		if err != nil {
			continue
		}
		_ = cache.SetConfig(ctx, bot)
	}
	log.Info().Int("count", len(ids)).Msg("bot cache initialized")
}
