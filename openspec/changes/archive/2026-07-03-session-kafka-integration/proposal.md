## Why

session-svc 目前没有 Kafka 集成，导致会话变更（最后消息摘要更新、未读计数变更、置顶/免打扰操作）无法通过事件通知其他服务或同步到多端。PRD IM-INT-003 明确要求"会话最后消息刷新、未读数更新"使用 Kafka 异步推送。当前 session-svc 是个孤岛，必须先补齐消息基础设施才能打通上下游。

## What Changes

- 给 session-svc 添加 Kafka 客户端依赖 (`github.com/segmentio/kafka-go`)
- 在 session-svc config 中增加 Kafka 配置项（brokers、topic 等）
- 创建 Kafka consumer 骨架（监听 `message_push` topic，为后续消费消息事件做准备）
- 创建 Kafka producer 骨架（为后续发布会话变更事件做准备）
- 更新 `go.mod` 和 `go.sum`
- 在 `cmd/main.go` 中启动 consumer

本次不涉及业务逻辑（消费消息后更新未读等），只做基础设施。

## Capabilities

### New Capabilities
- `session-kafka`: session-svc 的 Kafka 消息能力，包括 consumer 和 producer 基础组件

### Modified Capabilities
- `message-core`: 无变更（message-svc 不需要改动）

## Impact

- `services/session-svc/internal/config/config.go` — 增加 Kafka 配置结构体
- `services/session-svc/internal/mq/` — 新建 mq 包，包含 consumer 和 producer
- `services/session-svc/cmd/main.go` — 增加 consumer 启动逻辑
- `services/session-svc/go.mod` — 增加 `github.com/segmentio/kafka-go` 依赖
