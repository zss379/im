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
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/shulian-paas/im/gateway-svc/internal/config"
	"github.com/shulian-paas/im/gateway-svc/internal/handler"
	"github.com/shulian-paas/im/gateway-svc/internal/middleware"
	"github.com/shulian-paas/im/gateway-svc/internal/proxy"
)

func main() {
	cfg := loadConfig()
	setupLogger(cfg.Log)

	pm := proxy.NewManager(parseRoutes(cfg.Routes))

	hc := handler.NewHealthChecker(pm.Backends(),
		parseDuration(cfg.HealthCheck.Interval, 30*time.Second),
		parseDuration(cfg.HealthCheck.Timeout, 5*time.Second))
	hc.Start()

	r := setupRouter(cfg, pm, hc)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", strconv.Itoa(cfg.Server.Port)),
		Handler: r,
	}

	go func() {
		log.Info().Int("port", cfg.Server.Port).Msg("gateway-svc starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("shutting down...")

	hc.Stop()

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

func setupRouter(cfg *config.Config, pm *proxy.Manager, hc *handler.HealthChecker) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.AccessLog())

	if cfg.CORS.Enabled {
		r.Use(middleware.CORS(middleware.CORSConfig{
			AllowedOrigins: cfg.CORS.AllowedOrigins,
			AllowedMethods: cfg.CORS.AllowedMethods,
			AllowedHeaders: cfg.CORS.AllowedHeaders,
		}))
	}

	r.Use(middleware.Metrics())

	if cfg.RateLimit.Enabled {
		rl := middleware.NewRateLimiter(cfg.RateLimit.RefillRate, cfg.RateLimit.BucketSize)
		r.Use(rl.Handle)
	}

	if cfg.Auth.JWTSecret != "" {
		authMw := middleware.NewAuth(cfg.Auth.JWTSecret, cfg.Auth.PublicPaths)
		r.Use(authMw.Handle)
	}

	// Proxy routes for each backend
	for _, route := range cfg.Routes {
		rp := pm.Find(route.Prefix)
		if rp == nil {
			continue
		}
		r.Any(route.Prefix+"/*path", gin.WrapH(rp.Proxy()))
	}

	// Health check endpoint
	r.GET("/health", hc.Handle)

	// Prometheus metrics
	if cfg.Prometheus.Enabled {
		r.GET(cfg.Prometheus.Path, gin.WrapH(promhttp.Handler()))
	}

	return r
}

func parseRoutes(routes []config.RouteConfig) []struct {
	Prefix  string
	Backend string
	Timeout time.Duration
} {
	result := make([]struct {
		Prefix  string
		Backend string
		Timeout time.Duration
	}, len(routes))
	for i, r := range routes {
		result[i] = struct {
			Prefix  string
			Backend string
			Timeout time.Duration
		}{
			Prefix:  r.Prefix,
			Backend: r.Backend,
			Timeout: r.ParseTimeout(),
		}
	}
	return result
}

func parseDuration(s string, fallback time.Duration) time.Duration {
	if s == "" {
		return fallback
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return fallback
	}
	return d
}
