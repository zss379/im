## 1. message-svc: config & HTTP client

- [x] 1.1 Add RCSvcConfig to message-svc config with address and timeout fields
- [x] 1.2 Create HTTP client wrapper in message-svc for rc-svc CheckChain API
- [x] 1.3 Wire the RC client into service constructor and dependency injection

## 2. message-svc: preflight check in SendMessage

- [x] 2.1 Add preflightCheck method to MessageService that calls rc-svc CheckChain
- [x] 2.2 Insert preflight check at top of SendMessage (before dedup, after request parsing)
- [x] 2.3 Return HTTP 403 with blocked_reason when sensitive word hits block strategy
- [x] 2.4 Return HTTP 429 with retry_after when rate limit exceeded
- [x] 2.5 Add fail-open fallback: log warning and continue on rc-svc timeout/error

## 3. message-svc: blocked-message audit event

- [x] 3.1 Add blocked-message Kafka topic to message-svc config
- [x] 3.2 Add PublishBlockedMessage method to Kafka producer
- [x] 3.3 Publish blocked-message event from preflightCheck when message is blocked

## 4. audit-svc: blocked-message consumer

- [x] 4.1 Add Kafka consumer in audit-svc for blocked-message topic
- [x] 4.2 Store blocked-message records in msg_audit_logs table
- [x] 4.3 Wire Kafka consumer into audit-svc main.go startup

## 5. Config & deployment

- [x] 5.1 Update message-svc config.yaml with rc-svc address (rc-svc:8088)
- [x] 5.2 Update deploy/docker/config/message-svc.yaml with rc-svc address
- [x] 5.3 Verify gateway-svc route for rc-svc is present in config

## 6. Tests

- [x] 6.1 Unit test: preflight passes → message continues
- [x] 6.2 Unit test: sensitive word block → 403, no persist, audit event sent
- [x] 6.3 Unit test: rate limit exceeded → 429
- [x] 6.4 Unit test: rc-svc timeout → fail-open allows message through
- [x] 6.5 Unit test: non-text message skips sensitive check but still rate-limited
