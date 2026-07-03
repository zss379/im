## Why

第一阶段给 session-svc 加上了 Kafka 基础设施（consumer + producer），但 consumer 目前只打日志没有业务逻辑。消息发送后，会话列表的未读计数和最后消息摘要仍然是断的，用户看不到红点和消息预览。需要消费 `message_push` 事件，更新 MySQL 中的会话记录和 Redis 中的未读缓存。

## What Changes

- 在 Consumer 中注入 `*repo.MySQLRepo` 和 `*repo.Cache`，使其能读写数据库和 Redis
- 新增 repo 方法 `FindByConversation` 按 conversation_id 查询所有关联会话
- `processMessage` 事件处理逻辑：
  - 事件类型 `message:new` → 更新未读计数 + 最后消息摘要
  - 事件类型 `message:recalled` → 标记最后消息为已撤回（暂不实现，保留扩展）
- 更新逻辑：
  - 发送者所在会话：更新 `last_message`、`last_msg_type`、`last_sender_id`、`last_message_at`，不增加未读
  - 其他参与者会话：同样更新最后消息字段，且 `unread_count += 1`
  - 同步更新 Redis (SetUnreadCount)
- 若会话不存在则静默跳过（会话由客户端首次进入时创建）

## Capabilities

### New Capabilities
- 无（复用第一阶段 `session-kafka` 能力）

### Modified Capabilities
- 无

## Impact

- `services/session-svc/internal/mq/consumer.go` — 注入 repo + cache，实现业务处理
- `services/session-svc/internal/repo/repo.go` — 新增 `FindByConversation` 方法
- `services/session-svc/cmd/main.go` — 创建 consumer 时传入 repo + cache
