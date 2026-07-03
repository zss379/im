# Message Core

## Purpose

The Message Core capability provides the fundamental message sending, retrieval, and management operations for the IM platform. It handles message persistence (MongoDB), real-time event publishing (Kafka), deduplication, message lifecycle (send → recall), and read receipts.

## Requirements

### Requirement: Send message
The system SHALL accept a message send request, perform pre-send validation (sensitive word check, rate limiting), persist the message to MongoDB, and publish a push event to Kafka. The system SHALL return the generated `msg_id` and server timestamp to the sender.

#### Scenario: Send text message successfully
- **WHEN** a user sends a text message to a conversation
- **AND** the content passes sensitive word check and rate limiting
- **THEN** the message is persisted to MongoDB with status=2 (sent)
- **THEN** a `message:new` event is published to Kafka
- **THEN** the response contains `msg_id` and `send_time`

#### Scenario: Send message blocked by sensitive word
- **WHEN** a user sends a message containing a sensitive word
- **AND** the strategy is "拦截" (block)
- **THEN** the message is rejected with error code 40006
- **THEN** the message is NOT persisted to MongoDB

#### Scenario: Send message rate limited
- **WHEN** a user sends more than 5 messages in 1 second
- **THEN** the message is rejected with error code 40005 (429)

#### Scenario: Send message with client_msg_id dedup
- **WHEN** a user resends the same `client_msg_id` within 1 hour
- **THEN** the system returns the existing `msg_id` without creating a duplicate

#### Scenario: Bot sends message
- **WHEN** a bot sends a message via `sender_bot_id`
- **THEN** the message is processed with bot-specific rate limiting (10 msg/s)
- **THEN** bot_trigger Kafka event is NOT published (avoids loop)

### Requirement: Pull history messages
The system SHALL support cursor-based pagination for pulling history messages within a conversation, ordered by send time descending.

#### Scenario: Pull first page
- **WHEN** a user requests messages with an empty cursor
- **THEN** the system returns the latest N messages
- **THEN** the response includes a `cursor` for the next page and `has_more`

#### Scenario: Pull next page
- **WHEN** a user requests messages with a valid cursor from the previous page
- **THEN** the system returns the next N older messages
- **THEN** when no more messages exist, `has_more` is false

### Requirement: Recall message
The system SHALL allow a sender to recall a message within 2 minutes of sending. Recalled messages SHALL be replaced with a system prompt.

#### Scenario: Recall within time limit
- **WHEN** a sender recalls a message within 2 minutes
- **THEN** the message status is updated to 4 (recalled)
- **THEN** a `message:recalled` event is published to Kafka

#### Scenario: Recall after timeout
- **WHEN** a sender tries to recall a message after 2 minutes
- **THEN** the system returns error code 40008

#### Scenario: Non-sender tries to recall
- **WHEN** a user who is not the sender tries to recall a message
- **THEN** the system returns error code 40003 (no permission)

### Requirement: Forward messages
The system SHALL support forwarding messages to other conversations, with two modes: individual forward (showing original sender) and merge forward (folded into a single combined message).

#### Scenario: Individual forward
- **WHEN** a user forwards messages in individual mode
- **THEN** each message is duplicated to the target conversation with original sender info preserved

#### Scenario: Merge forward
- **WHEN** a user forwards messages in merge mode
- **THEN** a single combined message (msg_type=10) is created containing all forwarded messages

### Requirement: Search messages
The system SHALL support keyword search across messages with optional filters for conversation, sender, message type, and time range.

#### Scenario: Search by keyword
- **WHEN** a user searches for a keyword within a conversation
- **THEN** the system returns matching messages with `highlight_ranges`

#### Scenario: Search with filters
- **WHEN** a user searches with filters (sender_id, msg_type, time range)
- **THEN** the query results are filtered accordingly

### Requirement: Read receipt (single chat)
The system SHALL support read receipts for single chat. When the recipient reads a message, the sender SHALL be able to query the read status.

#### Scenario: Mark message as read
- **WHEN** a recipient opens a single chat conversation
- **THEN** all messages in that conversation are marked as read

#### Scenario: Query read status
- **WHEN** a sender queries the read status of a message
- **THEN** the system returns `is_read` and `read_at` for single chat messages

### Requirement: SSE streaming token forwarding
The system SHALL accept streaming tokens from bot-svc and forward them to the client via the real-time channel.

#### Scenario: Receive SSE token
- **WHEN** bot-svc sends a streaming token via POST /messages/sse
- **THEN** the system accepts and acknowledges the token
