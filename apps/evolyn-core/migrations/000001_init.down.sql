-- 初始基线回滚：按依赖逆序删除全部表与数据（不可逆，仅空环境调试用）
DROP TABLE IF EXISTS group_roles;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS resources;
DROP TABLE IF EXISTS user_groups;
DROP TABLE IF EXISTS groups;
DROP TABLE IF EXISTS department_users;
DROP TABLE IF EXISTS departments;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS auth_infos;
DROP TABLE IF EXISTS accounts;
DROP TABLE IF EXISTS tenants;
