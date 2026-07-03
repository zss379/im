-- ============================================================
-- 数莲 PaaS IM 系统 — 种子数据初始化
-- 包含默认租户、系统机器人、默认风控规则
-- ============================================================

USE im_shared;

-- ============================================================
-- 1. 默认租户
-- ============================================================
INSERT INTO tenant (tenant_id, tenant_name, domain, status) VALUES
    (1, '数莲科技', 'shulian.com', 1)
ON DUPLICATE KEY UPDATE tenant_name = VALUES(tenant_name);

-- ============================================================
-- 2. 系统机器人（3个内置，不可删除）
-- ============================================================
INSERT INTO bot (bot_id, tenant_id, bot_type, bot_name, description, status) VALUES
    (1, 1, 1, '消息通知助手', '系统公告、版本通知推送', 1),
    (2, 1, 1, '流程助手', '审批、流程超时提醒', 1),
    (3, 1, 1, '文件助手', '文件上传下载通知', 1)
ON DUPLICATE KEY UPDATE bot_name = VALUES(bot_name);

-- ============================================================
-- 3. 默认频控规则
-- ============================================================
INSERT INTO frequency_control_rule (rule_id, tenant_id, target_type, max_count, time_window_seconds, action, status) VALUES
    (1, 1, 1, 5,   1,  1, 1),   -- 用户: 1秒最多5条，限流429
    (2, 1, 2, 10,  1,  1, 1)    -- 机器人: 1秒最多10条，限流429
ON DUPLICATE KEY UPDATE max_count = VALUES(max_count);

-- ============================================================
-- 4. 默认文件上传限制
-- ============================================================
INSERT INTO file_limit_config (config_id, tenant_id, file_type, max_size_mb, allowed_extensions, status) VALUES
    (1, 1, 'image',    10,  'jpg,jpeg,png,gif,webp',           1),
    (2, 1, 'video',    100, 'mp4,mov,avi,mkv',                 1),
    (3, 1, 'document', 50,  'pdf,doc,docx,xls,xlsx,ppt,pptx,txt,zip,rar', 1)
ON DUPLICATE KEY UPDATE max_size_mb = VALUES(max_size_mb);

-- ============================================================
-- 5. 示例敏感词（开发测试用）
-- ============================================================
INSERT INTO sensitive_word (tenant_id, word, strategy, replacement, category, status) VALUES
    (1, '赌博',     2, '***', '违法', 1),
    (1, '色情',     2, '***', '违法', 1),
    (1, '政治敏感词', 2, '***', '政治',  1)
ON DUPLICATE KEY UPDATE word = VALUES(word);
