-- 回滚 000061：恢复 000059 执行后的列名与账号 ID 值，供迁移回退工具使用。

ALTER TABLE files RENAME COLUMN creator_member_id TO creator_id;

ALTER TABLE role_groups RENAME COLUMN creator_member_id TO creator_id;
UPDATE role_groups AS rg
SET creator_id = u.account_id
FROM users AS u
WHERE rg.creator_id IS NOT NULL
  AND rg.creator_id <> 0
  AND u.id = rg.creator_id
  AND u.tenant_id = rg.tenant_id;
