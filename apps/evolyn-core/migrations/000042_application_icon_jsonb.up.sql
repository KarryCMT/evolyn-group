-- 000042: 应用图标改为 JSONB，直接存储 type/name/background 对象。
ALTER TABLE applications
  ALTER COLUMN icon DROP DEFAULT,
  ALTER COLUMN icon TYPE jsonb USING icon::jsonb,
  ALTER COLUMN icon SET DEFAULT '{"type":"remix","name":"bookmark","background":"#f7be54, #eda426"}'::jsonb;
COMMENT ON COLUMN applications.icon IS '应用图标 JSONB：remix 为 type/name/background，自定义图标为 type/name';
