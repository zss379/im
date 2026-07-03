-- ============================================================
-- 数莲 PaaS IM 系统 — 数据库初始化脚本
-- 目标数据库: MySQL 8.0
-- 执行方式: docker-compose 启动时自动执行
-- ============================================================

CREATE DATABASE IF NOT EXISTS im_shared
    CHARACTER SET utf8mb4
    COLLATE utf8mb4_unicode_ci;

USE im_shared;

-- ============================================================
-- 1. 租户体系
-- ============================================================

-- 1.1 租户表
CREATE TABLE IF NOT EXISTS tenant (
    tenant_id       BIGINT       NOT NULL AUTO_INCREMENT,
    tenant_name     VARCHAR(100) NOT NULL COMMENT '租户名称',
    domain          VARCHAR(200) DEFAULT NULL COMMENT '租户域名',
    logo_url        VARCHAR(500) DEFAULT NULL COMMENT '租户Logo',
    status          TINYINT      NOT NULL DEFAULT 1 COMMENT '1=正常, 0=停用',
    contact_phone   VARCHAR(20)  DEFAULT NULL COMMENT '联系电话',
    contact_email   VARCHAR(100) DEFAULT NULL COMMENT '联系邮箱',
    created_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (tenant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='租户表';

-- 1.2 用户基础表
CREATE TABLE IF NOT EXISTS user (
    user_id         BIGINT       NOT NULL AUTO_INCREMENT,
    tenant_id       BIGINT       NOT NULL COMMENT '租户ID',
    account         VARCHAR(64)  NOT NULL COMMENT '登录账号',
    password_hash   VARCHAR(256) NOT NULL COMMENT 'bcrypt加密密码',
    phone           VARCHAR(20)  DEFAULT NULL COMMENT '手机号(AES加密)',
    email           VARCHAR(100) DEFAULT NULL COMMENT '邮箱',
    display_name    VARCHAR(64)  NOT NULL COMMENT '显示名称',
    avatar_url      VARCHAR(500) DEFAULT NULL COMMENT '头像URL',
    position        VARCHAR(100) DEFAULT NULL COMMENT '岗位',
    department_id   BIGINT       DEFAULT NULL COMMENT '主部门ID',
    status          TINYINT      NOT NULL DEFAULT 1 COMMENT '1=正常, 2=离职, 3=删除',
    ext_json        JSON         DEFAULT NULL COMMENT '扩展字段',
    created_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id),
    UNIQUE KEY uk_tenant_account (tenant_id, account),
    KEY idx_tenant_status (tenant_id, status),
    KEY idx_tenant_dept (tenant_id, department_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户基础表';

-- 1.3 用户状态表
CREATE TABLE IF NOT EXISTS user_status (
    user_id          BIGINT    NOT NULL COMMENT '用户ID(PK)',
    online_status    TINYINT   NOT NULL DEFAULT 1 COMMENT '1=在线, 2=离线, 3=忙碌, 4=请勿打扰',
    custom_status    VARCHAR(50) DEFAULT NULL COMMENT '自定义状态文字',
    last_active_at   DATETIME  DEFAULT NULL COMMENT '最后活跃时间',
    updated_at       DATETIME  NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户状态表';

-- 1.4 部门表
CREATE TABLE IF NOT EXISTS department (
    dept_id         BIGINT       NOT NULL AUTO_INCREMENT,
    tenant_id       BIGINT       NOT NULL COMMENT '租户ID',
    parent_dept_id  BIGINT       DEFAULT NULL COMMENT '父部门ID',
    dept_name       VARCHAR(100) NOT NULL COMMENT '部门名称',
    sort_order      INT          NOT NULL DEFAULT 0 COMMENT '排序顺序',
    member_count    INT          NOT NULL DEFAULT 0 COMMENT '成员人数',
    status          TINYINT      NOT NULL DEFAULT 1 COMMENT '1=正常, 0=停用',
    created_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (dept_id),
    KEY idx_tenant_parent (tenant_id, parent_dept_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='部门表';

-- 1.5 用户部门关联表
CREATE TABLE IF NOT EXISTS user_department (
    id          BIGINT  NOT NULL AUTO_INCREMENT,
    user_id     BIGINT  NOT NULL,
    dept_id     BIGINT  NOT NULL,
    is_primary  TINYINT NOT NULL DEFAULT 0 COMMENT '1=主部门',
    joined_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_user_dept (user_id, dept_id),
    KEY idx_dept (dept_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户部门关联表';

-- ============================================================
-- 2. 会话消息
-- ============================================================

-- 2.1 会话主表
CREATE TABLE IF NOT EXISTS conversation (
    conversation_id   BIGINT       NOT NULL AUTO_INCREMENT,
    tenant_id         BIGINT       NOT NULL COMMENT '租户ID',
    conversation_type TINYINT      NOT NULL COMMENT '1=单聊, 2=群聊',
    owner_user_id     BIGINT       NOT NULL COMMENT '会话所属用户',
    target_id         BIGINT       NOT NULL COMMENT '对方用户ID或群组ID',
    latest_msg_id     VARCHAR(64)  DEFAULT NULL COMMENT '最后一条消息ID',
    latest_msg_content VARCHAR(500) DEFAULT NULL COMMENT '最后消息摘要',
    latest_msg_type   TINYINT      DEFAULT NULL COMMENT '最后消息类型',
    latest_msg_time   DATETIME     DEFAULT NULL COMMENT '最后消息时间',
    unread_count      INT          NOT NULL DEFAULT 0 COMMENT '未读消息数',
    is_pin            TINYINT      NOT NULL DEFAULT 0 COMMENT '1=置顶',
    is_mute           TINYINT      NOT NULL DEFAULT 0 COMMENT '1=免打扰',
    is_folded         TINYINT      NOT NULL DEFAULT 0 COMMENT '1=折叠',
    draft_text        VARCHAR(500) DEFAULT NULL COMMENT '草稿',
    ext_json          JSON         DEFAULT NULL COMMENT '扩展字段',
    created_at        DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (conversation_id),
    UNIQUE KEY uk_owner_target (owner_user_id, conversation_type, target_id),
    KEY idx_owner_sort (owner_user_id, is_pin DESC, latest_msg_time DESC),
    KEY idx_tenant (tenant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='会话表';

-- 2.2 文件资源表
CREATE TABLE IF NOT EXISTS file_resource (
    file_id         BIGINT       NOT NULL AUTO_INCREMENT,
    tenant_id       BIGINT       NOT NULL COMMENT '租户ID',
    uploader_id     BIGINT       NOT NULL COMMENT '上传者',
    file_name       VARCHAR(255) NOT NULL COMMENT '原始文件名',
    file_size       BIGINT       NOT NULL COMMENT '文件大小(字节)',
    file_type       VARCHAR(20)  NOT NULL COMMENT 'image/video/audio/document/other',
    mime_type       VARCHAR(100) DEFAULT NULL COMMENT 'MIME类型',
    storage_path    VARCHAR(500) NOT NULL COMMENT '存储路径',
    storage_type    TINYINT      NOT NULL DEFAULT 1 COMMENT '1=MinIO, 2=S3',
    thumbnail_path  VARCHAR(500) DEFAULT NULL COMMENT '缩略图路径',
    sha256_hash     VARCHAR(64)  DEFAULT NULL COMMENT '文件哈希',
    status          TINYINT      NOT NULL DEFAULT 1 COMMENT '1=正常, 2=已清理',
    created_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (file_id),
    KEY idx_tenant_uploader (tenant_id, uploader_id),
    KEY idx_tenant_type (tenant_id, file_type),
    KEY idx_hash (sha256_hash)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='文件资源表';

-- ============================================================
-- 3. 群组管理
-- ============================================================

-- 3.1 群组信息表
CREATE TABLE IF NOT EXISTS group_info (
    group_id          BIGINT       NOT NULL AUTO_INCREMENT,
    tenant_id         BIGINT       NOT NULL COMMENT '租户ID',
    group_name        VARCHAR(100) NOT NULL COMMENT '群名称',
    group_avatar      VARCHAR(500) DEFAULT NULL COMMENT '群头像',
    description       VARCHAR(500) DEFAULT NULL COMMENT '群描述',
    owner_id          BIGINT       NOT NULL COMMENT '群主用户ID',
    member_count      INT          NOT NULL DEFAULT 0 COMMENT '当前成员数',
    max_members       INT          NOT NULL DEFAULT 2000 COMMENT '最大成员数',
    join_type         TINYINT      NOT NULL DEFAULT 1 COMMENT '1=无需审批, 2=管理员审批',
    status            TINYINT      NOT NULL DEFAULT 1 COMMENT '1=正常, 2=已解散',
    notice            TEXT         DEFAULT NULL COMMENT '群公告(富文本)',
    notice_updated_at DATETIME     DEFAULT NULL COMMENT '公告更新时间',
    can_member_rename TINYINT      NOT NULL DEFAULT 0 COMMENT '0=仅管理员, 1=全员可改名',
    can_at_all        TINYINT      NOT NULL DEFAULT 0 COMMENT '0=仅管理员, 1=全员可@所有人',
    can_member_invite TINYINT      NOT NULL DEFAULT 0 COMMENT '0=仅管理员, 1=全员可邀请',
    qrcode_url        VARCHAR(500) DEFAULT NULL COMMENT '群二维码URL',
    qrcode_updated_at DATETIME     DEFAULT NULL COMMENT '二维码刷新时间',
    created_at        DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (group_id),
    KEY idx_tenant (tenant_id),
    KEY idx_owner (owner_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='群组信息表';

-- 3.2 群成员关系表
CREATE TABLE IF NOT EXISTS group_member (
    id            BIGINT   NOT NULL AUTO_INCREMENT,
    group_id      BIGINT   NOT NULL COMMENT '群组ID',
    user_id       BIGINT   NOT NULL COMMENT '用户ID',
    role          TINYINT  NOT NULL DEFAULT 3 COMMENT '1=群主, 2=管理员, 3=普通成员',
    is_mute       TINYINT  NOT NULL DEFAULT 0 COMMENT '1=被禁言',
    mute_end_time DATETIME DEFAULT NULL COMMENT '禁言截止时间',
    group_alias   VARCHAR(64) DEFAULT NULL COMMENT '群昵称',
    joined_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '加入时间',
    PRIMARY KEY (id),
    UNIQUE KEY uk_group_user (group_id, user_id),
    KEY idx_user_groups (user_id),
    KEY idx_group_role (group_id, role)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='群成员关系表';

-- 3.3 入群申请表
CREATE TABLE IF NOT EXISTS group_join_request (
    request_id   BIGINT       NOT NULL AUTO_INCREMENT,
    group_id     BIGINT       NOT NULL COMMENT '群组ID',
    user_id      BIGINT       NOT NULL COMMENT '申请人',
    reason       VARCHAR(500) DEFAULT NULL COMMENT '申请理由',
    status       TINYINT      NOT NULL DEFAULT 0 COMMENT '0=pending, 1=approved, 2=rejected, 3=expired',
    handled_by   BIGINT       DEFAULT NULL COMMENT '处理人',
    handled_at   DATETIME     DEFAULT NULL COMMENT '处理时间',
    created_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '申请时间',
    expired_at   DATETIME     NOT NULL COMMENT '过期时间(默认7天)',
    PRIMARY KEY (request_id),
    KEY idx_group_status (group_id, status),
    KEY idx_user (user_id),
    KEY idx_expired (status, expired_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='入群申请表';

-- ============================================================
-- 4. 机器人管理
-- ============================================================

-- 4.1 机器人配置表
CREATE TABLE IF NOT EXISTS bot (
    bot_id         BIGINT       NOT NULL AUTO_INCREMENT,
    tenant_id      BIGINT       NOT NULL COMMENT '租户ID',
    bot_type       TINYINT      NOT NULL COMMENT '1=系统机器人, 2=用户自定义',
    bot_name       VARCHAR(100) NOT NULL COMMENT '机器人名称',
    avatar_url     VARCHAR(500) DEFAULT NULL COMMENT '头像URL',
    description    VARCHAR(500) DEFAULT NULL COMMENT '描述',
    webhook_url    VARCHAR(500) DEFAULT NULL COMMENT 'Webhook地址',
    api_key        VARCHAR(128) DEFAULT NULL COMMENT 'API密钥',
    openim_user_id BIGINT       DEFAULT NULL COMMENT 'OpenIM用户ID',
    response_mode  TINYINT      NOT NULL DEFAULT 1 COMMENT '1=sync, 2=async, 3=sse',
    callback_url   VARCHAR(500) DEFAULT NULL COMMENT '异步回调接收URL',
    ip_whitelist   JSON         DEFAULT NULL COMMENT 'IP白名单',
    status         TINYINT      NOT NULL DEFAULT 1 COMMENT '1=启用, 0=停用',
    creator_id     BIGINT       DEFAULT NULL COMMENT '创建者',
    created_at     DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at     DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (bot_id),
    KEY idx_tenant (tenant_id),
    KEY idx_tenant_type (tenant_id, bot_type),
    KEY idx_openim_user (openim_user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='机器人配置表';

-- 4.2 群机器人关联表
CREATE TABLE IF NOT EXISTS group_bot (
    id              BIGINT       NOT NULL AUTO_INCREMENT,
    group_id        BIGINT       NOT NULL,
    bot_id          BIGINT       NOT NULL,
    webhook_url     VARCHAR(500) DEFAULT NULL COMMENT '群专属Webhook地址',
    token           VARCHAR(128) DEFAULT NULL COMMENT '回调验证Token',
    response_mode   TINYINT      NOT NULL DEFAULT 1 COMMENT '1=sync, 2=async, 3=sse',
    callback_url    VARCHAR(500) DEFAULT NULL COMMENT '异步回调接收URL',
    status          TINYINT      NOT NULL DEFAULT 1 COMMENT '1=启用, 0=停用',
    created_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_group_bot (group_id, bot_id),
    KEY idx_bot (bot_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='群机器人关联表';

-- ============================================================
-- 5. 风控管理
-- ============================================================

-- 5.1 敏感词库表
CREATE TABLE IF NOT EXISTS sensitive_word (
    word_id     BIGINT       NOT NULL AUTO_INCREMENT,
    tenant_id   BIGINT       NOT NULL COMMENT '租户ID',
    word        VARCHAR(200) NOT NULL COMMENT '敏感词',
    strategy    TINYINT      NOT NULL DEFAULT 1 COMMENT '1=替换***, 2=拦截, 3=仅记录日志',
    replacement VARCHAR(10)  DEFAULT '***' COMMENT '替换字符',
    category    VARCHAR(50)  DEFAULT NULL COMMENT '分类',
    status      TINYINT      NOT NULL DEFAULT 1 COMMENT '1=启用, 0=停用',
    created_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (word_id),
    KEY idx_tenant (tenant_id),
    KEY idx_word (word(20))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='敏感词库表';

-- 5.2 频控规则表
CREATE TABLE IF NOT EXISTS frequency_control_rule (
    rule_id             BIGINT  NOT NULL AUTO_INCREMENT,
    tenant_id           BIGINT  NOT NULL COMMENT '租户ID',
    target_type         TINYINT NOT NULL COMMENT '1=用户, 2=机器人',
    max_count           INT     NOT NULL COMMENT '最大请求次数',
    time_window_seconds INT     NOT NULL COMMENT '时间窗口(秒)',
    action              TINYINT NOT NULL DEFAULT 1 COMMENT '1=限流返回429, 2=拦截',
    status              TINYINT NOT NULL DEFAULT 1 COMMENT '1=启用, 0=停用',
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (rule_id),
    KEY idx_tenant (tenant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='频控规则表';

-- 5.3 文件上传限制表
CREATE TABLE IF NOT EXISTS file_limit_config (
    config_id          BIGINT       NOT NULL AUTO_INCREMENT,
    tenant_id          BIGINT       NOT NULL,
    file_type          VARCHAR(20)  NOT NULL COMMENT 'image/video/document',
    max_size_mb        INT          NOT NULL COMMENT '文件大小上限(MB)',
    allowed_extensions VARCHAR(500) DEFAULT NULL COMMENT '允许扩展名,逗号分隔',
    status             TINYINT      NOT NULL DEFAULT 1,
    created_at         DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (config_id),
    KEY idx_tenant_type (tenant_id, file_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='文件上传限制配置表';

-- ============================================================
-- 6. 其他
-- ============================================================

-- 6.1 卡片模板表
CREATE TABLE IF NOT EXISTS card_template (
    template_id     BIGINT       NOT NULL AUTO_INCREMENT,
    tenant_id       BIGINT       NOT NULL COMMENT '租户ID',
    template_name   VARCHAR(100) NOT NULL COMMENT '模板名称',
    template_schema JSON         NOT NULL COMMENT '卡片JSON Schema定义',
    status          TINYINT      NOT NULL DEFAULT 1 COMMENT '1=启用, 0=停用',
    version         INT          NOT NULL DEFAULT 1 COMMENT '版本号',
    created_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (template_id),
    KEY idx_tenant (tenant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='卡片模板表';

-- 6.2 系统配置表
CREATE TABLE IF NOT EXISTS system_config (
    config_id    BIGINT       NOT NULL AUTO_INCREMENT,
    tenant_id    BIGINT       NOT NULL COMMENT '租户ID',
    config_key   VARCHAR(100) NOT NULL COMMENT '配置键',
    config_value TEXT         NOT NULL COMMENT '配置值',
    description  VARCHAR(200) DEFAULT NULL COMMENT '描述',
    created_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (config_id),
    UNIQUE KEY uk_tenant_key (tenant_id, config_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='系统配置表';
