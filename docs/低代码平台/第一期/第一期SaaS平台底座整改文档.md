# Evolyn Group 第一期 SaaS 平台底座整改文档

> 项目：`KarryCMT/evolyn-group`\
> 审计范围：后端代码级审计 + 数据库模型代码级审计\
> 目标版本：第一期 SaaS Foundation 收尾\
> 范围说明：本期**不包含**应用、表单、表单设计、表单数据、Schema
> Engine、动态 DDL、Data Engine、Workflow、Dashboard
> 等后续低代码业务能力；**不考虑任何前端实现**。

## 1. 整改目标

本轮整改的目标不是扩展低代码业务功能，而是将现有 SaaS
平台底座从"基础能力已成型"提升到"多租户边界清晰、数据库约束可靠、生命周期闭环、可安全进入下一阶段开发"的状态。

当前后端已经具备 PostgreSQL、Redis、JWT
Authentication、TenantContext、Authorization、RateLimit、Log、Trace、Monitor、Account、Tenant、Member、Department、Role/RBAC
等基础能力。本次整改重点处理实际代码与数据库模型审计中发现的多租户一致性、Schema
漂移、租户状态控制、关系约束及平台域隔离问题。

## 2. 已确认无需重复建设的能力

以下能力已在当前后端实现或已接入请求链路，本期不再作为"未完成任务"重复建设：

-   PostgreSQL 初始化与访问基础能力。
-   Redis 初始化与基础能力。
-   JWT Authentication。
-   TenantMiddleware / TenantContext。
-   自研 Authorization / RBAC 基础能力。
-   RateLimitMiddleware。
-   LogMiddleware、TraceMiddleware、MonitorMiddleware。
-   Account 与 Tenant Member 身份拆分。
-   Tenant 基础 CRUD 与状态字段。
-   Department 基础模型与 CRUD。
-   Member 查询、修改、删除、角色和部门关联基础能力。
-   Role / Resource / Operation / UserRole / GroupRole 等 RBAC
    基础模型。

> 注意：Casbin 当前未实现，但项目并非"没有权限系统"。现状是自研
> Authorizer + Role/Resource/Rules 模型，因此是否引入 Casbin
> 应作为专项架构评审，而不是直接认定为一期缺陷。

## 3. 整改任务总览

  ----------------------------------------------------------------------------------------------------------
  ID             优先级         整改项                         类型           目标
  -------------- -------------- ------------------------------ -------------- ------------------------------
  FIX-001        P0             Role Model 与数据库模型统一    模型一致性     修复软删除/时间字段映射差异

  FIX-002        P0             roles 名称改为租户内唯一       数据约束       防止多租户同名角色冲突

  FIX-003        P0             groups 名称改为租户内唯一      数据约束       防止多租户同名用户组冲突

  FIX-004        P0             users 增加 tenant+account      数据约束       保证账号在同一租户只有一个
                                唯一约束                                      Member

  FIX-005        P0             users.account_id 增加外键      数据完整性     防止孤立 Member

  FIX-006        P0             多租户关联写入增加同租户校验   安全           防止跨租户
                                                                              Role/Group/Department 绑定

  FIX-007        P0             Tenant frozen/deleted          生命周期       冻结/注销后真正阻止租户访问
                                请求级拦截                                    

  FIX-008        P0             Platform API 与 Tenant         架构边界       平台域不依赖租户上下文
                                Middleware 隔离                               

  FIX-009        P0             明确数据库 Migration           工程治理       消除 db.sql 与 AutoMigrate
                                唯一事实来源                                  漂移

  FIX-010        P1             Member 新增/加入 Tenant 闭环   SaaS 能力      完整成员生命周期

  FIX-011        P1             Member Quota 执行闭环          SaaS 商业化    验证套餐/配额执行架构

  FIX-012        P1             Tenant 注销 retention/purge    生命周期       建立租户数据销毁闭环

  FIX-013        P1             Audit Log 持久化               审计           记录关键业务操作

  FIX-014        P1             Tenant/Account BaseModel 分层  模型治理       移除 tenants 自身 tenant_id
                                                                              语义问题

  FIX-015        P1             Department parent_id 完整性    数据完整性     防止无效/跨租户父部门

  FIX-016        P1             Tenant owner_account_id 外键   数据完整性     Owner 与 Account 建立真实引用

  FIX-017        P1             AuthInfo 第三方身份唯一约束    身份安全       防止同一外部身份绑定多个账号

  FIX-018        P1/P2          关系表 tenant_id 方案评审      数据隔离       提升数据库层租户隔离能力

  FIX-019        专项           自研 RBAC 与 Casbin 选型评审   架构           决定是否继续自研权限模型
  ----------------------------------------------------------------------------------------------------------

## 4. P0 必须整改

### 4.1 FIX-001：统一 Role GORM Model 与 roles 表

**现状**：数据库 `roles` 存在
`tenant_id / created_at / updated_at / deleted_at`，但当前 Role Model
未完整继承统一 BaseModel。Repository 使用 `Delete`
时可能产生物理删除，而数据库设计意图表现为软删除。

**整改**：统一 Role 生命周期模型，使
`TenantID / CreatedAt / UpdatedAt / DeletedAt`
与数据库完全一致；删除行为统一为软删除，确需物理删除时显式 `Unscoped`。

**验收**：创建、更新、删除 Role 后字段行为与 SQL Schema
一致；普通查询自动过滤已删除角色；删除不会直接丢失记录。

### 4.2 FIX-002 / FIX-003：Role、Group 改为租户内唯一

**现状**：`roles.name`、`groups.name` 在 `db.sql` 中为全局
`UNIQUE`，不符合多租户 SaaS 模型。

**整改建议**：采用软删除友好的部分唯一索引：

``` sql
CREATE UNIQUE INDEX uk_roles_tenant_name
ON roles (tenant_id, name)
WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX uk_groups_tenant_name
ON groups (tenant_id, name)
WHERE deleted_at IS NULL;
```

**验收**：Tenant A 和 Tenant B 可以分别创建 `admin`；同一 Tenant
内不能创建两个未删除的同名角色/用户组；软删除后可重新创建同名记录。

### 4.3 FIX-004 / FIX-005：Member 身份唯一性与 Account FK

**现状**：`users` 表承担 Tenant Member 身份，但数据库没有强制
`(tenant_id, account_id)` 唯一，同时 `account_id` 缺少对 `accounts(id)`
的正式 FK。

**整改建议**：

``` sql
ALTER TABLE users
ADD CONSTRAINT fk_users_account
FOREIGN KEY (account_id) REFERENCES accounts(id);

CREATE UNIQUE INDEX uk_users_tenant_account
ON users (tenant_id, account_id)
WHERE deleted_at IS NULL;
```

历史数据需先清理 `account_id = 0`、不存在 Account、同租户重复 Member
等异常记录。

**验收**：一个 Account 可以属于多个 Tenant，但同一 Tenant
只能拥有一个有效 Member；无法写入不存在的 Account。

### 4.4 FIX-006：阻止跨租户关联污染

重点关系包括：`department_users`、`user_groups`、`user_roles`、`group_roles`。

**风险**：单字段 FK 只能保证实体存在，无法保证两端属于同一个
Tenant。例如 Tenant A 的 User 在数据库层可能绑定 Tenant B 的 Role。

**一期最低要求**：所有关联写操作在 Service 层加载两端实体并验证 TenantID
一致，不允许 Repository 直接盲写关系。

``` text
User.TenantID == Role.TenantID
User.TenantID == Group.TenantID
User.TenantID == Department.TenantID
Group.TenantID == Role.TenantID
```

同时增加跨租户攻击测试：Tenant A 用户尝试绑定 Tenant B
Role/Group/Department 时必须失败。

**后续增强**：评估给关系表增加 `tenant_id`，使用 `(tenant_id, ...)`
组合主键/唯一键，为未来 RLS、审计和租户清理提供基础。

### 4.5 FIX-007：Tenant frozen/deleted 状态真正进入请求控制

**现状**：Tenant 已支持 `active / frozen / deleted` 状态修改，但
TenantMiddleware 重点是解析和注入
TenantContext，冻结状态还需要形成请求级闭环。

**整改后的请求链**：

``` text
Authentication
    ↓
Resolve Tenant
    ↓
Tenant Status Check
    ├── active  → continue
    ├── frozen  → reject
    └── deleted → reject
    ↓
Authorization
```

建议定义稳定错误码，例如 `TENANT_FROZEN`、`TENANT_DISABLED`，并考虑
Redis 缓存 Tenant 状态及状态变更后的缓存失效。

**验收**：已有 JWT 在租户被冻结后也不能继续访问租户业务
API；恢复后重新允许访问。

### 4.6 FIX-008：Platform API 与 Tenant API 中间件隔离

**现状**：TenantMiddleware 当前处于 Engine 全局链路，因此
`/api/v1/platform/*` 也会经过
TenantMiddleware，平台域与租户域边界不够清晰。

**目标结构**：

``` text
/api/v1/platform/*
Authentication
PlatformAuthorization
NO TenantContext

/api/v1/*
Authentication
TenantMiddleware
TenantStatusCheck
TenantAuthorization
```

建议使用 Gin Route Group 分层注册，而不是把 TenantMiddleware 作为所有
API 的全局强依赖。

**验收**：Platform Admin 查询任意 Tenant 不依赖当前 TenantID；普通
Tenant API 必须存在有效 TenantContext；两种权限域不可串用。

### 4.7 FIX-009：Migration 成为数据库 Schema 唯一事实来源

**现状**：项目同时维护 `scripts/db.sql` 和 GORM
`AutoMigrate`，审计已发现 Role、Group、Resource 等模型存在 SQL/GORM
漂移。

**整改方向**：生产环境使用版本化 SQL Migration 作为唯一 Schema Source of
Truth；AutoMigrate 只允许在测试或开发实验场景使用，不参与正式生产升级。

建议目录：

``` text
apps/evolyn-core/migrations/
├── 000001_init.up.sql
├── 000001_init.down.sql
├── 000002_fix_tenant_unique.up.sql
├── 000002_fix_tenant_unique.down.sql
├── 000003_member_constraints.up.sql
└── ...
```

每次 Model 修改必须同时提交 Migration；CI 可创建空库执行全部
Migration，再运行 Repository/Integration Test。

## 5. P1 SaaS 底座收尾

### 5.1 FIX-010：Member 新增/加入 Tenant 闭环

当前 Member
查询、修改、删除、角色/部门关联已有基础，但需要补齐通用的"Account 成为
Tenant
Member"入口。一期不要求复杂邮件/短信邀请系统，可先提供内部创建/加入
API。

建议事务流程：校验 Account → 校验 Tenant → Quota Check → 创建 Member →
绑定 Department/Role → Audit Log。

### 5.2 FIX-011：先用 Member Quota 打通配额引擎

Tenant 已有
`plan`、`quotas JSONB`，但配额执行能力尚未形成闭环。由于应用/表单不属于一期，本期只需用
`members` 验证架构。

建议形成统一接口：

``` text
QuotaService.Check(tenantID, "members")
```

超过限制返回稳定业务错误码
`QUOTA_EXCEEDED`。后续应用、表单、存储、流程次数都复用同一服务。

### 5.3 FIX-012：Tenant 注销 retention / purge

当前
deleted/软删除不等于完整的数据生命周期。建议补充注销请求时间、保留截止时间、最终清理时间，并提供定时
Purge Worker。

建议字段：`delete_requested_at`、`retention_until`、`purged_at`。

一期可以先实现"注销 → 保留期 → Worker
标记/清理"的框架，具体业务数据清理策略随未来模块扩展。

### 5.4 FIX-013：业务 Audit Log 持久化

技术 Log/Trace/Monitor
已存在，不重复建设；需要补的是"谁在什么租户对什么资源做了什么修改"的业务审计。

建议最小字段：

``` text
audit_logs
id
tenant_id
account_id
member_id
module
action
resource_type
resource_id
request_id
ip
user_agent
before_data JSONB
after_data JSONB
created_at
```

优先记录：租户状态、成员增删、部门调整、角色调整、权限调整、套餐/配额修改。

## 6. P1 数据模型治理

### 6.1 FIX-014：拆分 PlatformBaseModel 与 TenantBaseModel

当前统一 BaseModel 包含 TenantID，导致 Tenant 本身也存在
`tenant_id`，语义上形成"租户属于哪个租户"的问题。

建议：

``` text
PlatformBaseModel
  CreatedAt / UpdatedAt / DeletedAt

TenantBaseModel
  TenantID / CreatedAt / UpdatedAt / DeletedAt
```

`Tenant / Account` 使用
PlatformBaseModel；`User / Department / Group / Role` 使用
TenantBaseModel。

此项涉及面较大，可在 P0 约束修复后单独迁移，不建议与第一批 SQL
修复混成一个大提交。

### 6.2 FIX-015：Department parent_id 完整性

建议从 `0 = root` 逐步迁移为 `NULL = root`，并增加自引用 FK。Service
层同时验证 parent 与 child 属于同一
Tenant，并禁止自己成为自己的父节点及循环层级。

### 6.3 FIX-016：Tenant Owner Account FK

`owner_account_id` 建议由 `0 = 未设置` 改为可空
FK：`NULL = 暂无 Owner`，非空时必须引用真实 Account。

### 6.4 FIX-017：AuthInfo 外部身份唯一性

建议建立软删除友好的唯一索引：

``` sql
CREATE UNIQUE INDEX uk_auth_identity
ON auth_infos (auth_type, auth_id)
WHERE deleted_at IS NULL;
```

如果后续同一 `auth_type` 存在多个 issuer/provider，再扩展为
`(provider, issuer, auth_id)`。

### 6.5 FIX-018：关系表 tenant_id 专项评审

一期可以先通过 Service
强校验解决跨租户写入，但长期建议评估关系表显式携带
`tenant_id`。收益包括数据库层隔离、未来 PostgreSQL
RLS、租户数据清理、审计查询和组合 FK 能力。

## 7. RBAC / Casbin 专项决策

当前项目已有自研 Role、Resource、Rules、UserRole、GroupRole、Authorizer
与 AuthorizationMiddleware，因此不应简单将"Casbin
未接入"认定为权限未完成。

建议在第一期 P0/P1 修复后，专项验证现有 RBAC
是否满足：租户隔离、资源+操作授权、成员角色、用户组角色、平台域与租户域隔离、缓存失效、默认拒绝、跨租户防护、后续数据权限扩展。

若满足，第一期继续自研体系；若规则模型开始复杂化、策略管理成本过高，再引入
Casbin Domain。不要为了技术选型而重写已经工作的权限体系。

## 8. 推荐实施顺序

### 阶段 A：数据库约束止血

1.  修复 Role Model。
2.  修复 Role/Group 租户内唯一。
3.  修复 User `(tenant_id, account_id)` 唯一。
4.  清理历史异常数据并增加 Account FK。
5.  增加跨租户关系测试和 Service 校验。

### 阶段 B：请求边界闭环

1.  Tenant frozen/deleted 状态拦截。
2.  Platform/Tenant Route Group 拆分。
3.  验证 PlatformAuthorization 与 TenantAuthorization 边界。

### 阶段 C：工程治理

1.  引入版本化 Migration。
2.  停止生产 AutoMigrate。
3.  CI 增加 Migration + Integration Test。

### 阶段 D：SaaS 功能收尾

1.  Member Add/Join Tenant。
2.  Member Quota。
3.  Audit Log。
4.  Tenant Retention/Purge。

### 阶段 E：模型进一步清理

1.  PlatformBaseModel / TenantBaseModel。
2.  Department parent FK。
3.  Owner Account FK。
4.  AuthInfo Unique。
5.  关系表 tenant_id 方案。
6.  RBAC/Casbin 专项评审。

## 9. 必须补充的测试

-   两个 Tenant 可创建同名 Role/Group。
-   同 Tenant 不可创建重复有效 Role/Group。
-   软删除后可重新创建同名 Role/Group。
-   同 Account 在同 Tenant 不能产生两个有效 Member。
-   同 Account 可以加入不同 Tenant。
-   Tenant A Member 不可绑定 Tenant B Role。
-   Tenant A Member 不可加入 Tenant B Group。
-   Tenant A Member 不可加入 Tenant B Department。
-   Tenant A Group 不可绑定 Tenant B Role。
-   Frozen Tenant 的旧 JWT 不能继续访问。
-   Deleted Tenant 不能继续访问。
-   Platform API 不依赖 TenantContext。
-   Tenant API 缺 TenantContext 时拒绝访问。
-   Role Delete 确认为软删除。
-   Migration 可从空数据库完整执行。
-   Migration 后 Model 与实际 Schema 一致。

## 10. 第一期完成判定

当以下条件全部满足后，可认为 Evolyn 第一期 SaaS Foundation
的后端底座完成：

-   多租户实体唯一约束正确。
-   Member 身份唯一且 Account 引用完整。
-   所有 IAM 关系不存在跨租户写入路径。
-   Tenant frozen/deleted 真正控制请求访问。
-   Platform 与 Tenant API 权限域彻底分离。
-   数据库 Schema 有唯一、可版本化的迁移来源。
-   Member 加入 Tenant 有完整后端闭环。
-   至少 `members` 配额已经真实执行。
-   关键管理操作进入业务审计日志。
-   Tenant 注销存在明确的数据保留/清理策略。
-   自研 RBAC 是否继续使用已有明确结论。

## 11. 本期明确不纳入整改范围

以下内容不应阻塞第一期 SaaS Foundation 完成：

-   Application / 应用管理。
-   Form / 表单模型。
-   表单设计器。
-   表单业务数据。
-   Schema Engine / Schema Version。
-   Physical Table DDL Engine。
-   Data Engine / 动态 CRUD。
-   Query DSL / Query Engine。
-   Dashboard。
-   Workflow / Flowable。
-   Automation。
-   AI / Eino。
-   PostgreSQL RLS（可在后续动态业务数据阶段结合关系表 tenant_id
    一起实施）。
-   独立数据库租户。
-   子域名租户识别。
-   任何前端页面或交互整改。

------------------------------------------------------------------------

**整改原则**：第一期优先解决"数据不能错、租户不能串、状态必须生效、Schema
必须可控"。在这些底座问题完成前，不建议提前扩展应用/表单等低代码业务模型。

## 12. 评审结论记录（实施时追加）

> 本章为整改实施时形成的评审结论，不回改前文；后续复评在本章续记。

### 12.1 FIX-018：关系表 tenant_id 方案评审结论

**结论：第一期不给 `department_users`、`user_groups`、`user_roles`、
`group_roles` 四张关系表增加 `tenant_id` 列。**

理由：

-   跨租户写入已由 Service 层同租户校验拦截（加载两端实体并比对
    TenantID，错误码 `CROSS_TENANT_BINDING_REJECTED`），并有跨租户
    绑定单测覆盖；
-   关系表行数最大（成员×角色等组合），加列回填与组合键改造的写放大
    在一期收益不明确；
-   租户注销的数据清理已由 Purge Worker 按租户维度清理关系行，
    不依赖关系表自带 tenant_id。

**复评触发条件**（满足任一即重新评审）：需要启用 PostgreSQL RLS；
关系表出现绕过 Service 层的直接写入路径；审计需要按关系行归属租户
直查。落地时同步评估组合外键 `(tenant_id, id)` 的实体表改造。

### 12.2 FIX-019：自研 RBAC 与 Casbin 选型评审结论

**结论：第一期继续使用自研 RBAC（Role/Resource/Rules +
Authorizer + 两条鉴权中间件），不引入 Casbin。**

对照第 7 章验收面逐项核验：

-   租户隔离：组/角色按租户过滤（GORM Callback），成员身份按租户
    签发 JWT；
-   资源+操作授权、成员角色、用户组角色：现有 Rules 模型覆盖；
-   平台域与租户域隔离：FIX-008 落地后由
    PlatformAuthorizationMiddleware 与 TenantMiddleware 两条链路
    分别承担，互不串用；
-   默认拒绝：Authorizer 无规则命中即拒绝，未认证请求按
    unauthenticated 角色收敛；
-   跨租户防护：FIX-006 同租户校验；
-   缓存失效：角色/分组变更路径已清 Redis 成员缓存，暂无独立策略
    缓存，复杂度可控。

**复评触发条件**：出现行级数据权限（按部门/分组过滤数据集）、
权限继承/临时授权/显式拒绝语义、或规则数量增长导致 Authorize
查询成为热点时，再评估引入 Casbin Domain 模型；届时 Authorizer
为唯一替换点，中间件与调用方无感。
