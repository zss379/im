package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog/log"
)

// Claims JWT 声明
type Claims struct {
	UserID   int64  `json:"user_id"`
	TenantID int64  `json:"tenant_id"`
	DeviceID string `json:"device_id"`
	jwt.RegisteredClaims
}

// AuthMiddleware Bearer Token 认证中间件
func AuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 40002, "msg": "missing token"})
			return
		}

		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		if tokenStr == auth {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 40002, "msg": "invalid token format"})
			return
		}

		token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 40002, "msg": "token expired or invalid"})
			return
		}

		claims, ok := token.Claims.(*Claims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 40002, "msg": "invalid token claims"})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("tenant_id", claims.TenantID)
		c.Set("device_id", claims.DeviceID)
		c.Next()
	}
}

// TenantMiddleware 从请求头提取租户ID（内部接口使用）
func TenantMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.GetHeader("X-Tenant-ID")
		if tenantID == "" {
			// 内部调用可省略，从 token 中获取
			c.Next()
			return
		}
		c.Set("x_tenant_id", tenantID)
		c.Next()
	}
}

// LoggerMiddleware 请求日志
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Info().Str("method", c.Request.Method).Str("path", c.Request.URL.Path).
			Int("status", c.Writer.Status()).Msg("request")
		c.Next()
	}
}

// RecoveryMiddleware 异常恢复
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Error().Interface("panic", err).Msg("panic recovered")
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": 50001, "msg": "internal error"})
			}
		}()
		c.Next()
	}
}
