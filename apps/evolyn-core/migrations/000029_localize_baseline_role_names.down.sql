-- 回滚默认角色中文展示名，恢复至此前版本依赖的英文名称。
UPDATE roles
SET name = CASE name
    WHEN '平台管理员' THEN 'cluster-admin'
    WHEN '租户管理员' THEN 'tenant-admin'
    WHEN '已认证用户' THEN 'authenticated'
    WHEN '未认证用户' THEN 'unauthenticated'
END
WHERE name IN ('平台管理员', '租户管理员', '已认证用户', '未认证用户');
