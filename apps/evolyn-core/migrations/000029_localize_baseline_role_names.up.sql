-- 默认角色名称直接展示在组织角色页，统一改为中文。
-- 角色权限仍由 rules 字段决定；仅改名，不调整既有绑定关系和授权规则。
UPDATE roles
SET name = CASE name
    WHEN 'cluster-admin' THEN '平台管理员'
    WHEN 'tenant-admin' THEN '租户管理员'
    WHEN 'authenticated' THEN '已认证用户'
    WHEN 'unauthenticated' THEN '未认证用户'
END
WHERE name IN ('cluster-admin', 'tenant-admin', 'authenticated', 'unauthenticated');
