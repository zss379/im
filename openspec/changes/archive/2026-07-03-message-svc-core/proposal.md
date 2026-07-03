## Why

消息服务（message-svc）是 IM 系统的核心，当前仅有一个简化占位的 `message_handler.go`，缺少完整消息收发、持久化、Kafka 推送、历史拉取、撤回转发等核心能力，需要通过本次变更加以实现。

## What Changes

- 实现 MongoDB 消息持久化层，基于 `messages` 集合（conversation_id hashed 分片）
- 实现完整消息发送链路：敏感词校验 → 频控校验 → MongoDB 持久化 → Kafka 推送事件
- 实现游标翻页历史消息拉取
- 实现消息撤回（2 分钟时效）
- 实现消息转发（逐条转发 / 合并转发）
- 实现单聊已读回执
- 实现消息内关键词搜索
- 实现 Kafka 事件推送（新消息通知、状态更新）
- 实现 SSE 流式消息 token 转发
- 完善 REST API 统一响应格式、认证中间件、全局错误码

## Capabilities

### New Capabilities
- `message-core`: 消息核心收发、持久化、历史拉取、撤回、转发、搜索能力

### Modified Capabilities

无。首次实现 message-svc 核心能力。

## Impact

- **新增服务**: `services/message-svc/` 完整目录结构（cmd、internal、config/etc）
- **新增依赖**: MongoDB 7.0 驱动（`go.mongodb.org/mongo-driver`）
- **新增依赖**: `github.com/gin-gonic/gin`（HTTP 框架，已有）
- **新增依赖**: `github.com/segmentio/kafka-go`（Kafka 生产者，已有）
- **数据层**: MongoDB `messages` 集合，索引 idx_conv_time / idx_sender_time / idx_msg_id
- **API**: 新增 7 个 REST 端点 + 1 个内部端点
- **消息通道**: Kafka topic `message_push` 用于服务端推送通知
