-- 000019 down: 按依赖逆序删除（子表 → 独立表）。settings 最后删，
-- 与 up 的创建顺序对称。
DROP TABLE IF EXISTS account_security_events;
DROP TABLE IF EXISTS account_sessions;
DROP TABLE IF EXISTS account_mfa_recovery_codes;
DROP TABLE IF EXISTS account_mfa_factors;
DROP TABLE IF EXISTS account_security_settings;
