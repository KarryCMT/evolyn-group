-- 000045: 表单公开编码；外部路由与 API 不再暴露数据库自增主键。
ALTER TABLE forms ADD COLUMN code varchar(64);

-- 存量资产以租户、主键和创建时间生成稳定编码；迁移完成后编码不可变。
UPDATE forms
SET code = 'form_' || substr(md5(tenant_id::text || ':' || id::text || ':' || COALESCE(created_at::text, '')), 1, 16)
WHERE code IS NULL;

ALTER TABLE forms ALTER COLUMN code SET NOT NULL;

CREATE UNIQUE INDEX uk_forms_tenant_code
    ON forms (tenant_id, code)
    WHERE deleted_at IS NULL;

COMMENT ON COLUMN forms.code IS '表单稳定公开编码（form_ 前缀）；路由、API 与菜单 target 使用，禁止暴露自增主键';
