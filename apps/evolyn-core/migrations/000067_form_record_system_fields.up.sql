-- 000067：表单记录系统字段（数据页「提交人/提交时间/更新时间」数据源）。
-- submitted_by_name 为提交时固化的展示名快照（成员改名/退出后历史展示不失真，
-- 与企业日志 actor_name_snapshot 口径一致）；updated_at 记录最后写回时间
-- （提交时=提交时间，审批编辑经 WorkflowRecordStore 写回 values 时同事务刷新）。
ALTER TABLE tn_form_records ADD COLUMN submitted_by_name varchar(100);
ALTER TABLE tn_form_records ADD COLUMN updated_at timestamp with time zone;

COMMENT ON COLUMN tn_form_records.submitted_by_name IS '提交人展示名快照（提交时按租户内昵称固化，昵称空回落账号昵称/登录名；存量与未命中行回填固定文案）';
COMMENT ON COLUMN tn_form_records.updated_at IS '记录最后更新时间（提交时等于提交时间；审批编辑写回时同事务刷新）';

-- 存量回填：提交人按「成员昵称 → 账号昵称 → 账号登录名」回落。软删成员仍回填
-- 真名（软删只是租户内不可见，历史提交人展示不失真）；物理清理后未命中的行由
-- 下一条兜底为固定文案。updated_at 一律回填为提交时间（存量行没有写回审计）。
UPDATE tn_form_records r
SET submitted_by_name = COALESCE(NULLIF(u.nickname, ''), NULLIF(a.nickname, ''), a.name, '已退出成员'),
    updated_at = r.submitted_at
FROM tn_users u
LEFT JOIN pf_accounts a ON a.id = u.account_id
WHERE u.id = r.submitted_by_member_id;

-- 未命中兜底：成员行不存在（物理清理）或提交时间为空的极端历史行。
UPDATE tn_form_records
SET submitted_by_name = COALESCE(submitted_by_name, '已退出成员'),
    updated_at = COALESCE(updated_at, submitted_at, created_at, LOCALTIMESTAMP);

ALTER TABLE tn_form_records ALTER COLUMN updated_at SET NOT NULL;
