-- 清理 weave 时代种子数据残留：admin/demo 种子账号邮箱与 root 系统组描述
-- 仍带旧品牌字样。000001 已应用、受 schema_migrations checksum 防篡改锁定，
-- 种子订正只能走增量迁移（仅改文案，不动结构与登录口令）。

UPDATE accounts SET email = replace(email, '@weave.com', '@evolyn.com'),
                    updated_at = LOCALTIMESTAMP
WHERE email LIKE '%@weave.com';

UPDATE groups SET "describe" = 'evolyn system group', updated_at = LOCALTIMESTAMP
WHERE kind = 'system' AND "describe" = 'weave system group';
