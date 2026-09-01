-- 000062: 为 000061 恢复成员归属列后的 files、role_groups 补齐账号审计字段。
-- 两张表此前的 creator_id 分别承载上传成员和角色组创建成员，不能复用为
-- accounts.id；成员归属列改名后，独立补建通用 creator_id 审计列。

ALTER TABLE files ADD COLUMN IF NOT EXISTS creator_id BIGINT;
ALTER TABLE role_groups ADD COLUMN IF NOT EXISTS creator_id BIGINT;

COMMENT ON COLUMN files.creator_id IS '创建账号 ID（accounts.id）；NULL 表示系统或无认证操作';
COMMENT ON COLUMN role_groups.creator_id IS '创建账号 ID（accounts.id）；NULL 表示系统或无认证操作';
