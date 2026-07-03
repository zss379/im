# Bot System — Spec Delta

## ADDED Requirements

### Requirement: Bot identity as OpenIM user
Every bot SHALL have a corresponding OpenIM user account for @mention support. The bot's `openim_user_id` MUST be stored in the `bot` table and synced to OpenIM on creation. The system SHALL maintain a Redis Set `bot:user_ids` containing all bot OpenIM user IDs for O(1) @mention detection.

#### Scenario: Bot created with OpenIM user
- **WHEN** a new custom bot is created
- **THEN** the system creates a corresponding OpenIM user account
- **THEN** the bot's `openim_user_id` is stored in the MySQL `bot` table
- **THEN** the bot's user ID is added to Redis Set `bot:user_ids`

#### Scenario: Bot deleted removes OpenIM user
- **WHEN** a custom bot is deleted
- **THEN** the corresponding OpenIM user account is deactivated or removed
- **THEN** the bot's user ID is removed from Redis Set `bot:user_ids`

### Requirement: @trigger message detection
The message-svc SHALL publish a `bot_trigger` event to Kafka when a message contains @mentions that match bot user IDs. The detection SHALL use Redis Set intersection (`SINTER`) between the message's `at_user_list` and the `bot:user_ids` Set. This SHALL be an asynchronous, non-blocking operation performed after message persistence.

#### Scenario: Message with @bot triggers webhook
- **WHEN** a user sends a message in a group with `at_user_list` containing a bot ID
- **THEN** message-svc persists the message to MongoDB
- **THEN** message-svc publishes a `bot_trigger` event to Kafka with the message context and matched bot IDs
- **THEN** bot-svc consumes the event and invokes the external webhook

#### Scenario: Message without @bot does not trigger
- **WHEN** a user sends a message in a group with `at_user_list` containing no bot IDs
- **THEN** message-svc does NOT publish a `bot_trigger` event
- **THEN** no webhook is invoked

#### Scenario: Single chat with bot triggers automatically
- **WHEN** a user sends a message in a single chat where the recipient is a bot
- **THEN** bot-svc detects via `conv_type=1` + Redis `SISMEMBER`
- **THEN** bot-svc invokes the external webhook without requiring @mention

### Requirement: Webhook sync response mode
When a bot is configured with `response_mode=sync`, bot-svc SHALL POST the trigger event to the external webhook URL and wait for a synchronous response. The timeout SHALL be 3 seconds. On timeout or failure, bot-svc SHALL retry up to 3 times with exponential backoff ([100ms, 500ms, 1s]). A successful response SHALL contain a `reply` field that bot-svc sends as a message to the original conversation.

#### Scenario: Sync webhook returns reply
- **WHEN** bot-svc invokes a sync webhook
- **AND** the external system responds within 3s with a valid `reply` field
- **THEN** bot-svc sends the reply text as a message to the conversation via message-svc

#### Scenario: Sync webhook times out
- **WHEN** bot-svc invokes a sync webhook
- **AND** the external system does not respond within 3s
- **THEN** bot-svc retries up to 3 times with exponential backoff
- **THEN** if all retries fail, the event is discarded and an error is logged

### Requirement: Webhook async response mode
When a bot is configured with `response_mode=async`, bot-svc SHALL POST the trigger event and expect a `202 Accepted` response. The external system SHALL call the provided `callback_url` when processing is complete. bot-svc SHALL store a pending state in Redis with a 30-minute TTL. A periodic cleanup task SHALL scan and expire stale pending events.

#### Scenario: Async webhook completes via callback
- **WHEN** bot-svc invokes an async webhook
- **AND** the external system returns `202 { "status": "accepted", "event_id": "evt_xxx" }`
- **THEN** bot-svc stores `bot:pending:{event_id}` in Redis with 30min TTL
- **THEN** when the external system POSTs to `callback_url`, bot-svc validates the `event_id`
- **THEN** bot-svc sends the reply to the original conversation
- **THEN** bot-svc deletes the pending record

#### Scenario: Async callback never arrives
- **WHEN** the external system returns 202
- **AND** no callback is received within 30 minutes
- **THEN** the periodic cleanup task marks the event as expired
- **THEN** an error is logged

### Requirement: Webhook SSE streaming mode
When a bot is configured with `response_mode=sse`, bot-svc SHALL receive a streaming SSE endpoint URL from the external system and connect to receive streaming tokens. bot-svc SHALL forward each token to message-svc's SSE streaming API in real time. bot-svc SHALL manage an SSE connection pool with configurable limits.

#### Scenario: SSE stream delivers tokens
- **WHEN** bot-svc invokes an SSE-mode webhook
- **AND** the external system returns `{ "status": "streaming", "sse_url": "https://..." }`
- **THEN** bot-svc connects to the SSE URL
- **THEN** for each `event: token` received, bot-svc forwards the token data to message-svc SSE API
- **THEN** on `event: done`, bot-svc closes the stream and sends the final message

#### Scenario: SSE connection broken
- **WHEN** the SSE connection is interrupted
- **THEN** bot-svc attempts to reconnect up to 3 times
- **THEN** if reconnection fails, the stream is terminated
- **THEN** a partial message (if any) is delivered as-is

#### Scenario: SSE idle timeout
- **WHEN** no data is received on the SSE connection for 5 minutes
- **THEN** bot-svc terminates the connection
- **THEN** accumulated tokens are sent as the final message

### Requirement: Bot reply goes through standard message pipeline
All bot-generated replies SHALL be sent through message-svc's standard message pipeline, including sensitive-word filtering, rate limiting, and permission checks. A bot sender identity SHALL be included in the message request so the pipeline can apply bot-specific rate limits (10 msg/s).

#### Scenario: Bot reply passes sensitive-word check
- **WHEN** bot-svc sends a reply via message-svc
- **THEN** the reply passes through the pre-send chain including sensitive-word check
- **THEN** the reply is persisted and delivered like a normal message

### Requirement: Bot trigger rate limiting
Bot-triggered messages SHALL be rate-limited independently from user messages, with a default limit of 10 messages per second per bot (configurable). This SHALL apply to both webhook-triggered replies and bot-initiated messages.

#### Scenario: Bot exceeds rate limit
- **WHEN** a bot sends more than 10 messages in 1 second
- **THEN** the 11th message is rejected with error code 40005 (rate limit exceeded)

## MODIFIED Requirements

### Requirement: Bot configuration includes webhook mode
The bot configuration SHALL include `response_mode`, `callback_url`, and `ip_whitelist` fields in addition to existing fields.

| Field | Type | Description |
|-------|------|-------------|
| webhook_url | VARCHAR(500) | External system webhook URL |
| api_key | VARCHAR(128) | HMAC-SHA256 signing key |
| response_mode | TINYINT | 1=sync, 2=async, 3=sse |
| callback_url | VARCHAR(500) | URL for async callback delivery |
| ip_whitelist | JSON | Optional IP whitelist |

#### Scenario: Create bot with response mode
- **WHEN** a user creates a custom bot with `response_mode=async`
- **THEN** the bot is created with async mode in the database
- **THEN** webhook triggers use async mode

### Requirement: Group bot webhook includes @trigger handling
Group webhook bots SHALL support both directions: external system → IM (active push via `POST /webhook/group/{group_id}/bot/{bot_id}`) and IM → external system (@trigger via Kafka `bot_trigger`). Both directions SHALL use the same bot configuration and webhook URL.

#### Scenario: Group bot receives @trigger
- **WHEN** a user @mentions a group bot in a group chat
- **THEN** the bot is triggered via the same webhook mechanism as custom bots
- **THEN** the webhook payload includes the group context

## REMOVED Requirements

None.
