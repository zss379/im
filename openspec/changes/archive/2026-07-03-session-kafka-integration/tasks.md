## 1. Config: 添加 Kafka 配置

- [x] 1.1 在 `internal/config/config.go` 中增加 `KafkaConfig` 结构体（Brokers, TopicMessagePush, TopicSessionSync, ConsumerGroupID）
- [x] 1.2 在 `Config` 结构体中增加 `Kafka KafkaConfig` 字段
- [x] 1.3 在 `DefaultConfig()` 中设置 Kafka 默认值
- [x] 1.4 在 `config.yaml` 中增加 Kafka 配置段
- [x] 1.5 在 `deploy/docker/config/session-svc.yaml` 中增加 Kafka 配置段

## 2. MQ 包: 创建 consumer 骨架

- [x] 2.1 创建 `internal/mq/consumer.go`，定义 `Consumer` 结构体（基于 audit-svc 的 consumer 模式）
- [x] 2.2 实现 `NewConsumer(brokers, topic, groupID)` 构造函数
- [x] 2.3 实现 `Start(ctx)` 方法启动消费循环（goroutine），`message_push` topic
- [x] 2.4 消费到消息后暂时只记录 debug 日志（业务逻辑下阶段实现）
- [x] 2.5 实现 `Stop()` 方法关闭 reader

## 3. MQ 包: 创建 producer 骨架

- [x] 3.1 创建 `internal/mq/producer.go`，定义 `Producer` 结构体
- [x] 3.2 实现 `NewProducer(brokers, topic)` 构造函数（`session_sync` topic）
- [x] 3.3 实现 `PublishSessionEvent(ctx, event)` 方法（发布事件到 `session_sync`）
- [x] 3.4 实现 `Close()` 方法关闭 writer
- [x] 3.5 在 `internal/mq/producer.go` 中定义 `SessionSyncEvent` 结构体

## 4. Main: 启动 consumer

- [x] 4.1 在 `cmd/main.go` 中创建 Kafka consumer 实例（consumer 和 producer）
- [x] 4.2 在服务启动后调用 `consumer.Start(ctx)`
- [x] 4.3 在优雅关闭流程中调用 `consumer.Stop()`
- [x] 4.4 确保 consumer 使用带超时的 context（继承 shutdown context）

## 5. Dependencies

- [x] 5.1 在 `go.mod` 中添加 `github.com/segmentio/kafka-go v0.4.42`
- [x] 5.2 运行 `go mod tidy` 更新依赖
