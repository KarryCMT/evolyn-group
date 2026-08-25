-- 回滚前仅允许保留原 varchar(256) 能容纳的头像，避免 PostgreSQL 静默截断数据。
ALTER TABLE accounts
    ALTER COLUMN avatar TYPE varchar(256)
    USING CASE
        WHEN avatar IS NULL OR length(avatar) <= 256 THEN avatar
        ELSE NULL
    END;

COMMENT ON COLUMN accounts.avatar IS '头像 URL';
