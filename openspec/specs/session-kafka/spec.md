## Purpose

session-svc 的 Kafka 集成能力，包括消息消费（更新会话数据）和事件生产（多端同步）的基础组件。

## Requirements

### Consumer: 消息事件消费
session-svc SHALL 消费 `message_push` topic 的消息事件，用于更新会话列表的最后消息摘要和未读计数。

#### Scenario: Consumer starts with valid config
- **GIVEN** Kafka broker 可用
- **WHEN** session-svc 启动
- **THEN** consumer 开始监听 `message_push` topic

#### Scenario: Graceful shutdown
- **WHEN** session-svc 收到 SIGTERM 信号
- **THEN** consumer 在关闭超时内完成清理

#### Scenario: Kafka broker unreachable
- **GIVEN** Kafka broker 不可用
- **WHEN** session-svc 启动 consumer
- **THEN** 记录 warning 日志
- **AND** 不阻塞服务 HTTP 请求

### Consumer: 更新会话未读和最后消息摘要

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

### Producer: 会话变更事件生产
session-svc SHALL 发布会话变更事件到 `session_sync` topic，用于多端同步。

#### Scenario: Publish session changed event
- **WHEN** 会话状态变更（置顶、免打扰、已读、删除）
- **THEN** producer 发布事件到 `session_sync` topic
- **AND** 事件包含 session_id、user_id、change_type、timestamp
