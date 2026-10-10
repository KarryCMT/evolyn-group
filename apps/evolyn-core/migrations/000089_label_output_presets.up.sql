-- 000089: 批量标签任务冻结发布尺寸，排队和重试不受模板后续发布影响。
ALTER TABLE tn_label_render_tasks
    ADD COLUMN IF NOT EXISTS output_preset_id VARCHAR(32),
    ADD COLUMN IF NOT EXISTS output_width NUMERIC(12,4),
    ADD COLUMN IF NOT EXISTS output_height NUMERIC(12,4),
    ADD COLUMN IF NOT EXISTS output_unit VARCHAR(8),
    ADD COLUMN IF NOT EXISTS output_dpi INTEGER;

ALTER TABLE tn_label_render_tasks
    ADD CONSTRAINT ck_tn_label_render_tasks_output_size
    CHECK (
        (output_preset_id IS NULL AND output_width IS NULL AND output_height IS NULL AND output_unit IS NULL AND output_dpi IS NULL)
        OR
        (output_preset_id IS NOT NULL AND output_preset_id <> '' AND output_width > 0 AND output_height > 0
            AND output_unit IN ('mm', 'px') AND output_dpi IN (96, 203, 300, 600))
    );

COMMENT ON COLUMN tn_label_render_tasks.output_preset_id IS '创建任务时冻结的发布输出预设 ID；NULL 仅表示 000089 前历史任务';
