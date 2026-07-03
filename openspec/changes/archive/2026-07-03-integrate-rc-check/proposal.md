## Why

消息发送链路目前缺少发送前风控检查。message-svc 的 SendMessage 直接将消息持久化到 MongoDB 后才推送，敏感词、频率限制等风控措施（已在 rc-svc 实现）未被调用，违反 PRD IM-MOD-003 "发送前走敏感词风控校验" 的约束。被拦截的消息也未被写入审计日志（IM-INT-011）。

## What Changes

- message-svc 在消息持久化前增加对 rc-svc 的同步调用，执行敏感词 + 频率限制检查
- 检查未通过时消息不被落库，直接返回拦截错误给客户端
- 拦截记录异步写入 audit-svc
- 为 message-svc 增加 rc-svc HTTP 客户端配置
- gateway-svc 将 rc-svc 的 `/api/v1/rc/sensitive/*` 和 `/api/v1/rc/rate-limit/*` 设为内部可路由路径

## Capabilities

### New Capabilities
- `message-preflight`: 消息发送前风控预检（敏感词 + 频控），集成 message-svc 与 rc-svc

### Modified Capabilities
<!-- 暂无现有 spec 需要变更 -->

## Impact

- **message-svc**: 新增 RCService HTTP client 和配置，SendMessage 流程插入 CheckChain 调用
- **rc-svc**: 无需修改（Check 端点已存在）
- **audit-svc**: 接收拦截事件的写入（通过 Kafka 或 HTTP）
- **gateway**: rc-svc 路由已在配置中，无需变更
