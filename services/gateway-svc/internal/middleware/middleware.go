package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog/log"
)

// ============================================================================
// Auth Middleware
// ============================================================================

type AuthMiddleware struct {
	jwtSecret   string
	publicPaths map[string]bool
}

func NewAuth(jwtSecret string, publicPaths []string) *AuthMiddleware {
	p := make(map[string]bool, len(publicPaths))
	for _, pp := range publicPaths {
		p[pp] = true
	}
	return &AuthMiddleware{jwtSecret: jwtSecret, publicPaths: p}
}

func (a *AuthMiddleware) Handle(c *gin.Context) {
	key := c.Request.Method + ":" + c.Request.URL.Path
	if a.publicPaths[key] {
		c.Next()
		return
	}

	tokenStr := extractBearerToken(c)
	if tokenStr == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "missing authorization header"})
		return
	}

	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(a.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "invalid or expired token"})
		return
	}

	if userID, ok := claims["user_id"]; ok {
		c.Request.Header.Set("X-User-ID", fmt.Sprintf("%v", userID))
	}
	if tenantID, ok := claims["tenant_id"]; ok {
		c.Request.Header.Set("X-Tenant-ID", fmt.Sprintf("%v", tenantID))
	}
	c.Next()
}

func extractBearerToken(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

// ============================================================================
// CORS Middleware
// ============================================================================

type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
}

func CORS(cfg CORSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", strings.Join(cfg.AllowedMethods, ", "))
		c.Header("Access-Control-Allow-Headers", strings.Join(cfg.AllowedHeaders, ", "))
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// ============================================================================
// Rate Limiter (In-Memory Token Bucket)
// ============================================================================

type tokenBucket struct {
	tokens     float64
	lastRefill time.Time
}

type RateLimiter struct {
	mu         sync.RWMutex
	buckets    map[string]*tokenBucket
	refillRate int
	bucketSize int
}

func NewRateLimiter(refillRate, bucketSize int) *RateLimiter {
	return &RateLimiter{
		buckets:    make(map[string]*tokenBucket),
		refillRate: refillRate,
		bucketSize: bucketSize,
	}
}

func (rl *RateLimiter) Handle(c *gin.Context) {
	if rl.refillRate <= 0 || rl.bucketSize <= 0 {
		c.Next()
		return
	}

	ip := c.ClientIP()
	rl.mu.Lock()
	b, ok := rl.buckets[ip]
	if !ok {
		b = &tokenBucket{
			tokens:     float64(rl.bucketSize),
			lastRefill: time.Now(),
		}
		rl.buckets[ip] = b
	}

	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens += elapsed * float64(rl.refillRate)
	if b.tokens > float64(rl.bucketSize) {
		b.tokens = float64(rl.bucketSize)
	}
	b.lastRefill = now

	if b.tokens < 1 {
		rl.mu.Unlock()
		c.Header("Retry-After", "1")
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"code": 429, "message": "rate limit exceeded"})
		return
	}

	b.tokens--
	rl.mu.Unlock()
	c.Next()
}

// ============================================================================
// Prometheus Metrics Middleware
// ============================================================================

var (
	reqCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gateway_requests_total",
		Help: "Total HTTP requests",
	}, []string{"method", "route", "status"})

	reqDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "gateway_request_duration_seconds",
		Help:    "HTTP request duration",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "route"})
)

func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		status := fmt.Sprintf("%d", c.Writer.Status())
		reqCount.WithLabelValues(c.Request.Method, path, status).Inc()
		reqDuration.WithLabelValues(c.Request.Method, path).Observe(time.Since(start).Seconds())
	}
}

// ============================================================================
// Recovery / Access Log
// ============================================================================

func AccessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Since(start)
		log.Info().
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status", c.Writer.Status()).
			Dur("latency", latency).
			Str("client", c.ClientIP()).
			Msg("access")
	}
}
