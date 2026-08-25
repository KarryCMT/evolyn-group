-- 头像由外部 URL 扩展为浏览器裁剪后的 data URL；原 varchar(256) 无法容纳压缩后的头像数据。
ALTER TABLE accounts ALTER COLUMN avatar TYPE text;

COMMENT ON COLUMN accounts.avatar IS '头像 URL 或裁剪压缩后的 data URL';
