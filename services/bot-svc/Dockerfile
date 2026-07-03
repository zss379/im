FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bot-svc ./cmd/

FROM alpine:3.18
RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /bot-svc /usr/local/bin/bot-svc
COPY config.yaml /etc/bot-svc/config.yaml
EXPOSE 8082
ENTRYPOINT ["bot-svc"]
CMD ["--config", "/etc/bot-svc/config.yaml"]
