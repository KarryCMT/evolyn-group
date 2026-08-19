-- 000010: 账号级注册引导画像（注册向导第 3 步「完善信息」）
-- 角色（你的角色）与了解渠道是「人」的属性而非租户属性，挂账号 JSONB：
--   {"role": "...", "channel": "..."}
-- 租户级画像（需求/行业/管理需求）在 tenants.config 的 onboarding 段，二者互补。
ALTER TABLE accounts
    ADD COLUMN IF NOT EXISTS onboarding jsonb NOT NULL DEFAULT '{}';

COMMENT ON COLUMN accounts.onboarding IS '账号注册引导画像：role 角色 / channel 了解渠道（注册向导第 3 步采集）';
