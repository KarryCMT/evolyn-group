-- 补齐全部业务表的表注释与字段注释（COMMENT ON）。
-- 只写元数据不改结构，天然幂等；注释语义与 internal 各域 GORM 模型
-- 及 ADR-006（账号×成员拆分）/FIX-001~017 整改结论保持一致。

-- ============ 平台账号 ============
COMMENT ON TABLE accounts IS '平台账号（登录身份）：登录名/手机号/密码与第三方凭证挂账号；账号跨租户，无 tenant_id（ADR-006/FIX-014）';
COMMENT ON COLUMN accounts.id IS '自增主键';
COMMENT ON COLUMN accounts.name IS '登录名，全局唯一';
COMMENT ON COLUMN accounts.nickname IS '平台级展示昵称；租户内昵称见 users.nickname';
COMMENT ON COLUMN accounts.phone IS '手机号，非空时全局唯一（部分唯一索引 uk_accounts_phone，未填写落空串不参与唯一）';
COMMENT ON COLUMN accounts.email IS '邮箱';
COMMENT ON COLUMN accounts.password IS '登录密码（bcrypt 摘要）；纯 OAuth 账号可为空';
COMMENT ON COLUMN accounts.avatar IS '头像 URL';
COMMENT ON COLUMN accounts.created_at IS '创建时间';
COMMENT ON COLUMN accounts.updated_at IS '更新时间';
COMMENT ON COLUMN accounts.deleted_at IS '软删除时间，NULL=未删除';

-- ============ 租户 ============
COMMENT ON TABLE tenants IS '租户（平台一级资源）：套餐/状态/Owner/租户级配置与注销生命周期；无 tenant_id（FIX-014）';
COMMENT ON COLUMN tenants.id IS '自增主键';
COMMENT ON COLUMN tenants.code IS '租户编码，全局唯一，登录时识别目标租户';
COMMENT ON COLUMN tenants.name IS '租户名称';
COMMENT ON COLUMN tenants.plan IS '套餐标识（free 等，套餐定义在租户域代码 plan.go，暂无套餐表）';
COMMENT ON COLUMN tenants.status IS '租户状态：active=正常 / frozen=冻结 / deleted=已注销';
COMMENT ON COLUMN tenants.owner_account_id IS '开通者（Owner）平台账号 ID，NULL=未设置；外键指向 accounts(id)（FIX-016）';
COMMENT ON COLUMN tenants.config IS '租户级配置 JSONB：水印/品牌主题/时区/语言';
COMMENT ON COLUMN tenants.quotas IS '套餐配额覆盖 JSONB；空对象表示使用套餐默认值';
COMMENT ON COLUMN tenants.delete_requested_at IS '注销申请时间（FIX-012 生命周期时间线）';
COMMENT ON COLUMN tenants.retention_until IS '数据保留截止时间，到期由 Purge Worker 执行清理';
COMMENT ON COLUMN tenants.purged_at IS '最终清理完成时间（墓碑标记）';
COMMENT ON COLUMN tenants.created_at IS '创建时间';
COMMENT ON COLUMN tenants.updated_at IS '更新时间';
COMMENT ON COLUMN tenants.deleted_at IS '软删除时间，NULL=未删除';

-- ============ 第三方登录凭证 ============
COMMENT ON TABLE auth_infos IS '第三方登录凭证（OAuth）：归属平台账号，无租户归属（ADR-006）';
COMMENT ON COLUMN auth_infos.id IS '自增主键';
COMMENT ON COLUMN auth_infos.account_id IS '归属平台账号 ID，外键指向 accounts(id)；存量回填前可能为 0';
COMMENT ON COLUMN auth_infos.url IS '第三方资料页地址';
COMMENT ON COLUMN auth_infos.auth_type IS '第三方渠道类型：github / wechat';
COMMENT ON COLUMN auth_infos.auth_id IS '第三方用户唯一标识；与 auth_type 组成软删友好唯一键（FIX-017）';
COMMENT ON COLUMN auth_infos.access_token IS '访问令牌';
COMMENT ON COLUMN auth_infos.refresh_token IS '刷新令牌';
COMMENT ON COLUMN auth_infos.expiry IS '令牌过期时间';
COMMENT ON COLUMN auth_infos.created_at IS '创建时间';
COMMENT ON COLUMN auth_infos.updated_at IS '更新时间';
COMMENT ON COLUMN auth_infos.deleted_at IS '软删除时间，NULL=未删除';

-- ============ 租户成员 ============
COMMENT ON TABLE users IS '租户成员（租户内身份）：归属账号+租户内昵称；部门/分组/角色关系见各关联表（ADR-006）';
COMMENT ON COLUMN users.id IS '自增主键';
COMMENT ON COLUMN users.account_id IS '归属平台账号 ID，外键指向 accounts(id)；同一账号在每租户至多一条有效成员关系（FIX-004）';
COMMENT ON COLUMN users.nickname IS '租户内展示昵称；空则前端回落账号昵称';
COMMENT ON COLUMN users.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN users.created_at IS '创建时间';
COMMENT ON COLUMN users.updated_at IS '更新时间';
COMMENT ON COLUMN users.deleted_at IS '软删除时间，NULL=未删除';

-- ============ 部门 ============
COMMENT ON TABLE departments IS '部门（租户内组织架构，邻接表树）：承载组织结构与汇报关系，权限授权走分组/角色';
COMMENT ON COLUMN departments.id IS '自增主键';
COMMENT ON COLUMN departments.parent_id IS '父部门 ID，NULL=根节点；自引用外键，跨租户父节点被 FK 与服务层共同拦截（FIX-015）';
COMMENT ON COLUMN departments.name IS '部门名称';
COMMENT ON COLUMN departments."order" IS '同级排序值，小者在前';
COMMENT ON COLUMN departments.status IS '部门状态：active=正常 / disabled=停用';
COMMENT ON COLUMN departments.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN departments.created_at IS '创建时间';
COMMENT ON COLUMN departments.updated_at IS '更新时间';
COMMENT ON COLUMN departments.deleted_at IS '软删除时间，NULL=未删除';

COMMENT ON TABLE department_users IS '部门-成员关联表（多对多），联合主键无独立生命周期';
COMMENT ON COLUMN department_users.department_id IS '部门 ID，联合主键之一，外键指向 departments(id)';
COMMENT ON COLUMN department_users.user_id IS '成员 ID，联合主键之一，外键指向 users(id)';

-- ============ 用户分组 ============
COMMENT ON TABLE groups IS '用户分组（租户内权限分组）：系统组（root/system:*）与自定义组，角色可挂分组间接授权成员';
COMMENT ON COLUMN groups.id IS '自增主键';
COMMENT ON COLUMN groups.name IS '分组名，租户内唯一（软删友好部分唯一索引 uk_groups_tenant_name，FIX-003）';
COMMENT ON COLUMN groups.kind IS '分组类型：system=系统组 / custom=自定义组';
COMMENT ON COLUMN groups.describe IS '分组描述（列名沿用历史拼写）';
COMMENT ON COLUMN groups.creator_id IS '创建人成员 ID';
COMMENT ON COLUMN groups.updater_id IS '最后更新人成员 ID';
COMMENT ON COLUMN groups.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN groups.created_at IS '创建时间';
COMMENT ON COLUMN groups.updated_at IS '更新时间';
COMMENT ON COLUMN groups.deleted_at IS '软删除时间，NULL=未删除';

COMMENT ON TABLE user_groups IS '成员-分组关联表（多对多），联合主键无独立生命周期';
COMMENT ON COLUMN user_groups.group_id IS '分组 ID，联合主键之一，外键指向 groups(id)';
COMMENT ON COLUMN user_groups.user_id IS '成员 ID，联合主键之一，外键指向 users(id)';

-- ============ RBAC ============
COMMENT ON TABLE resources IS '平台资源目录（RBAC 鉴权资源清单）：平台级，不挂租户';
COMMENT ON COLUMN resources.id IS '自增主键';
COMMENT ON COLUMN resources.name IS '资源名，对应角色规则里的 resource（users/groups/auth 等）';
COMMENT ON COLUMN resources.scope IS '作用域：cluster=平台级';
COMMENT ON COLUMN resources.kind IS '资源种类：resource=API 资源 / menu=菜单';

COMMENT ON TABLE roles IS '角色（租户内）：rules 声明资源-操作授权规则，可挂成员或分组';
COMMENT ON COLUMN roles.id IS '自增主键';
COMMENT ON COLUMN roles.name IS '角色名，租户内唯一（软删友好部分唯一索引 uk_roles_tenant_name，FIX-002）';
COMMENT ON COLUMN roles.scope IS '作用域：cluster=平台级';
COMMENT ON COLUMN roles.namespace IS '命名空间，预留扩展';
COMMENT ON COLUMN roles.rules IS '授权规则 JSON 数组，元素形如 {"resource":"*","operation":"*"}；operation 支持 */edit/view';
COMMENT ON COLUMN roles.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN roles.created_at IS '创建时间';
COMMENT ON COLUMN roles.updated_at IS '更新时间';
COMMENT ON COLUMN roles.deleted_at IS '软删除时间，NULL=未删除';

COMMENT ON TABLE user_roles IS '成员-角色关联表（多对多），联合主键无独立生命周期';
COMMENT ON COLUMN user_roles.user_id IS '成员 ID，联合主键之一，外键指向 users(id)';
COMMENT ON COLUMN user_roles.role_id IS '角色 ID，联合主键之一，外键指向 roles(id)';

COMMENT ON TABLE group_roles IS '分组-角色关联表（多对多）：分组内成员经本表继承角色授权';
COMMENT ON COLUMN group_roles.group_id IS '分组 ID，联合主键之一，外键指向 groups(id)';
COMMENT ON COLUMN group_roles.role_id IS '角色 ID，联合主键之一，外键指向 roles(id)';

-- ============ 审计 ============
COMMENT ON TABLE audit_logs IS '业务审计日志（FIX-013）：追加写流水，记录谁在什么租户对什么资源做了什么；无更新/软删语义';
COMMENT ON COLUMN audit_logs.id IS '自增主键';
COMMENT ON COLUMN audit_logs.tenant_id IS '租户 ID，0=平台级操作（运营域）';
COMMENT ON COLUMN audit_logs.account_id IS '操作者平台账号 ID，0=系统或未知';
COMMENT ON COLUMN audit_logs.member_id IS '操作者租户成员 ID，0=平台级操作或未知';
COMMENT ON COLUMN audit_logs.module IS '业务域模块：tenant/iam/...';
COMMENT ON COLUMN audit_logs.action IS '动作：create/update/delete/bind/...';
COMMENT ON COLUMN audit_logs.resource_type IS '目标资源类型（users/groups/roles 等）';
COMMENT ON COLUMN audit_logs.resource_id IS '目标资源 ID，可空';
COMMENT ON COLUMN audit_logs.request_id IS '链路追踪请求 ID，可空';
COMMENT ON COLUMN audit_logs.ip IS '客户端 IP，可空';
COMMENT ON COLUMN audit_logs.user_agent IS '客户端 User-Agent，可空';
COMMENT ON COLUMN audit_logs.before_data IS '变更前数据快照 JSONB，可空';
COMMENT ON COLUMN audit_logs.after_data IS '变更后数据快照 JSONB，可空';
COMMENT ON COLUMN audit_logs.created_at IS '记录发生时间，默认当前时间';
