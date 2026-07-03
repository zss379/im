# ============================================================
# 数莲 PaaS IM 系统 — 多阶段构建 Dockerfile
# 构建上下文必须为每个服务的根目录 (services/<name>/)
# 每个服务是独立的 Go module，拥有自己的 go.mod
#
# docker-compose 使用（推荐）:
#   docker compose -f deploy/docker/docker-compose.yml up -d
#
# 手动构建单个服务:
#   docker build -t im/auth-svc \
#     -f deploy/docker/Dockerfile \
#     services/auth-svc
# ============================================================

# === Stage 1: Build ===
FROM golang:1.21-alpine AS builder

RUN apk --no-cache add git ca-certificates tzdata

WORKDIR /workspace

# 缓存依赖下载层
COPY go.mod go.sum ./
RUN go mod download

# 复制源码
COPY . .

# 编译（CGO_ENABLED=0 生成静态二进制）
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" \
    -o /app/bin/service ./cmd

# === Stage 2: Runtime ===
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/bin/service .

ARG SERVICE_PORT=8080
EXPOSE $SERVICE_PORT

ENTRYPOINT ["./service"]
