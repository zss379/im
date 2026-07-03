-- ============================================================
-- 数莲 PaaS IM 系统 — v0.2 机器人 @触发 功能迁移
-- 目标版本: v0.1 → v0.2
-- 执行方式: mysql -u root -p im_shared < v0.2-bot-trigger.sql
-- ============================================================

USE im_shared;

-- ============================================================
-- 1. bot 表新增字段
-- ============================================================
ALTER TABLE bot
    ADD COLUMN openim_user_id BIGINT       DEFAULT NULL COMMENT 'OpenIM用户ID' AFTER api_key,
    ADD COLUMN response_mode  TINYINT      NOT NULL DEFAULT 1 COMMENT '1=sync, 2=async, 3=sse' AFTER api_key,
    ADD COLUMN callback_url   VARCHAR(500) DEFAULT NULL COMMENT '异步回调接收URL' AFTER response_mode,
    ADD COLUMN ip_whitelist   JSON         DEFAULT NULL COMMENT 'IP白名单' AFTER callback_url,
    ADD INDEX idx_openim_user (openim_user_id);

-- ============================================================
-- 2. group_bot 表新增字段
-- ============================================================
ALTER TABLE group_bot
    ADD COLUMN response_mode  TINYINT      NOT NULL DEFAULT 1 COMMENT '1=sync, 2=async, 3=sse' AFTER token,
    ADD COLUMN callback_url   VARCHAR(500) DEFAULT NULL COMMENT '异步回调接收URL' AFTER response_mode;

-- ============================================================
-- 3. 频控规则更新：确认机器人频控已配置（目标类型2）
-- ============================================================
INSERT INTO frequency_control_rule (tenant_id, target_type, max_count, time_window_seconds, action, status)
VALUES (1, 2, 10, 1, 1, 1)
ON DUPLICATE KEY UPDATE max_count = VALUES(max_count);
