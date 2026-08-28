-- 回滚为 JSON 文本字段。
ALTER TABLE applications
  ALTER COLUMN icon TYPE varchar(512) USING icon::text,
  ALTER COLUMN icon SET DEFAULT '{"type":"remix","name":"bookmark","background":"#f7be54, #eda426"}';
COMMENT ON COLUMN applications.icon IS '应用图标 JSON 文本';
