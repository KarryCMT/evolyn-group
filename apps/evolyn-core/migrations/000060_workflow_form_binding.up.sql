-- 000060: 流程定义与表单资产的绑定列（Phase 9 流程设计器，ADR-012）。
-- 语义：一条流程型表单（forms.form_type='workflow'，000044 固化）至多绑定
-- 一条流程定义——表单工作区「流程设计」页以此为定位口令加载/懒建定义；
-- 空串 = 未绑定的独立定义（租户级流程库，后续审批中心扩展使用）。
-- 绑定不引入跨域外键（流程域不依赖表单域 Schema），仅按 code 字符串关联，
-- 与实例发起 StartInstanceRequest.FormCode 同口径。

ALTER TABLE wf_definition
    ADD COLUMN IF NOT EXISTS form_code varchar(64) NOT NULL DEFAULT '';

-- 租户内未删除行的绑定唯一：一条表单只挂一条有效流程定义。
-- 部分唯一索引（form_code <> ''）允许多条独立定义共存。
CREATE UNIQUE INDEX IF NOT EXISTS uk_wf_definition_form_code
    ON wf_definition (tenant_id, form_code)
    WHERE deleted_at IS NULL AND form_code <> '';

-- 按绑定表单定位定义（设计器进入时精确查询）
CREATE INDEX IF NOT EXISTS idx_wf_definition_form_code
    ON wf_definition (tenant_id, form_code)
    WHERE deleted_at IS NULL AND form_code <> '';

COMMENT ON COLUMN wf_definition.form_code IS '绑定的表单公开编码（form_ 前缀，forms.code）：流程型表单的工作区流程设计页定位口令，一条表单租户内至多绑定一条未删除定义；空串=独立定义（不属于任何表单）';
