## ADDED Requirements

### Requirement: Consumer updates session on message:new event
When consumer receives a `message:new` event from `message_push` topic, it SHALL update the corresponding session records in MySQL and Redis.

#### Scenario: Update sender session
- **GIVEN** a `message:new` event with sender_id and conversation_id
- **WHEN** consumer processes the event
- **THEN** sender's session SHALL have last_message, last_msg_type, last_sender_id, last_message_at updated
- **AND** sender's session unread_count SHALL NOT be incremented

#### Scenario: Update participant sessions
- **GIVEN** a `message:new` event for a group conversation
- **WHEN** consumer processes the event
- **THEN** all non-sender sessions for that conversation SHALL have unread_count incremented by 1
- **AND** their last_message, last_msg_type, last_sender_id, last_message_at SHALL be updated

#### Scenario: Redis cache synced
- **WHEN** consumer updates unread_count in MySQL
- **THEN** Redis cache SHALL also be updated with the new unread count

#### Scenario: Session not found
- **GIVEN** no session exists for the conversation_id and user_id
- **WHEN** consumer processes the event
- **THEN** consumer SHALL silently skip (no error, no auto-create)
