-- 回滚 000046：删除个人收藏表与 hidden 列（对称回滚）。
DROP INDEX IF EXISTS idx_application_menu_favorites_member;
DROP TABLE IF EXISTS application_menu_favorites;

ALTER TABLE application_menu_entries
    DROP COLUMN IF EXISTS hidden;

-- 恢复 000044 的 form_type 列注释（up 已放宽为可切换语义，ADR-011）
COMMENT ON COLUMN forms.form_type IS '表单类型：standard 标准表单 / workflow 流程表单；创建后不可变，设计器能力以此字段为准';
