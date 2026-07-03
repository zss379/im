## Context

message-svc 的 SendMessage 流程是：去重 → 落库 MongoDB → Kafka 推送。rc-svc 已经实现完整的 CheckChain（敏感词 DFA + 频控 + 文件限制），但没有任何服务调用它。PRD 要求消息发送前必须走风控校验（IM-MOD-003），拦截记录须写入审计（IM-INT-011）。

message-svc 是纯 Go HTTP 服务，rc-svc 也是独立 HTTP 服务。服务间无 HTTP/gRPC 直连调用，仅通过 Kafka 异步通信。敏感词检查需要同步拦截（"整体校验耗时 ≤ 100ms"），因此消息发送前必须是同步调用。

## Goals / Non-Goals

**Goals:**
- message-svc 在持久化前同步调用 rc-svc 做敏感词 + 频控检查
- 检查未通过时消息不落库，返回 403 给客户端
- 拦截记录异步写入 audit-svc
- 通过 gateway 路由转发，避免服务间硬编码地址

**Non-Goals:**
- 不在本次修改 rc-svc 的 CheckChain 逻辑（已完备）
- 不改变 audit-svc 现有写入端点
- 不改动 gateway-svc 的路由配置（rc-svc 路由已存在）

## Decisions

| 决策 | 选项 | 选择理由 |
|---|---|---|
| 通信方式 | 直连 rc-svc vs 通过 gateway | **直连 rc-svc:8088**：发送前检查是高频内网调用，绕开 gateway 减少一跳延迟，配置简单 |
| 检查接口 | 单独调用 vs CheckChain 一口调用 | **CheckChain**：rc-svc 已提供批量检查接口，一次 HTTP POST 完成敏感词 + 频控，减少网络开销 |
| 调用时机 | handler 层 vs service 层 | **service 层**：handler 只做参数解析和响应，检查逻辑属于业务编排，放在 service.SendMessage 开头 |
| 拦截写入 | HTTP 同步 vs Kafka 异步 | **Kafka 异步**：拦截不阻塞发送主流程，符合 PRD "异步写入不阻塞发消息主流程" |
| HTTP 客户端 | 标准 net/http vs 封装 | **标准 net/http**：仅一个调用端点，无需引入 gRPC 或 thrift |

## Risks / Trade-offs

| 风险 | 缓解措施 |
|---|---|
| rc-svc 宕机导致消息无法发送 | 设超时（2s），超时/失败时放行（fail-open），记录告警日志 |
| DFA 引擎内存占用随词库增长 | rc-svc 已实现增量刷新，词库上限受业务控制 |
| 增加发送延迟（预期 <5ms 内网调用） | 与 rc-svc 同 Docker 网络，直连 + 短超时，影响可忽略 |
