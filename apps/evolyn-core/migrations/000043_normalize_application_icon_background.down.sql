-- 回滚默认图标格式；已有图标值保持不变。
ALTER TABLE applications
  ALTER COLUMN icon SET DEFAULT '{"type":"remix","name":"bookmark","background":"#f7be54,#eda426"}'::jsonb;
