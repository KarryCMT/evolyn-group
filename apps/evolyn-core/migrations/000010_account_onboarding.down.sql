-- 回滚 000010：移除账号注册引导画像列
ALTER TABLE accounts
    DROP COLUMN IF EXISTS onboarding;
