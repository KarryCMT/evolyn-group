-- 000021 down：首页形态仅为应用元数据，回滚时移除约束及字段。
ALTER TABLE applications
    DROP CONSTRAINT IF EXISTS chk_applications_home_mode,
    DROP COLUMN IF EXISTS home_mode;
