-- 000021: 应用首页形态。该状态属于应用自身，而非当前成员菜单可见性的派生值：
-- builder 显示首次构建引导；application 直接进入运行时应用首页。
ALTER TABLE applications
    ADD COLUMN home_mode varchar(16) NOT NULL DEFAULT 'builder',
    ADD CONSTRAINT chk_applications_home_mode
        CHECK (home_mode IN ('builder', 'application'));

-- 已有任一未删除菜单节点的存量应用迁移为运行时首页。此处只做一次历史
-- 数据归类；后续菜单发布/导入流程须与菜单变更同事务维护 home_mode。
UPDATE applications AS a
SET home_mode = 'application'
WHERE EXISTS (
    SELECT 1
    FROM application_menu_entries AS e
    WHERE e.application_id = a.id
      AND e.deleted_at IS NULL
);

COMMENT ON COLUMN applications.home_mode IS '应用首页形态：builder 显示首次构建引导 / application 进入运行时应用首页；由应用生命周期维护，不按当前成员可见菜单数量推导';
