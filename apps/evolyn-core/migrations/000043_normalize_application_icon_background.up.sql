-- 000043: 统一 Remix 图标渐变色格式，逗号前后不保留空格。
UPDATE applications
SET icon = jsonb_set(
  icon,
  '{background}',
  to_jsonb(regexp_replace(icon->>'background', '\s*,\s*', ',', 'g')),
  true
)
WHERE icon->>'type' = 'remix' AND icon ? 'background';

ALTER TABLE applications
  ALTER COLUMN icon SET DEFAULT '{"type":"remix","name":"bookmark","background":"#f7be54,#eda426"}'::jsonb;
