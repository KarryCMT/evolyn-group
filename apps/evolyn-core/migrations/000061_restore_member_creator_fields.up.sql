-- 000061: 恢复 files 与 role_groups 的成员归属字段语义。
-- 000059 已在部分环境执行，不能重写；该版本显式修正其将两张表的
-- creator_id 统一为账号审计字段所造成的列语义冲突。

-- files.creator_id 一直是上传成员归属，不属于通用账号审计字段。
ALTER TABLE files RENAME COLUMN creator_id TO creator_member_id;

-- role_groups.creator_id 在 000059 中已被转换为 accounts.id。先按租户
-- 映射回 users.id，再恢复为成员归属列；无法映射的历史账号使用旧哨兵值 0。
UPDATE role_groups AS rg
SET creator_id = COALESCE((
    SELECT u.id
    FROM users AS u
    WHERE u.account_id = rg.creator_id
      AND u.tenant_id = rg.tenant_id
    ORDER BY u.id DESC
    LIMIT 1
), 0)
WHERE rg.creator_id IS NOT NULL
  AND rg.creator_id <> 0;

ALTER TABLE role_groups RENAME COLUMN creator_id TO creator_member_id;

COMMENT ON COLUMN files.creator_member_id IS '文件归属成员 ID（users.id），用于上传者访问边界';
COMMENT ON COLUMN role_groups.creator_member_id IS '创建角色组的成员 ID（users.id）';
