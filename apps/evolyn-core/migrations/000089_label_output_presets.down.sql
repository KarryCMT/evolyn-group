ALTER TABLE tn_label_render_tasks DROP CONSTRAINT IF EXISTS ck_tn_label_render_tasks_output_size;
ALTER TABLE tn_label_render_tasks
    DROP COLUMN IF EXISTS output_dpi,
    DROP COLUMN IF EXISTS output_unit,
    DROP COLUMN IF EXISTS output_height,
    DROP COLUMN IF EXISTS output_width,
    DROP COLUMN IF EXISTS output_preset_id;
