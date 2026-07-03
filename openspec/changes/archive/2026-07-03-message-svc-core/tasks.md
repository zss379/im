## 1. 项目结构与基础框架

- [x] 1.1 Create `go.mod` with dependencies (gin, mongo-driver, kafka-go, redis, prometheus, zerolog)
- [x] 1.2 Create `config.yaml` with MongoDB/Redis/Kafka/Server config
- [x] 1.3 Create `internal/config/config.go` with Config struct and Load/Default functions
- [x] 1.4 Create `cmd/main.go` with wiring: config → MongoDB → Redis → Kafka → repos → services → HTTP server → graceful shutdown
- [x] 1.5 Create `Makefile` and `Dockerfile` for build/deploy

## 2. 数据模型与存储层

- [x] 2.1 Create `internal/model/message.go` — Message struct, MessageContent types, SendMessageReq, PullMessagesResp, SearchReq
- [x] 2.2 Create `internal/repo/message_repo.go` — MongoDB Insert/FindByID/FindByConversation/UpdateStatus/Search with cursor pagination
- [x] 2.3 Create `internal/repo/message_cache.go` — Redis dedup (client_msg_id SETNX), cursor cache, already-read set

## 3. HTTP Handler — 消息发送

- [x] 3.1 Implement `POST /messages` — full send flow: parse request → dedup check → pre-send validation (rc-svc) → MongoDB persist → Kafka push → response
- [x] 3.2 Integrate bot_trigger Kafka publish (when AtUserList non-empty + not bot sender)
- [x] 3.3 Add `GET /messages` — cursor-based pagination with MongoDB query

## 4. HTTP Handler — 消息操作

- [x] 4.1 Implement `POST /messages/{msg_id}/recall` — check permission, check 2min window, update status, publish Kafka event
- [x] 4.2 Implement `POST /messages/forward` — individual and merge forward modes
- [x] 4.3 Implement `GET /messages/search` — keyword search with filters (conversation, sender, msg_type, time range)

## 5. 已读回执

- [x] 5.1 Implement `PUT /conversations/{conversation_id}/read` — mark messages as read in single chat
- [x] 5.2 Implement `POST /messages/{msg_id}/read-receipt` — get read status for single chat
- [x] 5.3 Implement `GET /messages/{msg_id}/read-status` — query read details

## 6. SSE 与 Kafka

- [x] 6.1 Implement `POST /messages/sse` — accept streaming tokens from bot-svc
- [x] 6.2 Create `internal/mq/producer.go` — Kafka producer for message_push topic (message:new, message:recalled, message:status)
- [x] 6.3 Implement Kafka message type models and serialization

## 7. 中间件与工具

- [x] 7.1 Implement auth middleware (Bearer Token validation)
- [x] 7.2 Implement tenant context middleware (X-Tenant-ID header)
- [x] 7.3 Implement response helper (unified JSON response format with error codes)
- [x] 7.4 Add Prometheus metrics (message count, latency, error rate)
- [x] 7.5 Add structured logging with zerolog

## 8. 测试

- [x] 8.1 Unit tests: message model serialization/deserialization
- [x] 8.2 Unit tests: message repo MongoDB operations (with mongodb in-memory test container)
- [x] 8.3 Unit tests: message cache Redis operations (with miniredis)
- [x] 8.4 Unit tests: send message handler with mock dependencies
- [x] 8.5 Unit tests: recall validation logic
- [x] 8.6 Unit tests: cursor encoding/decoding
