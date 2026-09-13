-- 000074: 成员公开编号。内部自增 ID 继续承担关系表主键；member_code 面向
-- 多数据库/分片场景的稳定引用，供表单成员字段和成员卡片等外部协议使用。
ALTER TABLE tn_users ADD COLUMN IF NOT EXISTS member_code varchar(40);

-- 存量成员回填随机 128-bit 编号。新成员由应用层使用 UUID 生成；本迁移只负责
-- 旧数据无空值地进入新约束，不复用可编辑且仅租户内唯一的企业编号。
UPDATE tn_users
SET member_code = 'mb_' || md5(random()::text || clock_timestamp()::text || id::text || tenant_id::text)
WHERE member_code IS NULL OR member_code = '';

ALTER TABLE tn_users ALTER COLUMN member_code SET NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uk_tn_users_member_code ON tn_users (member_code);

COMMENT ON COLUMN tn_users.member_code IS '成员关系全局不可变公开编号：跨数据库稳定，内部关联仍使用 id';
