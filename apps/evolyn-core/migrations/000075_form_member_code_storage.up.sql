-- 000075: 成员控件物理列由内部 BIGINT ID 切换为全局 member_code 文本。
-- 已有记录的数字 ID 先转文本保留；读取侧继续兼容并批量解析，后续写入统一
-- 使用 mb_ 前缀的成员编号。动态表名/列名均来自已应用的 StorageModel，使用
-- format('%I') 引用，避免把元数据当作裸 SQL 标识符拼接。
DO $$
DECLARE
    storage_record RECORD;
    child_record RECORD;
    column_record RECORD;
BEGIN
    FOR storage_record IN
        SELECT s.table_name, v.model
        FROM tn_form_storages s
        JOIN tn_form_storage_schema_versions v ON v.id = s.applied_schema_version_id
        WHERE s.backend = 'PHYSICAL' AND v.state = 'APPLIED'
    LOOP
        FOR column_record IN
            SELECT value
            FROM jsonb_array_elements(COALESCE(storage_record.model->'columns', '[]'::jsonb))
            WHERE value->>'widgetType' = 'user'
              AND value->>'kind' = 'ref'
              -- 早期试验性模型可能没有 v8 fieldId，无法安全定位动态列；它们
              -- 不进入物理发布链路，保留给现有历史兼容读取，禁止猜列名。
              AND NULLIF(value->>'fieldId', '') IS NOT NULL
        LOOP
            EXECUTE format(
                'ALTER TABLE %I ALTER COLUMN %I TYPE TEXT USING %I::TEXT',
                storage_record.table_name,
                'f_' || lower(column_record.value->>'fieldId'),
                'f_' || lower(column_record.value->>'fieldId')
            );
        END LOOP;

        FOR child_record IN
            SELECT value
            FROM jsonb_array_elements(COALESCE(storage_record.model->'children', '[]'::jsonb))
        LOOP
            FOR column_record IN
                SELECT value
                FROM jsonb_array_elements(COALESCE(child_record.value->'columns', '[]'::jsonb))
                WHERE value->>'widgetType' = 'user'
                  AND value->>'kind' = 'ref'
                  AND NULLIF(value->>'fieldId', '') IS NOT NULL
            LOOP
                EXECUTE format(
                    'ALTER TABLE %I ALTER COLUMN %I TYPE TEXT USING %I::TEXT',
                    child_record.value->>'table',
                    'f_' || lower(column_record.value->>'fieldId'),
                    'f_' || lower(column_record.value->>'fieldId')
                );
            END LOOP;
        END LOOP;
    END LOOP;
END $$;

-- 物理 DML 读取 applied SchemaVersion.model；必须同步更新已应用元数据，否则
-- 新 member_code 会在编码器中仍按数字引用处理。只更新 APPLIED 版本：待执行
-- DDL 的计划与 checksum 保持同一事实源，交由下一次发布按新模型重建。
UPDATE tn_form_storage_schema_versions schema_version
SET model = jsonb_set(
    jsonb_set(
        schema_version.model,
        '{columns}',
        COALESCE((
            SELECT jsonb_agg(
                CASE
                    WHEN column_value->>'widgetType' = 'user'
                      AND column_value->>'kind' = 'ref'
                      AND NULLIF(column_value->>'fieldId', '') IS NOT NULL
                    THEN jsonb_set(jsonb_set(column_value, '{kind}', '"member"'::jsonb), '{type}', '"TEXT"'::jsonb)
                    ELSE column_value
                END
                ORDER BY ordinal
            )
            FROM jsonb_array_elements(COALESCE(schema_version.model->'columns', '[]'::jsonb)) WITH ORDINALITY AS columns(column_value, ordinal)
        ), '[]'::jsonb),
        true
    ),
    '{children}',
    COALESCE((
        SELECT jsonb_agg(
            jsonb_set(
                child_value,
                '{columns}',
                COALESCE((
                    SELECT jsonb_agg(
                        CASE
                            WHEN child_column_value->>'widgetType' = 'user'
                              AND child_column_value->>'kind' = 'ref'
                              AND NULLIF(child_column_value->>'fieldId', '') IS NOT NULL
                            THEN jsonb_set(jsonb_set(child_column_value, '{kind}', '"member"'::jsonb), '{type}', '"TEXT"'::jsonb)
                            ELSE child_column_value
                        END
                        ORDER BY child_ordinal
                    )
                    FROM jsonb_array_elements(COALESCE(child_value->'columns', '[]'::jsonb)) WITH ORDINALITY AS child_columns(child_column_value, child_ordinal)
                ), '[]'::jsonb),
                true
            )
            ORDER BY ordinal
        )
        FROM jsonb_array_elements(COALESCE(schema_version.model->'children', '[]'::jsonb)) WITH ORDINALITY AS children(child_value, ordinal)
    ), '[]'::jsonb),
    true
), updated_at = LOCALTIMESTAMP
WHERE schema_version.state = 'APPLIED'
  AND EXISTS (
      SELECT 1 FROM tn_form_storages storage
      WHERE storage.applied_schema_version_id = schema_version.id AND storage.backend = 'PHYSICAL'
  );
