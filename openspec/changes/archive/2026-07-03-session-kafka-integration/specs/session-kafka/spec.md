## Purpose

session-svc 的 Kafka 集成能力，包括消息消费和事件生产的基础组件。

## Requirements

### Consumer: match 消息消费

session-svc SHALL 消费 `message_push` topic 的消息事件，用于后续更新会话列表的最后消息摘要和未读计数。

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

### Producer: 会话变更事件生产

session-svc SHALL 发布会话变更事件到 `session_sync` topic，用于多端同步。

#### Scenario: Publish session changed event
- **WHEN** 会话状态变更（置顶、免打扰、已读、删除）
- **THEN** producer 发布事件到 `session_sync` topic
- **AND** 事件包含 session_id、user_id、change_type、timestamp
