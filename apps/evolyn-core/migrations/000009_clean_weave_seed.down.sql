-- 回滚 000009：种子邮箱与系统组描述还原为 weave 时代文案。
-- 仅命中 @evolyn.com 邮箱域与订正后的系统组描述，业务数据不含该域时无操作。

UPDATE accounts SET email = replace(email, '@evolyn.com', '@weave.com'),
                    updated_at = LOCALTIMESTAMP
WHERE email LIKE '%@evolyn.com';

UPDATE groups SET "describe" = 'weave system group', updated_at = LOCALTIMESTAMP
WHERE kind = 'system' AND "describe" = 'evolyn system group';
