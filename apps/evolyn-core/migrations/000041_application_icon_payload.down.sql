-- 回滚时无法容纳的新图标配置回退为默认书签图标。
ALTER TABLE applications ALTER COLUMN icon TYPE varchar(32) USING CASE WHEN char_length(icon) <= 32 THEN icon ELSE 'bookmark' END;
COMMENT ON COLUMN applications.icon IS '稳定图标键（bookmark/briefcase/contacts/chart/check），不存前端组件名';
