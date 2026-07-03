## 1. 数据库与模型

- [x] 1.1 Add `response_mode`, `callback_url`, `ip_whitelist` fields to `bot` table migration
- [x] 1.2 Create `group_bot` table migration
- [x] 1.3 Add bot-related Redis data structures (`bot:user_ids` Set, `bot:{id}` Hash) to cache layer

## 2. 机器人身份管理 (bot-svc)

- [x] 2.1 Implement OpenIM user sync: create deactivate OpenIM user account on bot create/delete
- [x] 2.2 Implement Redis Set `bot:user_ids` maintenance (add on create, remove on delete)
- [x] 2.3 Implement Redis Hash `bot:{bot_id}` write-through cache on bot config update
- [x] 2.4 Add `response_mode` support to bot CRUD API (create/update/toggle)

## 3. @触发消息链路 (message-svc)

- [x] 3.1 Create Kafka topic `bot_trigger` with schema and producer config
- [x] 3.2 Implement bot_trigger event publishing in message-svc after message persistence
- [x] 3.3 Implement bot_trigger consumer in bot-svc with Redis Set intersection (`SINTER bot:user_ids + at_user_list`)
- [x] 3.4 Implement single-chat trigger detection (`conv_type=1` + `SISMEMBER bot:user_ids`)

## 4. Webhook Sync 模式 (bot-svc)

- [x] 4.1 Implement sync webhook invoker with 3s timeout context
- [x] 4.2 Implement retry logic with exponential backoff (max 3 retries)
- [x] 4.3 Implement reply parsing and message-svc call to send reply to conversation
- [x] 4.4 Implement error handling: timeout log, retry exhaustion, invalid response discard

## 5. Webhook Async 模式 (bot-svc)

- [x] 5.1 Implement async webhook invoker (POST → expect 202)
- [x] 5.2 Implement Redis pending state store (`bot:pending:{event_id}` with 30min TTL)
- [x] 5.3 Implement callback receiver endpoint (`POST /internal/bot/callback`)
- [x] 5.4 Implement callback validation (event_id match, not expired)
- [x] 5.5 Implement periodic cleanup task (SCAN expired pending entries)

## 6. Webhook SSE 模式 (bot-svc)

- [x] 6.1 Implement SSE webhook invoker (GET/POST → receive SSE URL)
- [x] 6.2 Implement SSE client connection with token forwarding to message-svc SSE API
- [x] 6.3 Implement SSE connection pool with configurable max connections (default 1000)
- [x] 6.4 Implement idle timeout goroutine (5min no-data → terminate)
- [x] 6.5 Implement reconnection logic (max 3 retries)
- [x] 6.6 Implement SSE stream duration limit (max 30min)

## 7. 机器人回复通道 (bot-svc → message-svc)

- [x] 7.1 Add bot-sender support to `POST /messages` (include `sender_bot_id` in request)
- [x] 7.2 Implement bot rate limiting (10 msg/s per bot, independent of user rate limits)
- [x] 7.3 Ensure bot replies pass through sensitive-word check and permission validation

## 8. 配置与部署

- [x] 8.1 Add bot-svc config struct (webhook timeout, retry, pending TTL, SSE limits)
- [x] 8.2 Add bot-svc Helm chart / docker-compose config with new environment variables
- [x] 8.3 Add Prometheus metrics for bot webhook (invocation count, latency, failure rate)
- [x] 8.4 Add structured logging for bot webhook events

## 9. 测试

- [x] 9.1 Unit tests: bot identity Redis sync
- [x] 9.2 Unit tests: @trigger detection logic
- [x] 9.3 Unit tests: sync webhook timeout and retry
- [x] 9.4 Unit tests: async callback validation and TTL expiry
- [x] 9.5 Integration test: message-svc → Kafka → bot-svc end-to-end
- [x] 9.6 Integration test: external system webhook → bot reply in conversation
