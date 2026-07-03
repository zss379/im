## 1. Repo: 新增 FindByConversation 方法

- [x] 1.1 在 `internal/repo/repo.go` 中新增 `FindByConversation(ctx, conversationID) ([]Session, error)` 方法
- [x] 1.2 SQL 按 `conversation_id` 查询所有未删除的会话（`is_deleted = false`）

## 2. Consumer: 注入 repo + cache

- [x] 2.1 修改 `Consumer` 结构体，增加 `repo *repo.MySQLRepo` 和 `cache *repo.Cache` 字段
- [x] 2.2 修改 `NewConsumer` 签名，增加 repo 和 cache 参数
- [x] 2.3 实现 `handleMessageNew(event)` 业务逻辑：按 `conversation_id` 查询所有会话
- [x] 2.4 对每个非发送者会话执行 `UPDATE unread_count = unread_count + 1, last_message = ..., last_message_at = ...`
- [x] 2.5 发送者会话只更新最后消息字段，不增加未读
- [x] 2.6 同步更新 Redis 未读缓存（SetUnreadCount）

## 3. Main: 更新 consumer 初始化

- [x] 3.1 在 `cmd/main.go` 中创建 consumer 时传入 `mysqlRepo` 和 `cache`
