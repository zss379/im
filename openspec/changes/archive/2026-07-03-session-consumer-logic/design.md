## Context

第一阶段已完成 session-svc 的 Kafka 基础设施（consumer/producer 骨架）。consumer 已接入 `message_push` topic，但事件到达后只打日志。消息发送后，会话的未读计数和最后消息摘要没有更新，导致用户看不到正确的会话列表。

## Goals / Non-Goals

**Goals:**
- Consumer 消费 `message:new` 事件，更新 MySQL 会话记录
- 发送者会话：更新最后消息摘要，不增加未读
- 非发送者会话：更新最后消息摘要，未读计数 +1
- 同步更新 Redis 未读缓存
- 新增 `FindByConversation` repo 方法

**Non-Goals:**
- `message:recalled` 事件处理（预留，本次不做）
- 会话不存在时自动创建（静默跳过，由客户端创建）
- 群组@消息穿透免打扰
- 测试覆盖（后续统一补齐）

## Decisions

**1. Consumer 注入 repo 而非 service**
- 直接操作 DB 和 Redis，避免 service 层的额外校验开销
- 与 audit-svc consumer 模式一致（注入 `*repo.MySQLRepo`）

**2. 使用 `FindByConversation` 批量查询而非逐条处理**
- 一条消息可能涉及多个参与者会话（群聊）
- 单条 SQL 查询所有会话，避免 N+1

**3. 更新策略：乐观更新，失败不阻塞**
- Consumer 处理失败只记录 error 日志，不重试
- 未读计数以消息事件为准，允许短暂不一致

## Risks / Trade-offs

- **[并发写入]** Redis 未读计数可能因并发消息导致覆盖 → 使用 INCR 而非 SET
- **[事件乱序]** Kafka 分区内保证顺序，但跨分区可能乱序 → 目前单 partition，无此问题
- **[MySQL 写入频率]** 大群消息可能导致高频写 → 群聊场景下暂可接受，后续可引入批量合并
