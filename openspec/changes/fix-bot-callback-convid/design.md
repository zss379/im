## Context

`AsyncWebhookService.Invoke()` 接收 `WebhookPayload`，其中 `Trigger.Conversation` 包含完整的会话上下文。但存入 `PendingState` 时只保存了 BotID、EventID、MsgID，丢失了会话信息。导致 `HandleCallback` 中只能用 `bot.BotName` 作为占位符。

## Decisions

| Decision | Choice | Rationale |
|---|---|---|
| 存在 PendingState 而非重建 | ConvID/ConvType/GroupID 来自触发事件，回调时已不可回溯 | 无状态回调只能依赖 Redis 保存的上下文 |
| GroupID 使用 `*int64` | 群聊时需要，单聊为 nil | 与 `ConversationContext` 类型一致 |

## Changes

1. `PendingState` 增加 `ConvID string`, `ConvType int8`, `GroupID *int64`
2. `webhook_async.go` `Invoke()` — 从 `payload.Trigger.Conversation` 赋值到 `PendingState`
3. `callback_handler.go` `HandleCallback()` — 用 `state.ConvID`/`state.ConvType`/`state.GroupID` 替代 `bot.BotName`
