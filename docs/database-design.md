# 数据库设计说明书

{ 正式版 }

## 变更记录

| 变更标识 | 章节号及名称 | 变更内容描述 | 变更人 | 变更日期 | 变更前版本号 | 批准人 |
|---------|------------|------------|-------|---------|-----------|-------|
| C | 初始化 | 文档创建初始化 | 邱凯 | 2026/07/02 | | |

> 注：变更标识说明：C——创建，A——增加，M——修改，D——删除

## 1. 概述

### 1.1 文档目的

本文档定义数莲 PaaS 平台 IM 系统的数据库设计方案，包括 MySQL 关系表、MongoDB 文档集合、Redis 缓存结构的详细设计，作为开发实现与 DBA 运维的依据。

### 1.2 设计依据

- PRD `docs/prd.md` 中的功能需求、容量限制、数据描述
- 系统架构：三层分层（Client → IM Server → Business System）
- 数据总量预估：10万并发在线，单聊 5000 TPS，消息永久存储

### 1.3 数据存储分层

| 存储引擎 | 承载数据 | 选型理由 |
|---------|---------|---------|
| MySQL 8.0 | 用户、部门、群组、会话关系、机器人配置、风控规则、文件元数据、卡片模板 | 强事务、关系查询、Schema 严格 |
| MongoDB 7.0 | 消息历史、管理员审计日志、消息审计日志、离线消息 | 高写入吞吐、灵活 Schema、自动分片 |
| Redis 7.0 | 在线状态、未读计数、Token、消息频控、分布式锁 | 内存读写、毫秒级延迟、TTL 自动过期 |
| MinIO/S3 | 图片、视频、文件、语音二进制数据 | 对象存储、CDN 加速、版本控制 |

### 1.4 命名规范

| 对象 | 规范 | 示例 |
|-----|------|------|
| MySQL 数据库 | `im_{tenant_type}` | `im_shared` |
| MySQL 表 | 小写下划线，单数名词 | `user`, `group_member` |
| MySQL 列 | 小写下划线 | `display_name`, `created_at` |
| MySQL 主键 | `{table}_id` | `user_id`, `group_id` |
| MySQL 索引 | `idx_{表名}_{列名}` | `idx_user_tenant` |
| MySQL 唯一约束 | `uk_{表名}_{列名}` | `uk_tenant_account` |
| MongoDB 数据库 | `im_{env}` | `im_prod`, `im_test` |
| MongoDB 集合 | 小写下划线复数 | `messages`, `admin_audit_logs` |
| MongoDB 字段 | 小写下划线（驼峰用于嵌套） | `send_time`, `msg_type` |
| Redis Key | `{业务域}:{子域}:{ID}` | `user:online:10001` |

## 2. ER 图（数据关系概览）

```
┌───────────┐     ┌───────────────┐     ┌──────────────┐
│  tenant   │1──N│     user      │1──1│  user_status  │
└───────────┘     └───────┬───────┘     └──────────────┘
      │                    │                    │
      │              N─────┼──────N             │
      │             │      │      │             │
      │     ┌───────┘      │      └───────┐     │
      │     │              │              │     │
      │  ┌──┴──────┐  ┌────┴─────┐  ┌────┴──┐  │
      │  │department│  │group_info│  │  bot  │  │
      │  └──┬───────┘  └────┬─────┘  └───────┘  │
      │     │               │                    │
      │     │ N          N──┼──N                 │
      │     │        ┌─────┘  └──────┐           │
      │     │        │               │           │
      │  ┌──┴─────────┐  ┌──────────┴───────┐   │
      │  │user_dept   │  │  group_member    │   │
      │  └────────────┘  └──────────────────┘   │
      │                                         │
      │           ┌─────────────────┐           │
      │           │  conversation   │           │
      │           └────────┬────────┘           │
      │                    │                    │
      │           ┌────────┴────────┐           │
      │           │    messages     │           │
      │           │  (MongoDB)     │           │
      │           └─────────────────┘           │
```

## 3. MySQL 表结构

### 3.1 租户体系

#### 3.1.1 tenant — 租户表

| 列名 | 类型 | 长度 | 可空 | 默认值 | 说明 |
|-----|------|------|------|--------|------|
| tenant_id | BIGINT | - | N | AUTO_INCREMENT | 主键 |
| tenant_name | VARCHAR | 100 | N | - | 租户名称 |
| domain | VARCHAR | 200 | Y | NULL | 租户域名 |
| logo_url | VARCHAR | 500 | Y | NULL | 租户Logo |
| status | TINYINT | - | N | 1 | 1=正常, 0=停用 |
| contact_phone | VARCHAR | 20 | Y | NULL | 联系电话 |
| contact_email | VARCHAR | 100 | Y | NULL | 联系邮箱 |
| created_at | DATETIME | - | N | CURRENT_TIMESTAMP | - |
| updated_at | DATETIME | - | N | CURRENT_TIMESTAMP ON UPDATE | - |

**索引：**
- PRIMARY KEY (`tenant_id`)

**说明：** IM 系统采用共享数据库 + tenant_id 行级隔离模式，所有业务表均携带 tenant_id。单表存储所有租户数据。

---

#### 3.1.2 user — 用户基础表

| 列名 | 类型 | 长度 | 可空 | 默认值 | 说明 |
|-----|------|------|------|--------|------|
| user_id | BIGINT | - | N | AUTO_INCREMENT | 主键 |
| tenant_id | BIGINT | - | N | - | 租户ID，FK → tenant |
| account | VARCHAR | 64 | N | - | 登录账号 |
| password_hash | VARCHAR | 256 | N | - | bcrypt 加密密码 |
| phone | VARCHAR | 20 | Y | NULL | 手机号（AES 加密存储） |
| email | VARCHAR | 100 | Y | NULL | 邮箱 |
| display_name | VARCHAR | 64 | N | - | 显示名称 |
| avatar_url | VARCHAR | 500 | Y | NULL | 头像URL |
| position | VARCHAR | 100 | Y | NULL | 岗位 |
| department_id | BIGINT | - | Y | NULL | 主部门ID，FK → department |
| status | TINYINT | - | N | 1 | 1=正常, 2=离职, 3=删除 |
| ext_json | JSON | - | Y | NULL | 扩展字段 |
| created_at | DATETIME | - | N | CURRENT_TIMESTAMP | 创建时间 |
| updated_at | DATETIME | - | N | CURRENT_TIMESTAMP ON UPDATE | 更新时间 |

**索引：**
- PRIMARY KEY (`user_id`)
- UNIQUE KEY `uk_tenant_account` (`tenant_id`, `account`)
- KEY `idx_tenant_status` (`tenant_id`, `status`)
- KEY `idx_tenant_dept` (`tenant_id`, `department_id`)

**容量约束：** 无硬性上限，随租户增长扩展。

---

#### 3.1.3 user_status — 用户状态表

| 列名 | 类型 | 长度 | 可空 | 默认值 | 说明 |
|-----|------|------|------|--------|------|
| user_id | BIGINT | - | N | - | 用户ID（PK，FK → user） |
| online_status | TINYINT | - | N | 1 | 1=在线, 2=离线, 3=忙碌, 4=请勿打扰 |
| custom_status | VARCHAR | 50 | Y | NULL | 自定义状态文字 |
| last_active_at | DATETIME | - | Y | NULL | 最后活跃时间 |
| updated_at | DATETIME | - | N | CURRENT_TIMESTAMP ON UPDATE | - |

**索引：**
- PRIMARY KEY (`user_id`)

**说明：** 与 user 表 1:1 关系，分离理由：状态频繁更新（每次心跳），避免写冲突影响 user 主表。同时配合 Redis 缓存实时状态。

---

#### 3.1.4 department — 部门表

| 列名 | 类型 | 长度 | 可空 | 默认值 | 说明 |
|-----|------|------|------|--------|------|
| dept_id | BIGINT | - | N | AUTO_INCREMENT | 主键 |
| tenant_id | BIGINT | - | N | - | 租户ID |
| parent_dept_id | BIGINT | - | Y | NULL | 父部门ID（自引用 FK） |
| dept_name | VARCHAR | 100 | N | - | 部门名称 |
| sort_order | INT | - | N | 0 | 排序顺序 |
| member_count | INT | - | N | 0 | 成员人数 |
| status | TINYINT | - | N | 1 | 1=正常, 0=停用 |
| created_at | DATETIME | - | N | CURRENT_TIMESTAMP | - |
| updated_at | DATETIME | - | N | CURRENT_TIMESTAMP ON UPDATE | - |

**索引：**
- PRIMARY KEY (`dept_id`)
- KEY `idx_tenant_parent` (`tenant_id`, `parent_dept_id`)

**说明：** parent_dept_id 自引用实现多级树形组织架构。数据实时从 HR 系统同步，增量更新。

---

#### 3.1.5 user_department — 用户部门关联表

| 列名 | 类型 | 长度 | 可空 | 默认值 | 说明 |
|-----|------|------|------|--------|------|
| id | BIGINT | - | N | AUTO_INCREMENT | 主键 |
| user_id | BIGINT | - | N | - | 用户ID，FK → user |
| dept_id | BIGINT | - | N | - | 部门ID，FK → department |
| is_primary | TINYINT | - | N | 0 | 1=主部门 |
| joined_at | DATETIME | - | N | CURRENT_TIMESTAMP | 加入时间 |

**索引：**
- PRIMARY KEY (`id`)
- UNIQUE KEY `uk_user_dept` (`user_id`, `dept_id`)
- KEY `idx_dept` (`dept_id`)

**说明：** 用户可属于多个部门。is_primary 标记主部门，对应 user.department_id。

---

### 3.2 会话消息

#### 3.2.1 conversation — 会话主表

| 列名 | 类型 | 长度 | 可空 | 默认值 | 说明 |
|-----|------|------|------|--------|------|
| conversation_id | BIGINT | - | N | AUTO_INCREMENT | 主键 |
| tenant_id | BIGINT | - | N | - | 租户ID |
| conversation_type | TINYINT | - | N | - | 1=单聊, 2=群聊 |
| owner_user_id | BIGINT | - | N | - | 会话所属用户 |
| target_id | BIGINT | - | N | - | 对方用户ID 或 群组ID |
| latest_msg_id | VARCHAR | 64 | Y | NULL | 最后一条消息ID（MongoDB _id） |
| latest_msg_content | VARCHAR | 500 | Y | NULL | 最后消息摘要 |
| latest_msg_type | TINYINT | - | Y | NULL | 最后消息类型 |
| latest_msg_time | DATETIME | - | Y | NULL | 最后消息时间 |
| unread_count | INT | - | N | 0 | 未读消息数 |
| is_pin | TINYINT | - | N | 0 | 1=置顶 |
| is_mute | TINYINT | - | N | 0 | 1=免打扰 |
| is_folded | TINYINT | - | N | 0 | 1=折叠 |
| draft_text | VARCHAR | 500 | Y | NULL | 草稿 |
| ext_json | JSON | - | Y | NULL | 扩展字段 |
| created_at | DATETIME | - | N | CURRENT_TIMESTAMP | - |
| updated_at | DATETIME | - | N | CURRENT_TIMESTAMP ON UPDATE | - |

**索引：**
- PRIMARY KEY (`conversation_id`)
- UNIQUE KEY `uk_owner_target` (`owner_user_id`, `conversation_type`, `target_id`)
- KEY `idx_owner_sort` (`owner_user_id`, `is_pin` DESC, `latest_msg_time` DESC)
- KEY `idx_tenant` (`tenant_id`)

**核心逻辑说明：**
- 每条会话以用户维度存储，单聊产生 2 条记录（A→B，B→A），群聊每成员 1 条
- 未读数实时更新：写 Redis 异步落 MySQL 持久化
- 置顶最多 5 条（业务层校验），`is_pin=1` 会话在 `idx_owner_sort` 索引中排最前
- 删除会话仅删除本地行，云端消息保留，`unread_count` 清零
- 容量约束：单用户总会话 ≤ 2000，超出仅清理本地列表

---

#### 3.2.2 file_resource — 文件资源表

| 列名 | 类型 | 长度 | 可空 | 默认值 | 说明 |
|-----|------|------|------|--------|------|
| file_id | BIGINT | - | N | AUTO_INCREMENT | 主键 |
| tenant_id | BIGINT | - | N | - | 租户ID |
| uploader_id | BIGINT | - | N | - | 上传者，FK → user |
| file_name | VARCHAR | 255 | N | - | 原始文件名 |
| file_size | BIGINT | - | N | - | 文件大小（字节） |
| file_type | VARCHAR | 20 | N | - | image/video/audio/document/other |
| mime_type | VARCHAR | 100 | Y | NULL | MIME 类型 |
| storage_path | VARCHAR | 500 | N | - | 存储路径（含bucket） |
| storage_type | TINYINT | - | N | 1 | 1=MinIO, 2=S3 |
| thumbnail_path | VARCHAR | 500 | Y | NULL | 缩略图路径 |
| sha256_hash | VARCHAR | 64 | Y | NULL | 文件哈希（去重） |
| status | TINYINT | - | N | 1 | 1=正常, 2=已清理 |
| created_at | DATETIME | - | N | CURRENT_TIMESTAMP | - |

**索引：**
- PRIMARY KEY (`file_id`)
- KEY `idx_tenant_uploader` (`tenant_id`, `uploader_id`)
- KEY `idx_tenant_type` (`tenant_id`, `file_type`)
- KEY `idx_hash` (`sha256_hash`)

**说明：** 仅存储文件元数据，二进制内容存 MinIO/S3。sha256_hash 用于秒传和去重。

---

### 3.3 群组管理

#### 3.3.1 group_info — 群组信息表

| 列名 | 类型 | 长度 | 可空 | 默认值 | 说明 |
|-----|------|------|------|--------|------|
| group_id | BIGINT | - | N | AUTO_INCREMENT | 主键 |
| tenant_id | BIGINT | - | N | - | 租户ID |
| group_name | VARCHAR | 100 | N | - | 群名称 |
| group_avatar | VARCHAR | 500 | Y | NULL | 群头像 |
| description | VARCHAR | 500 | Y | NULL | 群描述 |
| owner_id | BIGINT | - | N | - | 群主用户ID，FK → user |
| member_count | INT | - | N | 0 | 当前成员数 |
| max_members | INT | - | N | 2000 | 最大成员数上限 |
| join_type | TINYINT | - | N | 1 | 1=无需审批, 2=管理员审批 |
| status | TINYINT | - | N | 1 | 1=正常, 2=已解散 |
| notice | TEXT | - | Y | NULL | 群公告（富文本） |
| notice_updated_at | DATETIME | - | Y | NULL | 公告更新时间 |
| can_member_rename | TINYINT | - | N | 0 | 0=仅管理员, 1=全员可改名 |
| can_at_all | TINYINT | - | N | 0 | 0=仅管理员, 1=全员可@所有人 |
| can_member_invite | TINYINT | - | N | 0 | 0=仅管理员, 1=全员可邀请 |
| qrcode_url | VARCHAR | 500 | Y | NULL | 群二维码 |
| qrcode_updated_at | DATETIME | - | Y | NULL | 二维码刷新时间 |
| created_at | DATETIME | - | N | CURRENT_TIMESTAMP | - |
| updated_at | DATETIME | - | N | CURRENT_TIMESTAMP ON UPDATE | - |

**索引：**
- PRIMARY KEY (`group_id`)
- KEY `idx_tenant` (`tenant_id`)
- KEY `idx_owner` (`owner_id`)

**容量约束：**
- 单群最大 2000 人（max_members）
- 群名称上限 30 字符
- 公告上限 5000 字符

---

#### 3.3.2 group_member — 群成员关系表

| 列名 | 类型 | 长度 | 可空 | 默认值 | 说明 |
|-----|------|------|------|--------|------|
| id | BIGINT | - | N | AUTO_INCREMENT | 主键 |
| group_id | BIGINT | - | N | - | 群组ID，FK → group_info |
| user_id | BIGINT | - | N | - | 用户ID，FK → user |
| role | TINYINT | - | N | 3 | 1=群主, 2=管理员, 3=普通成员 |
| is_mute | TINYINT | - | N | 0 | 1=被禁言 |
| mute_end_time | DATETIME | - | Y | NULL | 禁言截止时间 |
| group_alias | VARCHAR | 64 | Y | NULL | 群昵称 |
| joined_at | DATETIME | - | N | CURRENT_TIMESTAMP | 加入时间 |

**索引：**
- PRIMARY KEY (`id`)
- UNIQUE KEY `uk_group_user` (`group_id`, `user_id`)
- KEY `idx_user_groups` (`user_id`)
- KEY `idx_group_role` (`group_id`, `role`)

**约束逻辑：**
- 群主数量 = 1（唯一），管理员 ≤ 成员总数 × 10%
- 群主不可被移除，转让后原群主降为普通成员
- 用户离职/删除：自动退群，历史消息标注「（已离职）」
- 群主离职：自动转让给最早管理员；无管理员则转让最早加入成员

---

#### 3.3.3 group_join_request — 入群申请表

| 列名 | 类型 | 长度 | 可空 | 默认值 | 说明 |
|-----|------|------|------|--------|------|
| request_id | BIGINT | - | N | AUTO_INCREMENT | 主键 |
| group_id | BIGINT | - | N | - | 群组ID，FK → group_info |
| user_id | BIGINT | - | N | - | 申请人，FK → user |
| reason | VARCHAR | 500 | Y | NULL | 申请理由 |
| status | TINYINT | - | N | 0 | 0=pending, 1=approved, 2=rejected, 3=expired |
| handled_by | BIGINT | - | Y | NULL | 处理人，FK → user |
| handled_at | DATETIME | - | Y | NULL | 处理时间 |
| created_at | DATETIME | - | N | CURRENT_TIMESTAMP | 申请时间 |
| expired_at | DATETIME | - | N | (created_at+7d) | 过期时间 |

**索引：**
- PRIMARY KEY (`request_id`)
- KEY `idx_group_status` (`group_id`, `status`)
- KEY `idx_user` (`user_id`)
- KEY `idx_expired` (`status`, `expired_at`)

**约束逻辑：**
- 有效期默认 7 天（1~30 天可配置）
- 单用户 24h 最多 3 次有效申请（rejected 计入，expired 不计）
- 待处理申请上限 50 条，超出必须处理旧申请
- 已过期（expired）和已拒绝（rejected）的记录保留 90 天后定时清理
- 已通过（approved）记录永久保留

---

### 3.4 机器人管理

#### 3.4.1 bot — 机器人配置表

| 列名 | 类型 | 长度 | 可空 | 默认值 | 说明 |
|-----|------|------|------|--------|------|
| bot_id | BIGINT | - | N | AUTO_INCREMENT | 主键 |
| tenant_id | BIGINT | - | N | - | 租户ID |
| bot_type | TINYINT | - | N | - | 1=系统机器人, 2=用户自定义 |
| bot_name | VARCHAR | 100 | N | - | 机器人名称 |
| avatar_url | VARCHAR | 500 | Y | NULL | 头像 |
| description | VARCHAR | 500 | Y | NULL | 描述 |
| webhook_url | VARCHAR | 500 | Y | NULL | Webhook 回调地址 |
| api_key | VARCHAR | 128 | Y | NULL | API 密钥 |
| status | TINYINT | - | N | 1 | 1=启用, 0=停用 |
| creator_id | BIGINT | - | Y | NULL | 创建者（用户机器人），FK → user |
| created_at | DATETIME | - | N | CURRENT_TIMESTAMP | - |
| updated_at | DATETIME | - | N | CURRENT_TIMESTAMP ON UPDATE | - |

**索引：**
- PRIMARY KEY (`bot_id`)
- KEY `idx_tenant` (`tenant_id`)
- KEY `idx_tenant_type` (`tenant_id`, `bot_type`)

**容量约束：**
- 租户自定义机器人 ≤ 20 个
- 系统机器人（通知助手、流程助手、文件助手）不可删除，仅可启停

---

#### 3.4.2 group_bot — 群机器人关联表

| 列名 | 类型 | 长度 | 可空 | 默认值 | 说明 |
|-----|------|------|------|--------|------|
| id | BIGINT | - | N | AUTO_INCREMENT | 主键 |
| group_id | BIGINT | - | N | - | 群组ID，FK → group_info |
| bot_id | BIGINT | - | N | - | 机器人ID，FK → bot |
| webhook_url | VARCHAR | 500 | Y | NULL | 群专属 Webhook 地址 |
| token | VARCHAR | 128 | Y | NULL | 回调验证 Token |
| status | TINYINT | - | N | 1 | 1=启用, 0=停用 |
| created_at | DATETIME | - | N | CURRENT_TIMESTAMP | - |

**索引：**
- PRIMARY KEY (`id`)
- UNIQUE KEY `uk_group_bot` (`group_id`, `bot_id`)
- KEY `idx_bot` (`bot_id`)

**容量约束：** 单群 Webhook 机器人 ≤ 5 个。

---

### 3.5 风控管理

#### 3.5.1 sensitive_word — 敏感词库表

| 列名 | 类型 | 长度 | 可空 | 默认值 | 说明 |
|-----|------|------|------|--------|------|
| word_id | BIGINT | - | N | AUTO_INCREMENT | 主键 |
| tenant_id | BIGINT | - | N | - | 租户ID |
| word | VARCHAR | 200 | N | - | 敏感词 |
| strategy | TINYINT | - | N | 1 | 1=替换***, 2=拦截, 3=仅记录日志 |
| replacement | VARCHAR | 10 | Y | '***' | 替换字符 |
| category | VARCHAR | 50 | Y | NULL | 分类 |
| status | TINYINT | - | N | 1 | 1=启用, 0=停用 |
| created_at | DATETIME | - | N | CURRENT_TIMESTAMP | - |
| updated_at | DATETIME | - | N | CURRENT_TIMESTAMP ON UPDATE | - |

**索引：**
- PRIMARY KEY (`word_id`)
- KEY `idx_tenant` (`tenant_id`)
- KEY `idx_word` (`word`(20))

**说明：**
- 支持批量导入/导出
- 匹配算法：DFA（确定性有限自动机），时间复杂度 O(n)
- 匹配耗时要求 ≤ 100ms
- 替换策略：敏感词前后保留各 1/4 字符，中间替换为 `***`

---

#### 3.5.2 frequency_control_rule — 频控规则表

| 列名 | 类型 | 长度 | 可空 | 默认值 | 说明 |
|-----|------|------|------|--------|------|
| rule_id | BIGINT | - | N | AUTO_INCREMENT | 主键 |
| tenant_id | BIGINT | - | N | - | 租户ID |
| target_type | TINYINT | - | N | - | 1=用户, 2=机器人 |
| max_count | INT | - | N | - | 时间窗口内最大请求数 |
| time_window_seconds | INT | - | N | - | 时间窗口（秒） |
| action | TINYINT | - | N | 1 | 1=限流返回429, 2=拦截 |
| status | TINYINT | - | N | 1 | 1=启用, 0=停用 |
| created_at | DATETIME | - | N | CURRENT_TIMESTAMP | - |

**索引：**
- PRIMARY KEY (`rule_id`)
- KEY `idx_tenant` (`tenant_id`)

**默认规则（PRD §4.9.5）：**
- 用户：1s 最多 5 条
- 机器人：1s 最多 10 条

---

#### 3.5.3 file_limit_config — 文件上传限制表

| 列名 | 类型 | 长度 | 可空 | 默认值 | 说明 |
|-----|------|------|------|--------|------|
| config_id | BIGINT | - | N | AUTO_INCREMENT | 主键 |
| tenant_id | BIGINT | - | N | - | 租户ID |
| file_type | VARCHAR | 20 | N | - | image/video/document |
| max_size_mb | INT | - | N | - | 文件大小上限（MB） |
| allowed_extensions | VARCHAR | 500 | Y | NULL | 允许扩展名（逗号分隔） |
| status | TINYINT | - | N | 1 | 1=启用, 0=停用 |
| created_at | DATETIME | - | N | CURRENT_TIMESTAMP | - |

**索引：**
- PRIMARY KEY (`config_id`)
- KEY `idx_tenant_type` (`tenant_id`, `file_type`)

**默认限制（PRD §4.9.5）：**
- 图片：10MB，扩展名 jpg/jpeg/png/gif/webp
- 视频：100MB，扩展名 mp4/mov/avi/mkv
- 文件：50MB，扩展名 pdf/doc/docx/xls/xlsx/ppt/pptx/txt/zip

---

### 3.6 其他

#### 3.6.1 card_template — 卡片模板表

| 列名 | 类型 | 长度 | 可空 | 默认值 | 说明 |
|-----|------|------|------|--------|------|
| template_id | BIGINT | - | N | AUTO_INCREMENT | 主键 |
| tenant_id | BIGINT | - | N | - | 租户ID |
| template_name | VARCHAR | 100 | N | - | 模板名称 |
| template_schema | JSON | - | N | - | 卡片 JSON Schema 定义 |
| status | TINYINT | - | N | 1 | 1=启用, 0=停用 |
| version | INT | - | N | 1 | 版本号 |
| created_at | DATETIME | - | N | CURRENT_TIMESTAMP | - |
| updated_at | DATETIME | - | N | CURRENT_TIMESTAMP ON UPDATE | - |

**索引：**
- PRIMARY KEY (`template_id`)
- KEY `idx_tenant` (`tenant_id`)

**说明：**
- template_schema 存储卡片结构的 JSON Schema，包含标题区、内容区、状态区、操作区定义
- 业务系统通过外部接口推送卡片时，传入参数 + 模板 ID 动态渲染
- 版本变更时兼容旧消息展示

---

#### 3.6.2 system_config — 系统配置表

| 列名 | 类型 | 长度 | 可空 | 默认值 | 说明 |
|-----|------|------|------|--------|------|
| config_id | BIGINT | - | N | AUTO_INCREMENT | 主键 |
| tenant_id | BIGINT | - | N | - | 租户ID |
| config_key | VARCHAR | 100 | N | - | 配置键 |
| config_value | TEXT | - | N | - | 配置值（JSON 字符串） |
| description | VARCHAR | 200 | Y | NULL | 描述 |
| created_at | DATETIME | - | N | CURRENT_TIMESTAMP | - |
| updated_at | DATETIME | - | N | CURRENT_TIMESTAMP ON UPDATE | - |

**索引：**
- PRIMARY KEY (`config_id`)
- UNIQUE KEY `uk_tenant_key` (`tenant_id`, `config_key`)

**说明：** KV 结构的灵活配置表，用于存储不适合硬编码的租户级配置项。

---

## 4. MongoDB 集合设计

### 4.1 分片策略总览

| 集合 | Shard Key | 分片策略 | 说明 |
|------|----------|---------|------|
| messages | `conversation_id` hashed | 哈希分片 | 写入均匀分布，同会话消息集中（单分片） |
| admin_audit_logs | `tenant_id` hashed | 哈希分片 | 按租户分散，避免单热点 |
| message_audit_logs | `tenant_id` hashed | 哈希分片 | 同上 |
| offline_messages | 无（小集合） | 未分片 | TTL 自动过期，数据量小 |

---

### 4.2 messages — 消息记录集合

**设计容量：** 10万在线 × 日均100条/人 = 1000万条/日，3年 ≈ 10亿条

**Shard Key：** `{ conversation_id: "hashed" }`

**文档结构：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `_id` | ObjectId | MongoDB 默认 ID |
| `msg_id` | UUID/String | 业务消息 ID（全局唯一，业务层生成） |
| `tenant_id` | Long | 租户ID |
| `conversation_id` | Long | 会话ID |
| `sender_id` | Long | 发送者用户ID |
| `msg_type` | Int | 1=text, 2=image, 3=video, 4=file, 5=voice, 6=card, 7=system, 8=@call |
| `content` | Object | 消息内容体（类型不同结构不同） |
| `status` | Int | 1=sending, 2=sent, 3=failed, 4=recalled |
| `send_time` | Date | 发送时间 |
| `edit_time` | Date | 编辑时间（可空） |
| `is_deleted` | Boolean | 软删除标记 |
| `ext_fields` | Object | 扩展字段 |

**content 对象结构：**

```jsonc
// msg_type=1 (text)
{ "text": "hello", "mentions": { "all": false, "users": [1001, 1002] } }

// msg_type=2 (image)
{ "url": "...", "thumbnail": "...", "width": 1920, "height": 1080, "size": 1048576 }

// msg_type=3 (video)
{ "url": "...", "thumbnail": "...", "duration": 30.5, "size": 10485760 }

// msg_type=4 (file)
{ "url": "...", "name": "report.pdf", "size": 2048000, "ext": "pdf" }

// msg_type=5 (voice)
{ "url": "...", "duration": 8.2, "size": 65536 }

// msg_type=6 (card)
{ "template_id": 1, "template_version": 1, "data": { "title": "...", "status": "...", "buttons": [...] } }

// msg_type=7 (system)
{ "text": "xxx 加入了群聊", "system_type": "member_join" }

// msg_type=8 (@call)
{ "text": "有人@你", "call_type": "mention", "conversation_id": 123, "sender_id": 456 }
```

**索引：**

| 索引 | 字段 | 说明 |
|------|------|------|
| PRIMARY | `_id` | 默认 |
| `idx_conv_time` | `{ conversation_id: 1, send_time: -1 }` | 按会话翻页拉取历史 |
| `idx_sender_time` | `{ sender_id: 1, send_time: -1 }` | 按发送者查询 |
| `idx_msg_text` | `{ "content.text": "text" }` | 全文检索（中文分词需配置 MongoDB Atlas / ICU） |
| `idx_msg_id` | `{ msg_id: 1 }` | 按业务消息ID精确查找 |

**说明：**
- 全文检索仅用于 Web 后台审计查询及会话内消息搜索
- conversation_id 作为 shard key，同一会话的消息集中在一个分片，方便翻页查询
- 单条消息大小平均约 2KB（纯文本）~ 200KB（含URL的媒体消息）

---

### 4.3 admin_audit_logs — 管理员操作审计集合

**Shard Key：** `{ tenant_id: "hashed" }`

**文档结构：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `_id` | ObjectId | - |
| `tenant_id` | Long | 租户ID |
| `operator_id` | Long | 操作员用户ID |
| `operator_ip` | String | 操作IP |
| `action_type` | String | 操作类型（login/group_create/msg_recall/bot_enable/...） |
| `target_type` | String | 操作对象类型（user/group/bot/sensitive_word/...） |
| `target_id` | String | 操作对象ID |
| `detail` | Object | 操作详情（JSON） |
| `result` | String | success/fail |
| `fail_reason` | String | 失败原因 |
| `created_at` | Date | 操作时间 |

**索引：**

| 索引 | 字段 | 说明 |
|------|------|------|
| `idx_tenant_time` | `{ tenant_id: 1, created_at: -1 }` | 按租户按时间倒序查询 |
| `idx_operator_time` | `{ operator_id: 1, created_at: -1 }` | 按操作人员查询 |
| `idx_action_time` | `{ action_type: 1, created_at: -1 }` | 按操作类型筛选 |

**数据保留：** TTL 索引 2 年自动过期（日志不可删改，到期自动清理）

**说明：**
- 对应 PRD §7.1，记录项包含：操作时间、操作员账号、IP、操作类型、操作对象、执行结果
- 日志不可修改删除，只追加
- 支持按时间、人员、操作类型筛选导出

---

### 4.4 message_audit_logs — 消息审计日志集合

**Shard Key：** `{ tenant_id: "hashed" }`

**文档结构：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `_id` | ObjectId | - |
| `tenant_id` | Long | 租户ID |
| `msg_id` | String | 消息ID |
| `sender_id` | Long | 发送者ID |
| `sender_ip` | String | 发送者IP |
| `conversation_id` | Long | 所属会话ID |
| `msg_type` | Int | 消息类型 |
| `content_snapshot` | String | 消息内容快照（文本截取前500字符） |
| `sensitive_word_hit` | [String] | 命中的敏感词列表 |
| `sent_at` | Date | 发送时间 |

**索引：**

| 索引 | 字段 | 说明 |
|------|------|------|
| `idx_tenant_time` | `{ tenant_id: 1, sent_at: -1 }` | 按租户+时间查询 |
| `idx_sender` | `{ sender_id: 1 }` | 按发送人查询 |
| `idx_conv` | `{ conversation_id: 1 }` | 按会话查询 |
| `idx_content_text` | `{ content_snapshot: "text" }` | 内容关键词检索 |
| `idx_sensitive_hit` | `{ sensitive_word_hit: 1 }` | 命中敏感词筛选 |

**数据保留：** TTL 索引 6 个月自动过期

**说明：**
- 对应 PRD §7.2，用于企业合规审计
- 支持多条件联合检索：发送人、接收会话、时间区间、关键词、消息类型、敏感词命中
- 内容快照仅截取前 500 字符用于检索，完整消息内容存 messages 集合
- 敏感词命中记录命中的具体词汇列表

---

### 4.5 offline_messages — 离线消息暂存集合

**Shard Key：** 无需分片（小集合，TTL 自动清理）

**文档结构：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `_id` | ObjectId | - |
| `user_id` | Long | 接收者用户ID |
| `messages` | [Object] | 离线消息数组 |
| `created_at` | Date | 创建时间 |

**TTL 索引：** `{ created_at: 1 }`，`expireAfterSeconds: 604800`（7天）

**说明：**
- 用户在线时通过 WebSocket 实时接收消息，无需写入此集合
- 仅当用户离线时，消息暂存于此
- 用户上线后拉取并删除对应文档
- PRD 要求离线消息永久存储（MongoDB messages 已满足），此集合仅作暂存转发

---

## 5. Redis 数据结构

### 5.1 在线状态

```
Key:       user:online:{user_id}
Value:     "online" | "busy" | "dnd"
TTL:       120 秒（Web 每 2 分钟心跳刷新）
说明:     30 分钟无心跳则自动离线，Redis 中 Key 过期后回查 MySQL 确认离线
```

### 5.2 会话未读计数

```
Key:       conv:unread:{user_id}
Type:      Hash
Field:     {conversation_id}
Value:    未读数量（Int）
说明:     消息到达 HINCRBY +1，读会话后 DEL field
          持久化：每分钟扫描前 100 个活跃会话刷入 MySQL conversation.unread_count
```

### 5.3 消息频控（滑动窗口）

```
Key:       rate:user:{user_id}
Type:      Sorted Set
Score:     当前时间戳（毫秒）
Member:    消息ID / 随机数
操作:
  发消息前: ZREMRANGEBYSCORE {key} 0 {now - window} + ZCARD
  超过上限: 返回 429
  放行:     ZADD {key} {now} {random}
TTL:       窗口时间 × 2

默认阈值:
  用户:  1s 窗口内 ≤ 5 条
  机器人: 1s 窗口内 ≤ 10 条
```

### 5.4 分布式锁

```
Key:       lock:{resource_type}:{resource_id}
Value:     {instance_id}:{thread_id}
NX + TTL:  10 秒（防止死锁）
用途:
  - 群主转让（防止并发操作）
  - 机器人配置变更
  - 敏感词批量导入
```

### 5.5 会话置顶列表

```
Key:       pin:{user_id}
Type:      Sorted Set
Score:     置顶时间戳（毫秒，越晚越前）
Member:    {conversation_id}
操作:
  置顶:     ZADD {key} {timestamp} {conv_id}
  取消置顶: ZREM {key} {conv_id}
  查询:     ZREVRANGE {key} 0 -1
约束:     业务层校验 ZCARD ≤ 5，超出时 ZREMRANGEBYRANK 移除 score 最小的
```

### 5.6 用户 Token

```
Key:       token:{token_string}
Value:     { user_id, tenant_id, device_id }
TTL:       7 天（每次请求自动续期）
用途:      API 请求鉴权，减少 MySQL 查询
```

### 5.7 Webhook 幂等去重

```
Key:       idempotent:webhook:{msg_id}
Value:     "processed"
TTL:       1 小时
说明:     Webhook 回调幂等校验，重复回调直接忽略
```

---

## 6. 索引策略总结

### 6.1 MySQL 核心查询索引对照

| 查询场景 | 表 | 索引 | 类型 |
|---------|------|------|------|
| 用户登录 | user | `uk_tenant_account` | 唯一 |
| 会话列表（按时间倒序，置顶优先） | conversation | `idx_owner_sort` | B+Tree |
| 单聊查会话 | conversation | `uk_owner_target` | 唯一 |
| 群成员列表 + 角色 | group_member | `idx_group_role` | B+Tree |
| 入群待审批列表 | group_join_request | `idx_group_status` | B+Tree |
| 消息历史按会话查询 | messages (MongoDB) | `idx_conv_time` | B+Tree |
| 敏感词全词匹配 | sensitive_word | `idx_word(20)` | B+Tree（前缀） |
| 审计日志按租户时间倒序 | admin_audit_logs (MongoDB) | `idx_tenant_time` | B+Tree |

### 6.2 复合索引设计原则

1. **等值条件在前，范围条件在后** — 如 `idx_owner_sort`：(owner_user_id 等值, is_pin 等值, latest_msg_time 范围)
2. **覆盖索引优先** — `uk_owner_target` 覆盖 owner + type + target 查询，避免回表
3. **索引下推** — MySQL 5.6+ 利用 `idx_tenant_status` 在索引内过滤 status
4. **短索引** — `sensitive_word.word(20)` 前缀索引，减少索引空间

---

## 7. 分片与分区策略

### 7.1 MongoDB 分片

| 集合 | Shard Key | 需分片节点 | 预分片建议 |
|------|----------|-----------|-----------|
| messages | `conversation_id` hashed | 3+ | 提前创建 4096 个 chunks |
| admin_audit_logs | `tenant_id` hashed | 2+ | 提前创建 1024 个 chunks |
| message_audit_logs | `tenant_id` hashed | 2+ | 提前创建 1024 个 chunks |

### 7.2 MySQL 分表策略

**当前阶段不需要分表。** 原因：

| 表 | 预估量级 | 分表必要性 |
|----|---------|----------|
| user | 万级 ~ 百万级 | ❌ 不需要 |
| conversation | 单用户 ≤ 2000，万级租户 = 千万级 | ⚠️ 观察后决策 |
| group_member | 群均 100 人，万群 = 百万级 | ❌ 不需要 |
| group_join_request | 留存 90 天清理 | ❌ 不需要 |
| sensitive_word | 千级 | ❌ 不需要 |

**如需分表优先考虑：**
- `conversation` 表按 `tenant_id` 哈希分 4~8 张表
- 使用 ShardingSphere 或应用层分表中间件

---

## 8. 数据归档与清理

| 数据 | 位置 | 保留策略 | 清理方式 |
|------|------|---------|---------|
| 消息历史 | MongoDB messages | 永久 | 不清理 |
| 管理员操作日志 | MongoDB admin_audit_logs | 2年 | TTL 索引自动过期 |
| 消息审计日志 | MongoDB message_audit_logs | 6个月 | TTL 索引自动过期 |
| 离线消息 | MongoDB offline_messages | 7天 | TTL 索引自动过期 |
| 入群申请（已拒绝/已过期） | MySQL group_join_request | 90天 | 定时任务 DELETE + 归档 |
| 撤回消息 | MongoDB messages | 标记撤回，内容保留 | 物理不清除 |
| 文件资源（离职用户单聊） | MinIO/S3 | 用户离职清理 | 定时任务清理孤儿文件 |

**定时清理脚本建议：**
```cron
# 每天凌晨 3:00 清理过期入群申请
0 3 * * * mysql -e "DELETE FROM group_join_request WHERE status IN (2,3) AND created_at < DATE_SUB(NOW(), INTERVAL 90 DAY)"

# 每天凌晨 4:00 清理无主文件
0 4 * * * php /scripts/cleanup_orphan_files.php
```

---

## 9. 容量估算

### 9.1 单条数据大小估算

| 表/集合 | 单条大小 | 说明 |
|---------|---------|------|
| user | ~500B | 含 JSON 扩展字段 |
| conversation | ~300B | 含摘要文本 |
| group_member | ~100B | 轻量关联 |
| messages (文本) | ~2KB | 含 content 对象 |
| messages (带媒体URL) | ~512B | 仅存URL，二进制在 MinIO |
| admin_audit_logs | ~1KB | 含 detail JSON |
| message_audit_logs | ~512B | 含快照文本 |

### 9.2 年增量估算（万级用户租户）

| 数据 | 日增量 | 月增量 | 年增量 |
|------|-------|-------|-------|
| messages | 1000万条 ≈ 5GB | 150亿条 ≈ 150GB | ~1.8TB |
| admin_audit_logs | 5万条 ≈ 50MB | 150万条 ≈ 1.5GB | ~18GB |
| message_audit_logs | 1000万条 ≈ 5GB | 3亿条 ≈ 150GB | ~1.8TB |
| file_resource | 5万条 ≈ 10MB | 150万条 ≈ 300MB | ~3.6GB |

### 9.3 磁盘建议

- **MongoDB**：SSD 2TB × 3 副本起步（按消息量扩容），建议 7.68TB NVMe SSD
- **MySQL**：SSD 500GB × 主从（主要存储结构化数据，量级可控）
- **Redis**：32GB 内存起步（在线状态 + 未读计数 + 频控）
- **MinIO/S3**：按实际文件量弹性扩展，建议 10TB 起步

---

## 10. 数据库初始化脚本

数据库初始化 SQL 脚本将产出至 `deploy/db/init.sql`，包含：

1. 数据库创建：`CREATE DATABASE IF NOT EXISTS im_shared CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;`
2. 17 张表完整的 `CREATE TABLE` 语句（含索引）
3. 默认配置数据初始化（系统机器人、默认频控规则、默认文件限制）
4. MongoDB 分片初始化命令（`sh.shardCollection()`）

初始化脚本在部署阶段由 Docker Compose 或 K8s init container 自动执行。
