---
title: 机器人系统能力规格
status: draft
version: 0.2
last_updated: 2026-07-03
---

# 机器人系统能力规格

## 1. 概述

本规格定义数莲 PaaS 平台 IM 机器人系统的能力范围、交互模型、接口约束及非功能要求。机器人系统是 IM 与外部业务系统/第三方服务的集成桥梁。

## 2. 机器人类型

### 2.1 系统机器人（System Bot）

| 属性 | 值 |
|------|-----|
| 可删除 | 否 |
| 可停用 | 是（管理员后台控制） |
| 消息来源 | bot-svc 内部逻辑触发 |
| 外部依赖 | 无 |
| 数量限制 | 固定 3 个 |

**内置列表：**
| 机器人 | 用途 |
|--------|------|
| 消息通知助手 | 系统公告、版本通知 |
| 流程助手 | 审批、流程超时提醒 |
| 文件助手 | 文件上传下载通知 |

**触发机制：** 业务服务通过内部 API 调用 bot-svc，由 bot-svc 调用 message-svc 发送消息。

### 2.2 用户自定义机器人（Custom Bot）

| 属性 | 值 |
|------|-----|
| 可删除 | 是 |
| 可启停 | 是 |
| 消息来源 | 外部系统 POST Webhook（主动推送），用户 @触发（被动触发） |
| 外部依赖 | 外部 Webhook URL |
| 数量限制 | 20 个/租户 |

### 2.3 群 Webhook 机器人（Group Bot）

| 属性 | 值 |
|------|-----|
| 归属 | 具体群聊 |
| 消息来源 | 外部系统 POST Group Webhook（主动推送），群成员 @触发（被动触发） |
| 数量限制 | 5 个/群 |
| 触发方式 | 支持双向：外部系统 → IM（主动推送）和 IM → 外部系统（@触发）|

## 3. 机器人 = OpenIM 用户

### 3.1 身份模型

每个机器人在 OpenIM 中对应一个**真实用户账号**，拥有独立的 `user_id`：

```
机器人记录 (bot-svc MySQL)          OpenIM 用户
┌──────────────────────┐          ┌──────────────────┐
│ bot_id: 4001         │          │ user_id: bot_4001 │
│ openim_user_id: 4001 │◀────────▶│ display_name:     │
│ display_name: 告警    │          │   告警机器人       │
│ avatar_url: ...      │          │ avatar: ...       │
│ webhook_url: ...     │          │ user_type: bot    │
└──────────────────────┘          └──────────────────┘
```

### 3.2 为什么

| 原因 | 说明 |
|------|------|
| @mention 支持 | 客户端自动补全 @机器人名，`at_user_list` 携带机器人 ID |
| 群成员可见 | 机器人在群成员列表中可见，可被添加/移除 |
| 权限模型统一 | 共享群 RBAC 体系，可被禁言、可设置免打扰 |
| 消息同步 | 机器人"发送"的消息自动走 OpenIM 消息通道 |

### 3.3 生命周期同步

- 创建机器人时，系统同步创建 OpenIM 用户账号，存储 `openim_user_id` 到 `bot` 表，并将用户 ID 添加至 Redis Set `bot:user_ids`
- 删除机器人时，系统停用对应 OpenIM 用户账号，并从 `bot:user_ids` 移除
- Redis `bot:user_ids` Set 用于 O(1) @mention 检测

## 4. 交互模型

### 4.1 外部系统 → IM（主动推送）

```
外部系统 ──POST Webhook──→ bot-svc ──→ message-svc ──→ OpenIM ──WSS──→ 用户
```

### 4.2 IM → 外部系统（@触发）

```
用户 @机器人 ──WSS──→ OpenIM ──callback──→ message-svc ──Kafka bot_trigger──→ bot-svc ──POST Webhook──→ 外部系统
```

**触发检测：**
- 群聊：message-svc 在消息持久化后，检查 `at_user_list` 是否包含机器人 ID（通过 Redis Set `bot:user_ids` 的 `SINTER` 检测），若匹配则发布 `bot_trigger` 事件到 Kafka
- 单聊：bot-svc 通过 `conv_type=1` + Redis `SISMEMBER bot:user_ids` 自动触发，无需 @提及
- 不匹配时：不发布 `bot_trigger` 事件，不调用 Webhook

**事件内容：** `BotTriggerEvent` 包含消息上下文（msg_id、text、at_user_list）、发送者信息（user_id、user_name）、会话上下文（conv_id、conv_type、group_id）、以及匹配的 bot_id 列表。

### 4.3 单聊机器人（无需 @）

用户在单聊会话中直接向机器人发送消息，等同于自动触发。流程同 @触发，但不需解析 at_user_list。bot-svc 通过 `conv_type=1` + `SISMEMBER` 检测接收方是否为机器人。

### 4.4 机器人回复通道

所有机器人回复（无论哪种 Webhook 模式）均通过 message-svc 的标准消息通道发送：
```
bot-svc ──POST /messages──→ message-svc ──敏感词检查──→ 权限校验──→ 频控检查──→ 持久化──→ OpenIM WSS──→ 群成员
```

- 请求中携带 `sender_bot_id` 标识机器人发送者身份
- 经过完整的发送前检查链：敏感词过滤、权限校验、频控检查
- 机器人频控独立于用户频控（10 msg/s，可配置）

## 5. Webhook 协议

### 5.1 签名

`HMAC-SHA256(timestamp + request_body, api_key)`

Header: `X-Webhook-Signature`, `X-Webhook-Timestamp`

### 5.2 安全

- 签名验证（全部模式）
- IP 白名单（可配置）
- 3s 超时，自动重试 3 次（sync 模式）
- 1 小时幂等去重（基于 msg_id）

### 5.3 响应模式（Response Mode）

机器人支持三种 Webhook 响应模式，通过 `response_mode` 字段配置：

#### 5.3.1 Sync（同步模式）

bot-svc POST 触发事件到外部 Webhook URL，同步等待响应。

| 属性 | 值 |
|------|-----|
| 超时 | 3 秒 |
| 重试 | 最多 3 次，指数退避 [100ms, 500ms, 1s] |
| 响应格式 | JSON `{ "reply": "回复文本" }` |
| 超时处理 | 记录日志，丢弃事件 |

**流程：**
1. bot-svc POST 事件到外部系统
2. 等待 3s 内同步响应
3. 若超时，重试（指数退避），最多 3 次
4. 成功后解析 `reply` 字段，通过 message-svc 发送到原会话
5. 全部重试失败后丢弃事件，记录错误日志

#### 5.3.2 Async（异步模式，Callback）

bot-svc POST 触发事件，期望 `202 Accepted` 响应。外部系统处理完成后回调 `callback_url`。

| 属性 | 值 |
|------|-----|
| 响应要求 | HTTP 202 `{ "status": "accepted", "event_id": "evt_xxx" }` |
| 待处理状态 | Redis `bot:pending:{event_id}`，TTL 30 分钟 |
| 回调端点 | `POST /internal/bot/callback` |
| 幂等控制 | 回调完成后删除 pending 记录，重复回调拒绝 |
| 过期清理 | 定时任务 SCAN 过期 pending 记录 |

**流程：**
1. bot-svc POST 事件到外部系统
2. 外部系统返回 202 + event_id
3. bot-svc 存储 pending 状态到 Redis（30min TTL）
4. 外部系统处理完成后 POST callback_url（携带 event_id + reply）
5. bot-svc 验证 event_id 有效且未过期
6. 发送回复到原会话，删除 pending 记录
7. 30 分钟内回调未到达 → 过期清理任务标记错误

**Callback 请求格式：**
```json
{
  "event_id": "evt_xxx",
  "reply": "审批已通过",
  "error": null
}
```

#### 5.3.3 SSE（流式输出模式）

bot-svc 从外部系统获取 SSE 流地址并连接，逐个接收 token 并实时转发到消息通道。

| 属性 | 值 |
|------|-----|
| 连接池上限 | 默认 1000（可配置） |
| 空闲超时 | 5 分钟无数据 → 断开 |
| 最大持续 | 30 分钟 |
| 重连 | 最多 3 次 |
| token 转发 | 实时转发到 message-svc SSE API |

**流程：**
1. bot-svc 调用外部系统，获取 SSE URL
2. 从连接池获取连接槽位
3. 连接 SSE 流，逐 token 接收
4. 实时转发每个 token 到 message-svc SSE API
5. `event: done` → 关闭流，发送最终消息
6. 连接中断 → 重连最多 3 次
7. 5 分钟无数据 → 空闲超时断开，已累积 token 作为最终消息

**SSE 事件格式：**
```
event: token
data: {"token": "你好", "seq": 1}

event: token
data: {"token": "世界", "seq": 2}

event: done
data: {"final": true, "full_text": "你好世界"}
```

### 5.4 机器人配置字段

| 字段 | 类型 | 描述 |
|------|------|------|
| webhook_url | VARCHAR(500) | 外部系统 Webhook URL |
| api_key | VARCHAR(128) | HMAC-SHA256 签名密钥 |
| response_mode | TINYINT | 1=sync, 2=async, 3=sse |
| callback_url | VARCHAR(500) | 异步模式回调 URL |
| ip_whitelist | JSON | 可选 IP 白名单 |

## 6. 非功能约束

| 维度 | 约束 |
|------|------|
| Webhook 超时（sync） | 3s |
| 自动重试 | 最多 3 次 |
| 机器人频控 | 10 msg/s（独立于用户频控） |
| 自定义机器人上限 | 20/租户 |
| 群机器人上限 | 5/群 |
| SSE 最大连接数 | 1000（可配置） |
| SSE 空闲超时 | 5 分钟 |
| SSE 最长持续时间 | 30 分钟 |
| 异步 pending TTL | 30 分钟 |
