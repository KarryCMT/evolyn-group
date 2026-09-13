-- 新成员编号已写入 TEXT 列，无法无损还原为旧的 BIGINT 自增成员 ID。
DO $$ BEGIN
    RAISE EXCEPTION '000075 cannot be rolled back after member_code values are written';
END $$;
