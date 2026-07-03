## Context

session-svc 目前是纯 HTTP 服务，没有消息队列集成。message-svc 已经有成熟的 Kafka 集成模式（使用 `github.com/segmentio/kafka-go`），session-svc 将复用相同模式。本阶段只做基础设施搭建，不包含业务消费逻辑。

## Goals / Non-Goals

**Goals:**
- session-svc 增加 Kafka 客户端依赖
- config 支持 Kafka broker 和 topic 配置
- 创建 mq 包：consumer（监听 `message_push`）和 producer（发布会话变更事件）
- 优雅启停 consumer
- 与 message-svc 使用相同的 Kafka 库和模式

**Non-Goals:**
- 不实现消费 `message_push` 后的业务逻辑（更新未读、最后消息摘要等留到下一阶段）
- 不修改其他服务
- 不涉及多端同步逻辑
- 不涉及测试（后续统一补齐）

## Decisions

**1. 使用 `github.com/segmentio/kafka-go` 而非 Sarama/Confluent**
- 与 message-svc 保持一致，复用相同模式和版本
- 纯 Go 实现，无 CGo 依赖，部署简单
- 支持 context 取消，便于优雅关闭

**2. Consumer 使用 `kafka.Reader` 而非 Consumer Group 高级 API**
- `kafka.Reader` 足够满足 session-svc 的消费需求
- 与 message-svc 的 audit-svc consumer 模式一致（参考 `services/audit-svc/internal/mq/consumer.go`）
- 便于后续指定 ConsumerGroupID 做分区消费

**3. Producer 使用 `kafka.Writer`**
- 与 message-svc 的 producer 模式一致
- 支持异步写入，不阻塞主流程

**4. Topic 命名**
- `message_push` — 消费已有 topic，监听消息发送事件
- `session_sync` — 新 topic，发布会话变更事件供多端同步

## Risks / Trade-offs

- **[单点故障]** Kafka 不可用时 session-svc 无法消费消息事件 → 消费失败 log warning，不阻塞服务主流程（与 message-svc 的 fail-open 模式一致）
- **[版本兼容]** `segmentio/kafka-go` 版本需要与 message-svc 使用的版本对齐，避免二进制不一致 → 直接复用相同 `v0.4.42`
- **[配置膨胀]** 需要额外配置项管理 → 统一放在 config.yaml，提供合理的默认值
