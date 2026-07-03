## Purpose

消息发送前的风控预检能力。message-svc 在持久化消息前调用 rc-svc 进行敏感词检查、频率控制和文件大小限制检查，根据结果决定拦截或放行。拦截消息异步发送至 audit-svc 记录审计日志。

## Requirements

### Requirement: Message preflight check before persistence
message-svc SHALL call rc-svc CheckChain before persisting any new message to MongoDB.

#### Scenario: Normal message passes check
- **WHEN** message-svc receives a SendMessage request with text content
- **AND** rc-svc CheckChain returns passed=true for all checks
- **THEN** message-svc continues to persist the message

#### Scenario: Sensitive word hit (block strategy)
- **WHEN** rc-svc CheckChain detects a sensitive word with block strategy
- **AND** CheckChain returns passed=false, blocked=true
- **THEN** message-svc SHALL NOT persist the message
- **AND** message-svc SHALL return HTTP 403 with blocked reason
- **AND** message-svc SHALL publish a blocked-message event to audit-svc

#### Scenario: Rate limit exceeded
- **WHEN** rc-svc CheckChain returns passed=false due to rate limit
- **THEN** message-svc SHALL NOT persist the message
- **AND** message-svc SHALL return HTTP 429 with retry-after hint

#### Scenario: rc-svc unavailable (fail-open)
- **WHEN** rc-svc returns HTTP error or connection times out (>2s)
- **THEN** message-svc SHALL log a warning
- **AND** message-svc SHALL allow the message through (fail-open)

#### Scenario: Non-text message (image, file, etc.)
- **WHEN** the message MsgType is not a text type
- **AND** content is empty
- **THEN** message-svc SHALL skip sensitive word check
- **AND** message-svc SHALL still check rate limit (if senderID > 0)

#### Scenario: Blocked message audit
- **WHEN** a message is blocked by CheckChain
- **THEN** message-svc SHALL asynchronously publish the blocked record to audit-svc
- **AND** audit-svc SHALL store the blocked message log
