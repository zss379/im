# 接口设计说明书

{ 正式版 }

## 变更记录

| 变更标识 | 章节号及名称 | 变更内容描述 | 变更人 | 变更日期 | 变更前版本号 | 批准人 |
|---------|------------|------------|-------|---------|-----------|-------|
| C | 初始化 | 文档创建初始化 | 邱凯 | 2026/07/02 | | |

> 注：变更标识说明：C——创建，A——增加，M——修改，D——删除

## 1. 概述

### 1.1 文档目的

本文档定义数莲 PaaS 平台 IM 系统的全部接口规范，涵盖 RESTful API、WebSocket 事件、Webhook 回调，作为前后端联调、客户端集成与第三方对接的依据。

### 1.2 接口架构全景

```
┌─────────────────────────────────────────────────────────────────┐
│                        客户端 (App / Web)                         │
├──────────────────────┬──────────────────────────────────────────┤
│   WebSocket 长连接    │        HTTP REST API                    │
│   (实时消息收发)       │        (登录/群管理/文件/查询)             │
└──────────┬───────────┴────────────────┬─────────────────────────┘
           │                            │
           ▼                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                    IM Server (OpenIM v3.8)                       │
│  消息路由 · 持久化 · 多端同步 · 离线推送                         │
├─────────────────────────────────────────────────────────────────┤
│                         业务系统                                  │
│   ┌───────────┐  ┌───────────┐  ┌──────────────┐               │
│   │ Webhook   │  │ Card API  │  │ Audit Query  │               │
│   │ 回调接收   │  │ 卡片推送    │  │ 审计查询/导出 │               │
│   └───────────┘  └───────────┘  └──────────────┘               │
└─────────────────────────────────────────────────────────────────┘
```

### 1.3 接口分类

| 类别 | 传输协议 | 认证方式 | 调用方 | 用途 |
|------|---------|---------|-------|------|
| 客户端 REST API | HTTPS | Bearer Token | App / Web | 登录、会话管理、群操作、文件上传、查询 |
| WebSocket | WSS | Token 握手 | App / Web | 实时消息收发、状态同步、未读更新 |
| 内部 RPC | HTTP(S) | 内部 Token / mTLS | 微服务间 | 用户信息查询、权限校验、风控校验 |
| 外部 Webhook | HTTP(S) | API Key / 签名 | 第三方业务系统 | 机器人消息推送、卡片推送 |
| OpenIM SDK API | gRPC | SDK Token | IM 服务端 | OpenIM 原生接口调用 |

### 1.4 全局约定

#### 1.4.1 基础 URL

| 环境 | REST API | WebSocket |
|------|---------|-----------|
| 开发 | `https://dev-im-api.shulian.com/v1` | `wss://dev-im-ws.shulian.com/v1/ws` |
| 测试 | `https://test-im-api.shulian.com/v1` | `wss://test-im-ws.shulian.com/v1/ws` |
| 生产 | `https://im-api.shulian.com/v1` | `wss://im-ws.shulian.com/v1/ws` |

#### 1.4.2 统一响应格式

**成功响应：**
```json
{
  "code": 0,
  "msg": "success",
  "data": { ... }
}
```

**错误响应：**
```json
{
  "code": 40001,
  "msg": "参数错误",
  "data": null
}
```

#### 1.4.3 全局错误码

| 错误码 | 描述 | 说明 |
|-------|------|------|
| 0 | success | 成功 |
| 40001 | 参数错误 | 请求参数校验失败 |
| 40002 | 认证失败 | Token 无效或过期 |
| 40003 | 无权限 | RBAC 权限不足 |
| 40004 | 资源不存在 | 目标用户/群组/消息不存在 |
| 40005 | 操作频率超限 | 触发消息频控，返回 429 |
| 40006 | 敏感词拦截 | 消息命中敏感词 |
| 40007 | 资源已达上限 | 超过容量限制（置顶5条/群2000人等） |
| 40008 | 操作不允许 | 如禁言用户发言、群主不可退出等 |
| 50001 | 服务内部错误 | 服务器异常，稍后重试 |
| 50002 | 服务暂不可用 | 熔断/降级中 |

#### 1.4.4 认证方式

所有 REST API 请求头需携带：

```
Authorization: Bearer {access_token}
X-Tenant-ID: {tenant_id}
```

Token 通过登录接口获取，有效期为 7 天，过期后调用 `/token/refresh` 续期。

#### 1.4.5 分页约定

| 参数 | 类型 | 说明 |
|------|------|------|
| `page` | Int | 页码（从 1 开始，默认 1） |
| `page_size` | Int | 每页条数（默认 20，最大 100） |

**响应：**
```json
{
  "list": [...],
  "total": 100,
  "page": 1,
  "page_size": 20
}
```

#### 1.4.6 游标翻页（消息场景）

消息拉取使用游标翻页（避免 offset 深度翻页性能问题）：

| 参数 | 类型 | 说明 |
|------|------|------|
| `cursor` | String | 翻页游标（首次传空，后端返回下一页 cursor） |
| `limit` | Int | 每页条数（默认 20，最大 100） |

**响应：**
```json
{
  "list": [...],
  "cursor": "eyJsYXN0X3RpbWUiOjE3...",
  "has_more": true
}
```

---

## 2. 用户认证与登录

### 2.1 登录

```
POST /auth/login
```

**Request：**
```json
{
  "account": "zhangsan",
  "password": "abc123",
  "device_id": "device-uuid-xxx",
  "device_type": "ios | android | web"
}
```

**Response：**
```json
{
  "user_id": 10001,
  "access_token": "eyJhbGciOiJI...",
  "expires_in": 604800,
  "refresh_token": "ref_xxx",
  "user_info": {
    "display_name": "张三",
    "avatar_url": "https://...",
    "department_name": "技术部",
    "position": "高级工程师"
  },
  "tenant_info": {
    "tenant_id": 1,
    "tenant_name": "数莲科技"
  }
}
```

**错误码：** `40001` 参数错误，`40002` 账号或密码错误，`40003` 账号已停用

### 2.2 Token 续期

```
POST /auth/token/refresh
```

**Request：**
```json
{
  "refresh_token": "ref_xxx"
}
```

**Response：**
```json
{
  "access_token": "eyJhbGciOiJI...",
  "expires_in": 604800,
  "refresh_token": "ref_yyy"
}
```

### 2.3 退出登录

```
POST /auth/logout
```

**Header：** `Authorization: Bearer {token}`

**Response：** `{ "code": 0, "msg": "success" }`

**说明：** 服务端清除 Token 缓存，断开 WebSocket 连接。

### 2.4 获取用户信息

```
GET /user/profile
```

**Response：**
```json
{
  "user_id": 10001,
  "account": "zhangsan",
  "display_name": "张三",
  "avatar_url": "https://...",
  "position": "高级工程师",
  "department_id": 101,
  "department_name": "技术部",
  "phone": "138****1234",
  "email": "zhang@...",
  "online_status": 1,
  "custom_status": "会议中"
}
```

### 2.5 批量查询用户信息

```
POST /user/batch-get
```

**Request：**
```json
{
  "user_ids": [10001, 10002, 10003]
}
```

**Response：**
```json
{
  "users": [
    {
      "user_id": 10001,
      "display_name": "张三",
      "avatar_url": "https://...",
      "online_status": 1
    }
  ]
}
```

**说明：** 内部接口（IM-INT-001），用于消息渲染时批量获取发送者信息。

### 2.6 更新个人状态

```
PUT /user/status
```

**Request：**
```json
{
  "online_status": 3,
  "custom_status": "会议中"
}
```

**状态枚举：** 1=在线, 2=离线, 3=忙碌, 4=请勿打扰

### 2.7 获取在线状态（批量）

```
POST /user/online-status
```

**Request：**
```json
{
  "user_ids": [10001, 10002]
}
```

**Response：**
```json
{
  "statuses": {
    "10001": { "status": 1, "last_active_at": "2026-07-02T10:00:00Z" },
    "10002": { "status": 2, "last_active_at": "2026-07-01T18:30:00Z" }
  }
}
```

---

## 3. 会话管理

### 3.1 获取会话列表

```
GET /conversations
```

**Query Parameters：**

| 参数 | 类型 | 必填 | 默认 | 说明 |
|------|------|------|------|------|
| page | Int | N | 1 | 页码 |
| page_size | Int | N | 100 | 每页条数 |
| filter | String | N | - | 筛选：unread/single/group |

**排序规则：** 置顶会话（is_pin=1）按置顶时间倒序在前 → 普通会话按 latest_msg_time 倒序

**Response：**
```json
{
  "list": [
    {
      "conversation_id": 5001,
      "conversation_type": 1,
      "target_id": 10002,
      "display_name": "李四",
      "avatar_url": "https://...",
      "latest_msg": {
        "content": "好的，下午见",
        "msg_type": 1,
        "send_time": "2026-07-02T09:30:00Z"
      },
      "unread_count": 3,
      "is_pin": true,
      "is_mute": false,
      "draft_text": ""
    }
  ],
  "total": 50,
  "page": 1,
  "page_size": 100
}
```

### 3.2 置顶/取消置顶会话

```
PUT /conversations/{conversation_id}/pin
```

**Request：**
```json
{
  "is_pin": true
}
```

**约束：** 置顶数量最多 5 条，超出返回 `40007`。

### 3.3 会话免打扰

```
PUT /conversations/{conversation_id}/mute
```

**Request：**
```json
{
  "is_mute": true
}
```

### 3.4 删除会话

```
DELETE /conversations/{conversation_id}
```

**说明：** 仅删除本地会话列表，云端消息保留，全局搜索仍可检索。

### 3.5 标记已读

```
PUT /conversations/{conversation_id}/read
```

**Response：**
```json
{
  "unread_count": 0
}
```

### 3.6 标记未读

```
PUT /conversations/{conversation_id}/unread
```

### 3.7 一键清除所有未读

```
PUT /conversations/read-all
```

### 3.8 会话详情（单聊专属设置）

```
GET /conversations/{conversation_id}
```

**Response：**
```json
{
  "conversation_id": 5001,
  "conversation_type": 1,
  "target_user": {
    "user_id": 10002,
    "display_name": "李四",
    "avatar_url": "https://..."
  },
  "remark": "客户李总",
  "is_pin": false,
  "is_mute": false,
  "created_at": "2026-06-01T00:00:00Z"
}
```

### 3.9 设置单聊备注

```
PUT /conversations/{conversation_id}/remark
```

**Request：**
```json
{
  "remark": "客户李总"
}
```

---

## 4. 消息管理

### 4.1 发送消息

```
POST /messages
```

**Request：**

```json
{
  "conversation_id": 5001,
  "msg_type": 1,
  "content": {
    "text": "你好，下午开会"
  },
  "client_msg_id": "uuid-xxx"
}
```

**content 按 msg_type 不同结构（见 §4.8 消息类型定义）。**

**Response：**
```json
{
  "msg_id": "msg_xxxx",
  "send_time": "2026-07-02T10:00:00Z",
  "status": 2
}
```

**说明：**
- `client_msg_id` 由客户端生成用于幂等去重
- 发送前服务端自动经过敏感词校验和频控检查
- 被禁言用户/无权限用户返回 `40008`

### 4.2 拉取历史消息（游标翻页）

```
GET /messages
```

**Query Parameters：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| conversation_id | Long | 是 | 会话ID |
| cursor | String | 否 | 翻页游标（首次不传） |
| limit | Int | 否 | 每页条数（默认 20，最大 100） |
| direction | Int | 否 | 0=向前翻（更早消息，默认），1=向后翻（更新消息） |

**Response：**
```json
{
  "list": [
    {
      "msg_id": "msg_xxxx",
      "client_msg_id": "uuid-xxx",
      "sender_id": 10001,
      "sender_name": "张三",
      "sender_avatar": "https://...",
      "msg_type": 1,
      "content": { "text": "你好" },
      "status": 2,
      "send_time": "2026-07-02T10:00:00Z"
    }
  ],
  "cursor": "eyJsYXN0X3RpbWUiOjE3...",
  "has_more": true
}
```

### 4.3 撤回消息

```
POST /messages/{msg_id}/recall
```

**约束：**
- 仅发送者可撤回
- 超过 2 分钟不可撤回，返回 `40008`
- 撤回后替换为系统提示：「张三 撤回了一条消息」

### 4.4 转发消息

```
POST /messages/forward
```

**Request：**
```json
{
  "msg_ids": ["msg_1", "msg_2"],
  "target_type": 2,
  "target_id": 20001,
  "forward_type": 1
}
```

| 参数 | 说明 |
|------|------|
| target_type | 1=单聊, 2=群聊 |
| forward_type | 1=逐条转发（显示原发送者）, 2=合并转发（折叠为一条合并消息） |

### 4.5 搜索消息

```
GET /messages/search
```

**Query Parameters：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| q | String | 是 | 关键词 |
| conversation_id | Long | 否 | 指定会话搜索（为空则搜索全部会话） |
| msg_type | Int | 否 | 按消息类型筛选 |
| sender_id | Long | 否 | 按发送者筛选 |
| start_time | String | 否 | 开始时间 (ISO8601) |
| end_time | String | 否 | 结束时间 (ISO8601) |
| page | Int | 否 | 页码（分页查询） |
| page_size | Int | 否 | 默认 20 |

**Response：**
```json
{
  "list": [
    {
      "msg_id": "msg_xxx",
      "conversation_id": 5001,
      "conversation_name": "张三",
      "sender_id": 10001,
      "sender_name": "张三",
      "msg_type": 1,
      "content": { "text": "下午开会" },
      "send_time": "2026-07-02T10:00:00Z",
      "highlight_ranges": [[0, 2]]
    }
  ],
  "total": 15,
  "page": 1,
  "page_size": 20
}
```

### 4.6 消息已读回执（单聊）

```
POST /messages/{msg_id}/read-receipt
```

**Response：**
```json
{
  "is_read": true,
  "read_at": "2026-07-02T10:05:00Z"
}
```

**说明：** 仅单聊支持。接收方阅读消息后自动触发，发送方可见。

### 4.7 获取消息已读状态（单聊）

```
GET /messages/{msg_id}/read-status
```

**Response：**
```json
{
  "msg_id": "msg_xxx",
  "is_read": true,
  "read_at": "2026-07-02T10:05:00Z"
}
```

### 4.8 消息类型定义

| msg_type | 类型 | content 结构 |
|----------|------|-------------|
| 1 | 文本 | `{ "text": "消息内容", "mentions": { "all": false, "users": [1001] } }` |
| 2 | 图片 | `{ "url": "...", "thumbnail": "...", "width": 1920, "height": 1080, "size": 1048576 }` |
| 3 | 视频 | `{ "url": "...", "thumbnail": "...", "duration": 30.5, "size": 10485760 }` |
| 4 | 文件 | `{ "url": "...", "name": "report.pdf", "size": 2048000, "ext": "pdf", "file_id": 3001 }` |
| 5 | 语音 | `{ "url": "...", "duration": 8.2, "size": 65536 }` |
| 6 | 名片 | `{ "user_id": 10002, "display_name": "李四", "avatar_url": "...", "department_name": "技术部" }` |
| 7 | 卡片 | `{ "template_id": 1, "template_version": 1, "data": { ... } }` |
| 8 | 系统提示 | `{ "text": "xxx 加入了群聊", "system_type": "member_join" }` |
| 9 | SSE(流式) | `{ "stream_id": "stream_xxx", "text": "分段内容", "is_end": false }` |
| 10 | 合并转发 | `{ "title": "聊天记录", "msg_ids": ["msg_1"], "sender_name": "张三" }` |

### 4.9 消息状态枚举

| status | 说明 |
|--------|------|
| 1 | 发送中（客户端本地，服务端不存储） |
| 2 | 已发送 |
| 3 | 发送失败 |
| 4 | 已撤回 |

### 4.10 消息发送约束

| 约束项 | 限制 | 超限处理 |
|--------|------|---------|
| 文本长度 | ≤ 5000 字符 | 输入框限制 |
| 单次图片 | ≤ 9 张 | 前端拦截 |
| 单图大小 | ≤ 10 MB | 上传时校验 |
| 发送频控 | 1s ≤ 5 条 | 返回 429 |
| 撤回时效 | ≤ 2 分钟 | 超时返回 40008 |

---

## 5. 群组管理

### 5.1 创建群聊

```
POST /groups
```

**Request：**
```json
{
  "member_ids": [10002, 10003],
  "group_name": "项目讨论组"
}
```

**说明：**
- 至少选择 2 人
- 创建者自动设为群主
- 不传 group_name 则默认为成员名称拼接
- 单用户创建群组上限 200 个

**Response：**
```json
{
  "group_id": 20001,
  "group_name": "项目讨论组",
  "owner_id": 10001,
  "member_count": 3,
  "created_at": "2026-07-02T10:00:00Z"
}
```

### 5.2 获取群信息

```
GET /groups/{group_id}
```

**Response：**
```json
{
  "group_id": 20001,
  "group_name": "项目讨论组",
  "group_avatar": "https://...",
  "description": "项目协作群",
  "owner_id": 10001,
  "member_count": 20,
  "max_members": 2000,
  "join_type": 1,
  "status": 1,
  "notice": "本周五项目评审，请提交材料",
  "notice_updated_at": "2026-07-01T00:00:00Z",
  "can_member_rename": false,
  "can_at_all": false,
  "can_member_invite": false,
  "role": 1
}
```

**role 说明：** 当前登录用户在群中的角色（1=群主, 2=管理员, 3=成员）。

### 5.3 修改群信息

```
PUT /groups/{group_id}
```

**Request：** （仅群主/管理员可操作）
```json
{
  "group_name": "新项目名",
  "group_avatar": "https://...",
  "description": "新描述",
  "join_type": 2,
  "can_member_rename": true,
  "can_at_all": true,
  "can_member_invite": true
}
```

### 5.4 获取群成员列表

```
GET /groups/{group_id}/members
```

**Query Parameters：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | Int | 否 | 页码 |
| page_size | Int | 否 | 默认 20 |
| role | Int | 否 | 按角色筛选：1=群主, 2=管理员, 3=普通成员 |
| q | String | 否 | 搜索关键词（按昵称/姓名） |

**Response：**
```json
{
  "list": [
    {
      "user_id": 10001,
      "display_name": "张三",
      "avatar_url": "https://...",
      "role": 1,
      "group_alias": "张工",
      "online_status": 1,
      "is_mute": false,
      "joined_at": "2026-06-01T00:00:00Z"
    }
  ],
  "total": 20,
  "page": 1,
  "page_size": 20
}
```

### 5.5 添加群成员

```
POST /groups/{group_id}/members
```

**Request：**
```json
{
  "user_ids": [10004, 10005]
}
```

**约束：**
- 仅群主/管理员可操作
- 单群上限 2000 人，超出返回 `40007`
- 单次添加上限 10 人

### 5.6 移除群成员

```
DELETE /groups/{group_id}/members
```

**Request：**
```json
{
  "user_ids": [10004]
}
```

**约束：** 仅群主/管理员可操作，不可移除群主。单次移除上限 10 人。

### 5.7 设置群角色

```
PUT /groups/{group_id}/members/{user_id}/role
```

**Request：**
```json
{
  "role": 2
}
```

**约束：** 仅群主可操作。管理员上限为成员总数的 10%。

### 5.8 禁言/取消禁言

```
PUT /groups/{group_id}/members/{user_id}/mute
```

**Request：**
```json
{
  "is_mute": true,
  "mute_duration_minutes": 60
}
```

### 5.9 全员禁言

```
PUT /groups/{group_id}/mute-all
```

**Request：**
```json
{
  "is_mute_all": true
}
```

### 5.10 群主转让

```
POST /groups/{group_id}/transfer
```

**Request：**
```json
{
  "new_owner_id": 10002
}
```

**说明：** 仅群主操作。转让后原群主降为普通成员。

### 5.11 解散群聊

```
POST /groups/{group_id}/dismiss
```

**说明：** 仅群主操作。二次确认不可逆。

### 5.12 退出群聊

```
POST /groups/{group_id}/leave
```

**说明：** 群主需先转让才可退出。成员直接退出。

### 5.13 发布/更新群公告

```
PUT /groups/{group_id}/notice
```

**Request：**
```json
{
  "notice": "本周五项目评审，请提交材料"
}
```

### 5.14 获取群公告历史

```
GET /groups/{group_id}/notices
```

### 5.15 入群申请

#### 5.15.1 提交申请

```
POST /groups/{group_id}/join-requests
```

**Request：**
```json
{
  "reason": "项目协作需要"
}
```

**约束：** 24 小时内最多 3 次有效申请。

#### 5.15.2 获取申请列表

```
GET /groups/{group_id}/join-requests
```

**Query Parameters：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| status | Int | 否 | 0=pending, 1=approved, 2=rejected |
| page | Int | 否 | - |
| page_size | Int | 否 | - |

#### 5.15.3 处理申请

```
PUT /groups/{group_id}/join-requests/{request_id}
```

**Request：**
```json
{
  "status": 1,
  "remark": "欢迎加入"
}
```

### 5.16 群聊二维码

```
GET /groups/{group_id}/qrcode
```

### 5.17 刷新群二维码

```
POST /groups/{group_id}/qrcode/refresh
```

---

## 6. 通讯录

### 6.1 获取组织架构树

```
GET /departments/tree
```

**Response：**
```json
{
  "departments": [
    {
      "dept_id": 1,
      "dept_name": "数莲科技",
      "member_count": 100,
      "children": [
        {
          "dept_id": 101,
          "dept_name": "技术部",
          "member_count": 30,
          "children": [
            {
              "dept_id": 1011,
              "dept_name": "前端组",
              "member_count": 10,
              "children": []
            }
          ]
        }
      ]
    }
  ]
}
```

### 6.2 获取部门成员列表

```
GET /departments/{dept_id}/members
```

**Query Parameters：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | Int | 否 | - |
| page_size | Int | 否 | - |
| include_sub | Boolean | 否 | 是否包含子部门成员（默认 false） |

### 6.3 搜索成员

```
GET /users/search
```

**Query Parameters：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| q | String | 是 | 姓名/拼音/手机号/部门模糊匹配 |
| page | Int | 否 | - |
| page_size | Int | 否 | 默认 20 |

**Response：**
```json
{
  "list": [
    {
      "user_id": 10001,
      "display_name": "张三",
      "avatar_url": "https://...",
      "department_name": "技术部",
      "position": "高级工程师",
      "online_status": 1
    }
  ],
  "total": 5,
  "page": 1,
  "page_size": 20
}
```

### 6.4 获取用户名片

```
GET /users/{user_id}/profile
```

**Response：**
```json
{
  "user_id": 10001,
  "display_name": "张三",
  "avatar_url": "https://...",
  "department_name": "技术部",
  "position": "高级工程师",
  "phone": "138****1234",
  "email": "zhang@...",
  "online_status": 1,
  "custom_status": "会议中",
  "is_deleted": false
}
```

### 6.5 我的群组

```
GET /groups/my
```

**Response：**
```json
{
  "managing": [
    { "group_id": 20001, "group_name": "项目A", "member_count": 15, "role": 1 }
  ],
  "joined": [
    { "group_id": 20002, "group_name": "全员群", "member_count": 100, "role": 3 }
  ]
}
```

---

## 7. 文件服务

### 7.1 上传文件

```
POST /files/upload
```

**Request：** `multipart/form-data`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| file | File | 是 | 文件内容 |
| file_type | String | 否 | image/video/audio/document（自动识别） |

**Response：**
```json
{
  "file_id": 3001,
  "file_name": "report.pdf",
  "file_size": 2048000,
  "file_type": "document",
  "mime_type": "application/pdf",
  "url": "https://minio.shulian.com/im/tenant_1/2026/07/02/xxx.pdf",
  "thumbnail_url": null,
  "expire_at": "2026-07-02T11:00:00Z"
}
```

### 7.2 获取文件下载链接

```
GET /files/{file_id}/download
```

**Response：**
```json
{
  "url": "https://minio.shulian.com/im/tenant_1/2026/07/02/xxx.pdf?token=xxx",
  "expire_at": "2026-07-02T11:00:00Z"
}
```

### 7.3 获取文件预览信息

```
GET /files/{file_id}/preview
```

---

## 8. 机器人管理

### 8.1 获取机器人列表

```
GET /bots
```

**Query Parameters：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| bot_type | Int | 否 | 1=系统, 2=自定义 |
| page | Int | 否 | - |
| page_size | Int | 否 | - |

**Response：**
```json
{
  "list": [
    {
      "bot_id": 4001,
      "bot_type": 1,
      "bot_name": "消息通知助手",
      "avatar_url": "https://...",
      "description": "系统公告、版本通知",
      "status": 1,
      "created_at": "2026-06-01T00:00:00Z"
    }
  ],
  "total": 5,
  "page": 1,
  "page_size": 20
}
```

### 8.2 创建自定义机器人

```
POST /bots
```

**Request：**
```json
{
  "bot_name": "告警机器人",
  "avatar_url": "https://...",
  "description": "发送监控告警消息"
}
```

**约束：** 租户最多创建 20 个自定义机器人。

### 8.3 更新机器人配置

```
PUT /bots/{bot_id}
```

### 8.4 启用/停用机器人

```
POST /bots/{bot_id}/toggle
```

**Request：**
```json
{
  "status": 1
}
```

### 8.5 重置 API Key

```
POST /bots/{bot_id}/reset-key
```

### 8.6 删除机器人

```
DELETE /bots/{bot_id}
```

### 8.7 群机器人列表

```
GET /groups/{group_id}/bots
```

### 8.8 群添加机器人

```
POST /groups/{group_id}/bots
```

**Request：**
```json
{
  "bot_id": 4001,
  "webhook_url": "https://..."
}
```

**约束：** 单群最多 5 个机器人。

### 8.9 移除群机器人

```
DELETE /groups/{group_id}/bots/{bot_id}
```

---

## 9. Web 后台管理 API

### 9.1 敏感词管理

#### 9.1.1 获取敏感词列表

```
GET /admin/sensitive-words
```

**Query Parameters：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| q | String | 否 | 关键词搜索 |
| category | String | 否 | 分类筛选 |
| page | Int | 否 | - |
| page_size | Int | 否 | 默认 20 |

#### 9.1.2 添加敏感词

```
POST /admin/sensitive-words
```

**Request：**
```json
{
  "word": "敏感词",
  "strategy": 1,
  "replacement": "***",
  "category": "广告"
}
```

#### 9.1.3 批量导入敏感词

```
POST /admin/sensitive-words/import
```

**Request：** `multipart/form-data`，CSV 文件

**CSV 格式：**
```
word,strategy,replacement,category
敏感词1,1,***,广告
敏感词2,2,,政治
```

#### 9.1.4 导出敏感词

```
GET /admin/sensitive-words/export
```

**Response：** CSV 文件下载

#### 9.1.5 删除敏感词

```
DELETE /admin/sensitive-words/{word_id}
```

### 9.2 频控规则管理

#### 9.2.1 获取频控规则

```
GET /admin/frequency-rules
```

#### 9.2.2 更新频控规则

```
PUT /admin/frequency-rules/{rule_id}
```

**Request：**
```json
{
  "max_count": 10,
  "time_window_seconds": 1,
  "action": 1
}
```

### 9.3 文件上传限制

#### 9.3.1 获取文件限制配置

```
GET /admin/file-limits
```

#### 9.3.2 更新文件限制配置

```
PUT /admin/file-limits/{config_id}
```

### 9.4 审计日志查询

#### 9.4.1 管理员操作日志

```
GET /admin/audit-logs
```

**Query Parameters：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| operator_id | Long | 否 | 操作员ID |
| action_type | String | 否 | 操作类型 |
| start_time | String | 否 | 开始时间 |
| end_time | String | 否 | 结束时间 |
| page | Int | 否 | - |
| page_size | Int | 否 | 默认 20 |

#### 9.4.2 消息审计查询

```
GET /admin/message-audit
```

**Query Parameters：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| sender_id | Long | 否 | 发送者 |
| conversation_id | Long | 否 | 会话ID |
| start_time | String | 否 | 开始时间 |
| end_time | String | 否 | 结束时间 |
| q | String | 否 | 消息关键词 |
| msg_type | Int | 否 | 消息类型 |
| has_sensitive | Boolean | 否 | 是否命中敏感词 |
| page | Int | 否 | - |
| page_size | Int | 否 | 默认 20 |

**Response：**
```json
{
  "list": [
    {
      "audit_id": "audit_xxx",
      "msg_id": "msg_xxx",
      "sender_id": 10001,
      "sender_name": "张三",
      "conversation_id": 5001,
      "msg_type": 1,
      "content_snapshot": "敏感内容...",
      "sensitive_word_hit": ["敏感词A"],
      "sent_at": "2026-07-02T10:00:00Z",
      "sender_ip": "192.168.1.100"
    }
  ],
  "total": 50,
  "page": 1,
  "page_size": 20
}
```

#### 9.4.3 导出审计结果

```
POST /admin/message-audit/export
```

**Request：** 同 9.4.2 查询参数

**Response：** CSV 文件下载（异步生成，返回任务 ID，轮询下载）

---

## 10. 外部接口

### 10.1 第三方登录认证

```
POST /api/v1/auth/third-party
```

**Request：**
```json
{
  "provider": "wechat | dingtalk | custom",
  "code": "auth_code_xxx",
  "api_key": "third_party_api_key"
}
```

### 10.2 Webhook 消息推送（机器人）

外部系统向机器人 Webhook 地址 POST 消息：

```
POST /webhook/bot/{bot_id}
```

**Header：**
```
Content-Type: application/json
X-Webhook-Signature: sha256_signature
X-Webhook-Timestamp: 1720080000000
```

**Request：**
```json
{
  "msg_type": 1,
  "content": {
    "text": "告警：服务器 CPU 超过 90%"
  },
  "conversation_type": 2,
  "target_id": 20001
}
```

**说明：**
- 签名算法：`HMAC-SHA256(timestamp + body, api_key)`
- 响应超时 3 秒，失败自动重试 3 次
- 消息受频控和敏感词校验

### 10.3 群 Webhook 机器人

```
POST /webhook/group/{group_id}/bot/{bot_id}
```

### 10.4 卡片消息推送（外部业务系统）

```
POST /api/v1/cards/push
```

**Header：** `Authorization: Bearer {api_key}`

**Request：**
```json
{
  "receiver_type": 1,
  "receiver_id": 10001,
  "template_id": 1,
  "template_version": 1,
  "data": {
    "title": "请假审批",
    "applicant": "张三",
    "leave_type": "年假",
    "start_date": "2026-07-03",
    "end_date": "2026-07-05",
    "status": "待审批",
    "buttons": [
      { "text": "通过", "action": "approve", "url": "https://..." },
      { "text": "拒绝", "action": "reject", "url": "https://..." }
    ]
  }
}
```

| 参数 | 说明 |
|------|------|
| receiver_type | 1=单聊, 2=群聊 |
| receiver_id | 用户ID 或 群组ID |

### 10.5 组织架构同步（HR 系统）

```
POST /api/v1/sync/department
```

**Header：** `Authorization: Bearer {sync_api_key}`

**Request：**
```json
{
  "action": "upsert | delete",
  "department": {
    "dept_id": 101,
    "parent_dept_id": 1,
    "dept_name": "技术部",
    "sort_order": 1
  }
}
```

---

## 11. WebSocket 事件

### 11.1 连接建立

```
wss://im-ws.shulian.com/v1/ws?token={access_token}&device_id={device_id}
```

**认证流程：**
1. 客户端携带 Token 握手
2. 服务端校验 Token 有效性
3. 连接建立后，服务端发送 `connected` 事件

```json
// 服务端 → 客户端
{ "event": "connected", "data": { "user_id": 10001, "server_time": "..." } }
```

### 11.2 心跳

```
// 客户端 → 服务端（每 30s）
{ "event": "ping" }

// 服务端 → 客户端
{ "event": "pong" }
```

说明：Web 端 2 分钟无心跳视为离线。

### 11.3 事件列表

#### 11.3.1 消息事件

| 事件名 | 方向 | 说明 |
|--------|------|------|
| `message:new` | 服务端→客户端 | 新消息到达 |
| `message:recalled` | 服务端→客户端 | 消息被撤回 |
| `message:status` | 服务端→客户端 | 消息状态更新 |

**message:new**
```json
{
  "event": "message:new",
  "data": {
    "msg_id": "msg_xxx",
    "conversation_id": 5001,
    "sender_id": 10001,
    "sender_name": "张三",
    "msg_type": 1,
    "content": { "text": "你好" },
    "send_time": "2026-07-02T10:00:00Z"
  }
}
```

**message:recalled**
```json
{
  "event": "message:recalled",
  "data": {
    "msg_id": "msg_xxx",
    "conversation_id": 5001,
    "sender_id": 10001
  }
}
```

#### 11.3.2 会话事件

| 事件名 | 方向 | 说明 |
|--------|------|------|
| `conversation:new` | 服务端→客户端 | 新会话创建 |
| `conversation:update` | 服务端→客户端 | 会话更新（置顶/免打扰/备注） |
| `conversation:unread` | 服务端→客户端 | 未读计数更新 |
| `conversation:delete` | 服务端→客户端 | 会话被删除 |

**conversation:unread**
```json
{
  "event": "conversation:unread",
  "data": {
    "conversation_id": 5001,
    "unread_count": 5,
    "total_unread": 12
  }
}
```

#### 11.3.3 群组事件

| 事件名 | 方向 | 说明 |
|--------|------|------|
| `group:member_added` | 服务端→客户端 | 新成员加入 |
| `group:member_removed` | 服务端→客户端 | 成员被移除 |
| `group:info_updated` | 服务端→客户端 | 群信息变更 |
| `group:role_changed` | 服务端→客户端 | 角色变更 |
| `group:transferred` | 服务端→客户端 | 群主转让 |
| `group:dismissed` | 服务端→客户端 | 群解散 |
| `group:mute_changed` | 服务端→客户端 | 禁言状态变更 |

**group:member_added**
```json
{
  "event": "group:member_added",
  "data": {
    "group_id": 20001,
    "new_members": [
      { "user_id": 10004, "display_name": "王五", "role": 3 }
    ],
    "operator_id": 10001
  }
}
```

#### 11.3.4 状态事件

| 事件名 | 方向 | 说明 |
|--------|------|------|
| `user:online_status` | 服务端→客户端 | 用户在线状态变更 |

```json
{
  "event": "user:online_status",
  "data": {
    "user_id": 10002,
    "online_status": 1,
    "custom_status": ""
  }
}
```

#### 11.3.5 输入状态事件

| 事件名 | 方向 | 说明 |
|--------|------|------|
| `typing` | 客户端→服务端 | 正在输入通知 |
| `typing` | 服务端→客户端 | 转发给对方客户端 |

```json
// 客户端 → 服务端
{ "event": "typing", "data": { "conversation_id": 5001 } }

// 服务端 → 客户端（对方）
{ "event": "typing", "data": { "conversation_id": 5001, "user_id": 10002 } }
```

**防抖规则：** 客户端每 3s 最多发送一次 typing 事件。服务端一段时间无新 typing 事件自动清除。

---

## 12. SSE 流式消息

### 12.1 AI 对话流式输出

```
GET /api/v1/sse/chat
```

**Header：**
```
Authorization: Bearer {token}
Accept: text/event-stream
```

**Query Parameters：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| conversation_id | Long | 是 | 会话ID |
| prompt | String | 是 | 用户输入文本 |

**Response（SSE 流）：**

```
event: message
data: {"stream_id": "stream_xxx", "index": 0, "text": "您好，", "is_end": false}

event: message
data: {"stream_id": "stream_xxx", "index": 1, "text": "有什么可以帮助您的？", "is_end": false}

event: message
data: {"stream_id": "stream_xxx", "index": 2, "text": "", "is_end": true}

event: done
data: {"stream_id": "stream_xxx", "msg_id": "msg_stream_xxx", "full_text": "您好，有什么可以帮助您的？"}
```

---

## 13. 内部接口

以下接口为微服务间内部调用，不对外暴露。

### 13.1 消息发送前风控校验

```
POST /internal/message/check-before-send
```

**Request：**
```json
{
  "sender_id": 10001,
  "tenant_id": 1,
  "content": "消息文本",
  "msg_type": 1,
  "target_type": 2,
  "target_id": 20001
}
```

**Response：**
```json
{
  "passed": true,
  "sensitive_hit": false,
  "frequency_limited": false
}
```

**性能要求：** 整体校验耗时 ≤ 100ms。

### 13.2 消息审计日志写入

```
POST /internal/audit/message
```

**说明：** 异步写入，不阻塞消息发送主流程。

### 13.3 批量查询用户信息

```
POST /internal/user/batch-get
```

（同 2.5，内部版本不鉴权，使用内部 Token）

---

## 14. 接口依赖关系汇总

### 14.1 接口-模块依赖矩阵

| API 模块 | 依赖内部接口 | 依赖外部系统 |
|---------|------------|------------|
| 用户认证 | - | - |
| 会话管理 | IM-INT-003（未读同步） | - |
| 消息发送 | IM-INT-004（风控校验），IM-INT-005（文件URL），IM-INT-011（审计写入） | - |
| 消息撤回 | IM-INT-011（审计写入） | - |
| 群管理 | IM-INT-006（发言权限），IM-INT-007（成员列表） | - |
| 通讯录 | - | HR 系统（组织同步） |
| 机器人 | IM-INT-009（消息发送），IM-INT-010（Webhook） | 外部系统回调 |
| 卡片 | IM-INT-012（模板生成） | 外部业务系统推送 |
| 文件 | IM-INT-015（上传回调），IM-INT-016（S3接口） | MinIO/S3 |
| 审计 | IM-INT-014（日志写入） | - |

### 14.2 内部接口时序示例：消息发送

```
客户端                  IM Server               风控服务             消息存储             推送服务
  │                       │                       │                   │                   │
  │── POST /messages ────→│                       │                   │                   │
  │                       │── INTERNAL: ─────────→│                   │                   │
  │                       │   check-before-send   │                   │                   │
  │                       │←────── passed ────────│                   │                   │
  │                       │                       │                   │                   │
  │                       │─── MongoDB: ─────────→│                   │                   │
  │                       │   insert message      │                   │                   │
  │                       │←────── success ───────│                   │                   │
  │                       │                       │                   │                   │
  │                       │── Kafka: ────────────────────────────────→│                   │
  │                       │   push event          │                   │                   │
  │                       │                       │                   │                   │
  │                       │── INTERNAL: ─────────→│                   │                   │
  │                       │   audit write         │                   │                   │
  │←─── { msg_id } ──────│                       │                   │                   │
  │                       │                       │                   │                   │
  │  WebSocket 推送       │                       │                   │                   │
  │←── message:new ──────│                       │                   │                   │
```

---

## 15. 通用枚举定义

### 15.1 会话类型

| 值 | 说明 |
|----|------|
| 1 | 单聊 |
| 2 | 群聊 |

### 15.2 消息类型

| 值 | 说明 |
|----|------|
| 1 | 文本 |
| 2 | 图片 |
| 3 | 视频 |
| 4 | 文件 |
| 5 | 语音 |
| 6 | 名片 |
| 7 | 结构化卡片 |
| 8 | 系统提示 |
| 9 | SSE 流式消息 |
| 10 | 合并转发 |

### 15.3 消息状态

| 值 | 说明 |
|----|------|
| 1 | 发送中（客户端仅本地） |
| 2 | 已发送 |
| 3 | 发送失败 |
| 4 | 已撤回 |

### 15.4 群角色

| 值 | 说明 |
|----|------|
| 1 | 群主 |
| 2 | 管理员 |
| 3 | 普通成员 |

### 15.5 在线状态

| 值 | 说明 |
|----|------|
| 1 | 在线 |
| 2 | 离线 |
| 3 | 忙碌 |
| 4 | 请勿打扰 |

### 15.6 入群申请状态

| 值 | 说明 |
|----|------|
| 0 | 待审批 |
| 1 | 已通过 |
| 2 | 已拒绝 |
| 3 | 已过期 |

### 15.7 敏感词策略

| 值 | 说明 |
|----|------|
| 1 | 替换为 *** |
| 2 | 拦截（消息发送失败） |
| 3 | 仅记录日志（放行） |

### 15.8 机器人类型

| 值 | 说明 |
|----|------|
| 1 | 系统机器人 |
| 2 | 用户自定义机器人 |

---

## 16. 接口安全规范

### 16.1 REST API 安全

| 要求 | 规范 |
|------|------|
| 传输加密 | 全链路 HTTPS/TLS 1.3 |
| 认证方式 | Bearer Token（JWT），每次请求携带 |
| Token 有效期 | 7 天，refresh_token 可续期 |
| 租户隔离 | 所有请求头携带 `X-Tenant-ID` |
| 敏感数据 | 手机号脱敏展示（138****1234） |
| 频率限制 | 全局 100 req/min/user，消息发送额外受频控规则限制 |
| 幂等性 | 消息发送使用 `client_msg_id` 幂等去重 |

### 16.2 Webhook 安全

| 要求 | 规范 |
|------|------|
| 签名验证 | HMAC-SHA256( timestamp + body, api_key ) |
| IP 白名单 | 支持配置可信 IP 列表 |
| 超时处理 | 3s 超时，自动重试 3 次 |
| 幂等 | 基于 `msg_id` 1 小时去重 |

### 16.3 WebSocket 安全

| 要求 | 规范 |
|------|------|
| 连接认证 | 通过 Token 握手建立连接 |
| 心跳保活 | 30s ping/pong，2 分钟无响应判离线 |
| 消息校验 | 服务端校验每条消息的发送权限 |
