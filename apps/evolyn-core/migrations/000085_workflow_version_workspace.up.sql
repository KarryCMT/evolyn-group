-- 000085: 流程设计器版本工作区。
-- 一个流程定义同一时刻至多存在一个可编辑版本；启用后该版本冻结为不可变
-- 快照，后续修改必须显式“添加新版本”，避免直接覆盖运行中的版本。
ALTER TABLE wf_definition
    ADD COLUMN IF NOT EXISTS draft_version_no INTEGER NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS has_draft BOOLEAN NOT NULL DEFAULT TRUE;

-- 存量定义若草稿与当前启用快照一致，则收口为只读启用版本；若内容不同，
-- 保留为下一设计版本，避免迁移吞掉尚未发布的编辑结果。
UPDATE wf_definition AS d
SET has_draft = CASE
        WHEN d.published_version = 0 THEN TRUE
        WHEN v.id IS NULL THEN TRUE
        ELSE d.draft_content IS DISTINCT FROM v.dsl_snapshot
    END,
    draft_version_no = CASE
        WHEN d.published_version = 0 THEN 1
        WHEN v.id IS NOT NULL AND d.draft_content IS NOT DISTINCT FROM v.dsl_snapshot
            THEN d.published_version
        ELSE d.published_version + 1
    END
FROM (SELECT id, dsl_snapshot FROM wf_definition_version) AS v
WHERE d.latest_version_id = v.id;

-- latest_version_id 为空或历史脏数据缺少对应快照时，保守保留为设计版本。
UPDATE wf_definition
SET has_draft = TRUE,
    draft_version_no = GREATEST(published_version + 1, 1)
WHERE latest_version_id IS NULL
   OR NOT EXISTS (
        SELECT 1
        FROM wf_definition_version AS v
        WHERE v.id = wf_definition.latest_version_id
   )
   OR draft_version_no < 1;

ALTER TABLE wf_definition
    ADD CONSTRAINT chk_wf_definition_version_workspace
    CHECK (
        draft_version_no >= 1
        AND (
            (has_draft AND draft_version_no > published_version)
            OR
            (NOT has_draft AND published_version > 0 AND draft_version_no = published_version)
        )
    );

COMMENT ON COLUMN wf_definition.draft_version_no IS '当前工作区版本号：has_draft=true 时为待启用设计版本；false 时等于当前启用版本';
COMMENT ON COLUMN wf_definition.has_draft IS '是否存在可编辑设计版本；启用成功后置 false，必须显式添加新版本后才能继续编辑';
