-- 000041: 应用图标统一字段支持系统图标配置或自定义文件地址 JSON。
ALTER TABLE applications ALTER COLUMN icon TYPE varchar(512);
COMMENT ON COLUMN applications.icon IS '应用图标：兼容历史稳定键；新增值为 JSON（remix: type/name/background，custom: type/name 文件内容地址）';
