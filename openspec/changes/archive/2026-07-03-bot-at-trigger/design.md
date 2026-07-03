---
title: Bot @Trigger 详细设计
status: draft
version: 0.1
last_updated: 2026-07-03
---

# Bot @Trigger 详细设计

## 1. 完整消息时序

```
用户A (群G)              OpenIM              message-svc              bot-svc              外部系统
  │                       │                    │                       │                    │
  │① WSS: 发送消息        │                    │                       │                    │
  │  @机器人 查服务器      │                    │                       │                    │
  │  at_user_list:[4001]  │                    │                       │                    │
  │──────────────────────>│                    │                       │                    │
  │                       │                    │                       │                    │
  │                       │② msg_callback      │                       │                    │
  │                       │───────────────────>│                       │                    │
  │                       │                    │                       │                    │
  │                       │                    │③ 发送前校验链           │                    │
  │                       │                    │   ┌─ rc-svc (敏感词) ──┐│                    │
  │                       │                    │   ├─ Redis (频控) ─────┤│                    │
  │                       │                    │   └─ group-svc (权限) ─┘│                    │
  │                       │                    │                       │                    │
  │                       │                    │④ 持久化 MongoDB        │                    │
  │                       │                    │                       │                    │
  │                       │                    │⑤ Kafka 投递            │                    │
  │                       │                    │  ├─ msg_push → 群成员   │                    │
  │                       │                    │  └─ bot_trigger ─────────────────────>│       │
  │                       │                    │                       │                    │
  │⑥ WSS: 推送给群成员     │                    │                       │                    │
  │<──────────────────────│                    │                       │                    │
  │                       │                    │                       │                    │
  │                       │                    │                       │⑦ 判断@是否命中机器人 │
  │                       │                    │                       │   Redis SMEMBERS    │
  │                       │                    │                       │   bot:user_ids      │
  │                       │                    │                       │                    │
  │                       │                    │                       │⑧ 查机器人配置       │
  │                       │                    │                       │   Redis HGETALL     │
  │                       │                    │                       │   bot:{bot_id}      │
  │                       │                    │                       │                    │
  │                       │                    │                       │⑨ POST Webhook       │
  │                       │                    │                       │  ┌─────────────────┐│
  │                       │                    │                       │  │ sync: 等3s响应   ││
  │                       │                    │                       │  │ async: 202 +回调用││
  │                       │                    │                       │  │ sse: 建立连接     ││
  │                       │                    │                       │  └─────────────────┘│
  │                       │                    │                       │───────────────────>│
  │                       │                    │                       │                    │
  │                       │                    │                       │<─── 响应 ───────────│
  │                       │                    │                       │                    │
  │                       │                    │⑩ 调用 message-svc     │                    │
  │                       │                    │<───────────────────────│                    │
  │                       │                    │   POST /messages       │                    │
  │                       │                    │   (机器人身份发送回复)   │                    │
  │                       │                    │                       │                    │
  │                       │                    │⑪ 回复走标准发送流程     │                    │
  │                       │                    │   → 敏感词/频控/权限    │                    │
  │                       │                    │   → 持久化             │                    │
  │                       │                    │   → Kafka 推送         │                    │
  │                       │                    │                       │                    │
  │⑫ WSS: 机器人回复       │                    │                       │                    │
  │<──────────────────────│                    │                       │                    │
```

## 2. Kafka Topic 设计

```
topic: msg_push            # 消息推送给接收方（已有）
topic: msg_audit           # 消息审计（已有）
topic: bot_trigger         # 新增：@机器人触发事件

bot_trigger 消息格式:
{
  "event_id": "evt_xxx",
  "event_type": "message.mention",
  "timestamp": 1720080000000,
  
  "message": {
    "msg_id": "msg_xxx",
    "client_msg_id": "cli_xxx",
    "send_time": 1720080000000,
    "text": "查一下服务器状态",
    "at_user_ids": [10001, 4001],
    "msg_type": 1
  },
  
  "sender": {
    "user_id": 10001,
    "user_name": "张三"
  },
  
  "conversation": {
    "conv_id": "sg_20001",
    "conv_type": 2,
    "group_id": 20001,
    "group_name": "运维通知群"
  },
  
  "bot_ids": [4001, 4002]   # 命中的机器人列表
}
```

## 3. 机器人身份识别

### 3.1 Redis 数据结构

```
# 集合：所有机器人的 OpenIM user_id（用于 O(1) 判断）
Key: bot:user_ids
Type: Set
Members: [4001, 4002, 4003, ...]

# Hash：机器人详细配置
Key: bot:{bot_id}
Type: Hash
Fields:
  - openim_user_id: 4001
  - bot_type: 1 | 2
  - webhook_url: https://...
  - api_key: sk_xxx
  - response_mode: sync | async | sse
  - callback_url: https://im-api.shulian.com/v1/webhook/bot/{bot_id}/callback
  - ip_whitelist: ["1.2.3.4/32"]
  - status: 1
```

### 3.2 判断逻辑

```
on bot_trigger(message):
    # 1. 取消息中 @的用户列表
    at_ids = message.at_user_ids
    
    # 2. 查 Redis 集合，找出哪些 @用户是机器人
    bot_ids = REDIS.sinter("bot:user_ids", at_ids)
    
    if bot_ids is empty:
        return  # 没有 @机器人
    
    # 3. 对每个机器人并行触发
    for bot_id in bot_ids:
        config = REDIS.hgetall("bot:{bot_id}")
        if config.status == 0:
            continue  # 机器人已停用
        go call_webhook(config, message)
```

## 4. Webhook 三种模式

### 4.1 Sync（同步响应）

适用于：简单查询、状态检查、快速回复

```
bot-svc                              外部系统
  │                                     │
  │ POST /webhook/custom/callback       │
  │ {                                   │
  │   "event": "message.mention",       │
  │   "trigger": {                      │
  │     "text": "查服务器状态",          │
  │     "sender": "张三",               │
  │     "group": "运维通知群"            │
  │   },                                │
  │   "response_mode": "sync"           │
  │ }                                   │
  │────────────────────────────────────>│
  │                                     │
  │<── { "reply": "CPU 45%, 内存 62%" } │
  │            3s 超时                   │
  │                                     │
  │ ← 超时 → 重试 (最多3次)             │
  │ ← 3次失败 → 丢弃 + error log        │
```

#### 4.1.1 bot-svc 实现

```go
func handleSync(webhookURL string, payload WebhookPayload) (*WebhookResponse, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()
    
    var resp WebhookResponse
    for retry := 0; retry < 3; retry++ {
        err := postJSON(ctx, webhookURL, payload, &resp)
        if err == nil {
            return &resp, nil
        }
        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        case <-time.After(backoff(retry)):
            // 指数退避重试
        }
    }
    return nil, ErrWebhookTimeout
}
```

### 4.2 Async（异步回调）

适用于：长任务处理、审批流程、需要外部处理后回复

```
bot-svc                              外部系统                           bot-svc callback
  │                                     │                                  │
  │ POST /webhook/custom/callback       │                                  │
  │ { response_mode: "async" }          │                                  │
  │────────────────────────────────────>│                                  │
  │<── 202 { "status": "accepted" }    │                                  │
  │                                     │                                  │
  │  bot-svc 存储 pending 状态          │                                  │
  │  Redis: bot:pending:{event_id}      │                                  │
  │  TTL: 30 分钟                       │                                  │
  │                                     │                                  │
  │                                     │ 外部系统处理完毕                  │
  │                                     │─────────────────────────────────>│
  │                                     │ POST callback_url                │
  │                                     │ {                                │
  │                                     │   "event_id": "evt_xxx",        │
  │                                     │   "reply": "审批已通过",         │
  │                                     │   "reply_type": "text"           │
  │                                     │ }                                │
  │                                     │                                  │
  │  bot-svc 收到回调                    │                                  │
  │  → 校验 event_id 在 pending 中       │                                  │
  │  → 删除 pending 记录                 │                                  │
  │  → 调用 message-svc 发消息           │                                  │
```

#### 4.2.1 超时与兜底

```
场景: 外部系统接受了请求但从未回调

定时任务 (bot-svc) 每分钟扫描:
  SCAN bot:pending:* 
  → 超过 TTL (30min) 的标记为 expired
  → 记录 error 日志
  → 不自动重发（防止重复处理）
```

### 4.3 SSE（流式输出）

适用于：AI 对话、长时间推理、逐步输出

```
bot-svc                              外部系统 (AI Service)                message-svc
  │                                     │                                  │
  │ POST /webhook/custom/callback       │                                  │
  │ { response_mode: "sse" }            │                                  │
  │────────────────────────────────────>│                                  │
  │<── 200 { "sse_url": "https://..." } │                                  │
  │                                     │                                  │
  │ bot-svc 连接 SSE URL                │                                  │
  │ GET /sse/stream/{session}           │                                  │
  │────────────────────────────────────>│                                  │
  │                                     │                                  │
  │<── event: token                     │                                  │
  │    data: "当前"                    │                                  │
  │<── event: token                     │                                  │
  │    data: "服务器"                    │                                  │
  │<── event: token                     │                                  │
  │    data: "状态"                      │                                  │
  │                                                                        │
  │  逐片调用 message-svc SSE API                                        │
  │ ──────────────────────────────────────────────────────────────>       │
  │  发送 SSE 流式消息片段                                                  │
  │                                                                        │
  │<── event: done                      │                                  │
  │    data: {"session_id": "..."}     │                                  │
  │                                                                        │
  │  调用 message-svc 结束流式消息                                          │
```

#### 4.3.1 bot-svc SSE 连接管理

```
┌────────────────────────────────────────────┐
│  bot-svc SSE Connection Pool                │
│                                             │
│  Per Session:                               │
│  ┌──────────────────────────────┐           │
│  │ goroutine 1: SSE 读取循环    │           │
│  │   → event token → message-svc│           │
│  │   → event error → 断开       │           │
│  │   → event done → 关闭       │           │
│  │                             │           │
│  │ goroutine 2: 超时监控        │           │
│  │   → 5分钟无数据 → kill      │           │
│  └──────────────────────────────┘           │
│                                             │
│  最大连接数: 1000 (可配置)                   │
│  空闲超时: 5 分钟                            │
└────────────────────────────────────────────┘
```

## 5. 数据结构定义

### 5.1 MySQL: bot 表

```sql
CREATE TABLE bot (
    bot_id          BIGINT PRIMARY KEY AUTO_INCREMENT,
    tenant_id       BIGINT NOT NULL,
    bot_type        TINYINT NOT NULL COMMENT '1=系统, 2=自定义',
    openim_user_id  BIGINT NOT NULL COMMENT 'OpenIM 用户ID',
    bot_name        VARCHAR(100) NOT NULL,
    avatar_url      VARCHAR(500),
    description     VARCHAR(500),
    webhook_url     VARCHAR(500) COMMENT '外部系统 Webhook URL',
    api_key         VARCHAR(128) COMMENT '签名密钥',
    response_mode   TINYINT DEFAULT 1 COMMENT '1=sync, 2=async, 3=sse',
    callback_url    VARCHAR(500) COMMENT '异步回调接收 URL',
    ip_whitelist    JSON COMMENT 'IP 白名单',
    status          TINYINT DEFAULT 1 COMMENT '0=停用, 1=启用',
    created_by      BIGINT NOT NULL,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_tenant (tenant_id),
    UNIQUE INDEX idx_openim_user (openim_user_id)
);
```

### 5.2 MySQL: group_bot 表

```sql
CREATE TABLE group_bot (
    id              BIGINT PRIMARY KEY AUTO_INCREMENT,
    group_id        BIGINT NOT NULL,
    bot_id          BIGINT NOT NULL,
    webhook_url     VARCHAR(500) COMMENT '群专属 Webhook URL',
    api_key         VARCHAR(128),
    status          TINYINT DEFAULT 1,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE INDEX idx_group_bot (group_id, bot_id)
);
```

### 5.3 Kafka: bot_trigger 事件

```protobuf
message BotTriggerEvent {
    string event_id = 1;
    string event_type = 2;  // "message.mention"
    int64 timestamp = 3;
    
    MessageContext message = 4;
    SenderInfo sender = 5;
    ConversationContext conversation = 6;
    repeated int64 bot_ids = 7;
}

message MessageContext {
    string msg_id = 1;
    string client_msg_id = 2;
    int64 send_time = 3;
    string text = 4;
    repeated int64 at_user_ids = 5;
}
```

### 5.4 Redis

```
# 机器人 ID 集合（判断 @ 命中）
Key: bot:user_ids
Type: Set

# 机器人详细配置
Key: bot:{bot_id}
Type: Hash

# 异步回调 pending 状态
Key: bot:pending:{event_id}
Type: String
Value: { "bot_id": 4001, "msg_id": "msg_xxx", "expire_at": ... }
TTL: 1800s (30min)
```

## 6. API 接口

### 6.1 新增内部接口

```
POST /internal/bot/trigger    # bot-svc 内部，处理 bot_trigger 事件
POST /internal/bot/callback   # 外部系统异步回调接收
```

### 6.2 已有接口（需适配）

```
POST /messages                # message-svc: 支持以机器人身份发消息
                              # 请求体中增加: sender_bot_id
```

### 6.3 Webhook 请求体标准格式

```json
{
  "event": "message.mention",
  "bot_id": 4001,
  "trigger": {
    "msg_id": "msg_xxx",
    "text": "查服务器状态",
    "sender": {
      "user_id": 10001,
      "user_name": "张三",
      "avatar_url": "https://..."
    },
    "conversation": {
      "conv_id": "sg_20001",
      "conv_type": 2,
      "group_name": "运维通知群",
      "group_id": 20001
    },
    "mentions": [
      {"user_id": 4001, "user_name": "告警机器人"}
    ]
  },
  "response_mode": {
    "type": "sync",
    "timeout_ms": 3000,
    "callback_url": "https://im-api.shulian.com/v1/webhook/bot/4001/callback"
  }
}
```

### 6.4 Webhook 响应格式

```json
// Sync: 直接回复
{
  "reply": "CPU 45%，内存 62%，磁盘 78%",
  "reply_type": "text",
  "reply_raw": {}
}

// Async: 接收确认
{
  "status": "accepted",
  "event_id": "evt_xxx"
}

// SSE: 指定流地址
{
  "status": "streaming",
  "sse_url": "https://external-ai.com/sse/session_xxx",
  "session_id": "session_xxx"
}
```

## 7. 配置项

```yaml
# bot-svc 配置
bot:
  webhook:
    sync_timeout: 3s           # Sync 模式超时
    max_retries: 3              # 最大重试次数
    retry_backoff: [100ms, 500ms, 1s]  # 指数退避
    max_body_size: 1MB          # Webhook 请求体大小限制
  
  async:
    pending_ttl: 30m            # 异步回调 pending 过期时间
    cleanup_interval: 1m        # 清理扫描间隔
  
  sse:
    max_connections: 1000       # SSE 最大连接数
    idle_timeout: 5m            # 空闲超时
    max_stream_duration: 30m    # 单次流最大持续时长
  
  rate_limit:
    bot: 10                     # 机器人每秒消息上限
```

## 8. 单聊场景差异

用户直接给机器人发消息（非群聊），流程略有不同：

```
相同点:
  - 消息 → OpenIM → message-svc → Kafka bot_trigger
  - bot-svc 处理 → Webhook → 回复

差异点:
  - 不需要检查 at_user_list（单聊时收件人本身就是机器人）
  - bot-svc 根据 conv_type=1 + target_user_id 判断
  - @触发 → 以群成员身份发消息
  - 单聊触发 → 以机器人身份回复单聊

命中判断（单聊）:
  on bot_trigger(message):
      if message.conv_type == 1:
          # 单聊：判断 receiver 是否是机器人
          if REDIS.sismember("bot:user_ids", message.receiver_id):
              trigger_bot(message.receiver_id, message)
      else:
          # 群聊：判断 at_user_list
          bot_ids = REDIS.sinter("bot:user_ids", message.at_user_ids)
          ...
```

## 9. 边界与异常

| 场景 | 处理 |
|------|------|
| 外部系统返回无效响应 | 丢弃，记录 error，重试 |
| Async 模式未在 TTL 内回调 | 标记 expired，发送失败通知给触发者 |
| SSE 连接中断 | bot-svc 自动重连（最多 3 次），超过则丢弃 |
| 机器人已停用/删除 | bot-svc 检查 status，跳过触发 |
| 群内机器人已移除 | bot-svc 收到事件 → 检查群-机器人关联 |
| 外部系统返回 429 | 退避重试 |
| 外部系统返回 5xx | 重试 3 次 |
