-- 000042: 应用图标改为 JSONB，直接存储 type/name/background 对象。
-- 000041 只放宽了字段长度，升级到本版本时存量数据仍可能是旧版稳定图标键；
-- 先将这些键转换为对象，避免直接 ::jsonb 因 bookmark 等纯文本值而中断迁移。
ALTER TABLE applications
  ALTER COLUMN icon DROP DEFAULT,
  ALTER COLUMN icon TYPE jsonb USING (
    CASE
      WHEN icon IN ('bookmark', 'briefcase', 'contacts', 'chart', 'check') THEN
        jsonb_build_object(
          'type', 'remix',
          'name', icon,
          'background', '#f7be54,#eda426'
        )
      ELSE icon::jsonb
    END
  ),
  ALTER COLUMN icon SET DEFAULT '{"type":"remix","name":"bookmark","background":"#f7be54, #eda426"}'::jsonb;
COMMENT ON COLUMN applications.icon IS '应用图标 JSONB：remix 为 type/name/background，自定义图标为 type/name';
