module github.com/shulian-paas/im/group-svc

go 1.21

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/go-redis/redis/v8 v8.11.5
	github.com/golang-jwt/jwt/v5 v5.1.0
	github.com/prometheus/client_golang v1.16.0
	github.com/rs/zerolog v1.31.0
	gopkg.in/yaml.v3 v3.0.1
	gorm.io/driver/mysql v1.5.2
	gorm.io/gorm v1.25.5
)

require github.com/alicebob/miniredis/v2 v2.31.1 // indirect
