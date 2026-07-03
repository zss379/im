## Context

当前 message-svc 仅有一个占位的 `internal/handler/message_handler.go`，实现了简化版消息发送和 SSE token 接收端点，缺少完整的消息持久化（MongoDB）、Kafka 事件推送、历史消息拉取、消息撤回/转发/搜索、已读回执等核心能力。与之对齐的 PRD §4.5、API 设计 §4、DB 设计 §4.2 已明确定义消息服务行为。

## Goals / Non-Goals

**Goals:**
- MongoDB 消息持久化（messages 集合，conversation_id hashed 分片）
- 完整消息发送链路：接收 → 风控校验（调用 rc-svc）→ 持久化 → Kafka 推送 → 响应
- 游标翻页历史消息拉取（避免 offset 深度翻页）
- 消息撤回（2 分钟时效，替换为系统提示）
- 消息转发（逐条转发 + 合并转发）
- 消息多条件搜索
- 单聊已读回执
- Kafka 事件推送（message:new, message:recalled, message:status）

**Non-Goals:**
- WebSocket 推送（由 OpenIM 处理）
- 离线消息暂存（由 OpenIM 处理）
- 多端同步（由 OpenIM 处理）
- 敏感词/频控引擎实现（属于 rc-svc）
- 文件上传/解析（属于 file-svc）

## Decisions

### 1. MongoDB 驱动：mongo-go-driver
Go 官方 MongoDB 驱动，社区成熟，支持分片、Change Streams、聚合管道。不使用 ORM 层，直接操作 Document 以获得最大性能和灵活性。

### 2. 消息 ID 策略：msg_id = UUID v4
全局唯一业务消息 ID，由服务端生成返回。`client_msg_id` 由客户端生成用于发送幂等去重（Redis SETNX `idempotent:msg:{client_msg_id}` TTL 1h）。

### 3. 游标翻页：send_time + _id 复合游标
历史消息拉取使用 `{ send_time: -1, _id: -1 }` 复合排序。游标编码为 base64 的 `{last_send_time, last_id}` JSON。相比 offset 翻页，游标在 MongoDB 分片集群下性能恒定。

### 4. Kafka 事件：topic `message_push`
订阅方为 OpenIM 回调处理服务/推送服务。事件类型：
- `message:new` — 新消息到达（含全文）
- `message:recalled` — 消息被撤回通知
- `message:status` — 消息状态变更
- `bot_trigger` — @机器人触发（已有）

### 5. 消息发送前校验：调用 rc-svc 内部接口
在消息持久化之前同步调用 `POST /internal/message/check-before-send`，传敏感词/频控校验。超时或 rc-svc 不可用时默认放行（熔断降级，不阻塞发消息）。

### 6. SSE 流式：token 直接转发
`POST /messages/sse` 接收 bot-svc 的 SSE token，直接通过 WebSocket 推送给客户端。当前简化实现返回成功，后续对接 OpenIM 的流式消息通道。

### 7. 已读回执：仅单聊
群聊不展示已读状态（PRD §4.5.5）。单聊接收方阅读消息后自动触发，发送方可查询是否已读。数据存 Redis Hash `read:{conversation_id}`。

## Risks / Trade-offs

- **[风险] MongoDB 分片集群部署复杂度** → 初期可部署单节点副本集，分片在月活超预期后开启。消息集合的 conversation_id hashed 分片可以在后期在线启用。
- **[风险] rc-svc 未就绪时消息发送阻塞** → 熔断降级方案：rc-svc 调用设置 100ms 超时，失败默认放行，消息记录标记 `check_skipped=true`。
- **[风险] Kafka 推送滞后导致多端同步延迟** → 推送是异步的，不阻塞主流程。监控 Kafka 生产延迟，设置 P99 < 200ms 告警。
- **[取舍] 不使用 OpenIM SDK 发送消息** → 消息持久化到 MongoDB 后，通过 Kafka 通知 OpenIM 回调服务，由回调服务调用 OpenIM API 推送。避免对 OpenIM 的直接依赖。
