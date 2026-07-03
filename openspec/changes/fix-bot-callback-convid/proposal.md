## Why

bot-svc 的异步 Webhook 回调模式中，`callback_handler.go` 发送回复消息时使用了 `bot.BotName` 作为 `ConvID`，导致回复消息发到错误的会话。原因是 `PendingState` 未保存触发事件的会话上下文。

## What Changes

- `PendingState` 增加会话上下文字段：ConvID、ConvType、GroupID
- `webhook_async.go` `Invoke()` 中保存会话上下文到 pending 状态
- `callback_handler.go` `HandleCallback()` 使用 `PendingState` 中的会话上下文发送回复

## Capabilities

### New Capabilities
- 无

### Modified Capabilities
- 无

## Impact

- `services/bot-svc/internal/repo/bot_cache.go` — PendingState 结构体增加字段
- `services/bot-svc/internal/service/webhook_async.go` — Invoke() 保存会话上下文
- `services/bot-svc/internal/handler/callback_handler.go` — HandleCallback() 使用正确 ConvID
