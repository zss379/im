# ============================================================
# 数莲 PaaS IM 系统 — 多阶段构建 Dockerfile
# 所有微服务共用此文件，通过 target 选择编译目标
#
# 构建单个服务:
#   docker build --target=user-svc -t im/user-svc .
#
# 完整构建（docker-compose 会自动选择 target）:
#   docker-compose build
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

# === Target: api-gateway ===
FROM builder AS api-gateway-builder
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w -X main.version=$(git describe --tags 2>/dev/null || echo dev)" \
    -o /app/bin/service ./cmd/api-gateway

# === Target: user-svc ===
FROM builder AS user-svc-builder
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" \
    -o /app/bin/service ./cmd/user-svc

# === Target: message-svc ===
FROM builder AS message-svc-builder
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" \
    -o /app/bin/service ./cmd/message-svc

# === Target: conversation-svc ===
FROM builder AS conversation-svc-builder
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" \
    -o /app/bin/service ./cmd/conversation-svc

# === Target: group-svc ===
FROM builder AS group-svc-builder
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" \
    -o /app/bin/service ./cmd/group-svc

# === Target: contact-svc ===
FROM builder AS contact-svc-builder
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" \
    -o /app/bin/service ./cmd/contact-svc

# === Target: rc-svc ===
FROM builder AS rc-svc-builder
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" \
    -o /app/bin/service ./cmd/rc-svc

# === Target: bot-svc ===
FROM builder AS bot-svc-builder
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" \
    -o /app/bin/service ./cmd/bot-svc

# === Target: file-svc ===
FROM builder AS file-svc-builder
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" \
    -o /app/bin/service ./cmd/file-svc

# === Target: audit-svc ===
FROM builder AS audit-svc-builder
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" \
    -o /app/bin/service ./cmd/audit-svc

# === Target: card-svc ===
FROM builder AS card-svc-builder
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" \
    -o /app/bin/service ./cmd/card-svc

# === Stage 2: Runtime Base ===
FROM alpine:3.18 AS base

RUN apk --no-cache add ca-certificates tzdata curl

WORKDIR /app
EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=5s --retries=3 \
    CMD curl -sf http://localhost:${SERVICE_PORT:-8080}/health || exit 1

# === Target: api-gateway (runtime) ===
FROM base AS api-gateway
COPY --from=api-gateway-builder /app/bin/service .
COPY configs/api-gateway.yaml ./configs/config.yaml
ENTRYPOINT ["./service"]

# === Target: user-svc (runtime) ===
FROM base AS user-svc
COPY --from=user-svc-builder /app/bin/service .
COPY configs/user-svc.yaml ./configs/config.yaml
ENTRYPOINT ["./service"]

# === Target: message-svc (runtime) ===
FROM base AS message-svc
COPY --from=message-svc-builder /app/bin/service .
COPY configs/message-svc.yaml ./configs/config.yaml
ENTRYPOINT ["./service"]

# === Target: conversation-svc (runtime) ===
FROM base AS conversation-svc
COPY --from=conversation-svc-builder /app/bin/service .
COPY configs/conversation-svc.yaml ./configs/config.yaml
ENTRYPOINT ["./service"]

# === Target: group-svc (runtime) ===
FROM base AS group-svc
COPY --from=group-svc-builder /app/bin/service .
COPY configs/group-svc.yaml ./configs/config.yaml
ENTRYPOINT ["./service"]

# === Target: contact-svc (runtime) ===
FROM base AS contact-svc
COPY --from=contact-svc-builder /app/bin/service .
COPY configs/contact-svc.yaml ./configs/config.yaml
ENTRYPOINT ["./service"]

# === Target: rc-svc (runtime) ===
FROM base AS rc-svc
COPY --from=rc-svc-builder /app/bin/service .
COPY configs/rc-svc.yaml ./configs/config.yaml
ENTRYPOINT ["./service"]

# === Target: bot-svc (runtime) ===
FROM base AS bot-svc
COPY --from=bot-svc-builder /app/bin/service .
COPY configs/bot-svc.yaml ./configs/config.yaml
ENTRYPOINT ["./service"]

# === Target: file-svc (runtime) ===
FROM base AS file-svc
COPY --from=file-svc-builder /app/bin/service .
COPY configs/file-svc.yaml ./configs/config.yaml
ENTRYPOINT ["./service"]

# === Target: audit-svc (runtime) ===
FROM base AS audit-svc
COPY --from=audit-svc-builder /app/bin/service .
COPY configs/audit-svc.yaml ./configs/config.yaml
ENTRYPOINT ["./service"]

# === Target: card-svc (runtime) ===
FROM base AS card-svc
COPY --from=card-svc-builder /app/bin/service .
COPY configs/card-svc.yaml ./configs/config.yaml
ENTRYPOINT ["./service"]
