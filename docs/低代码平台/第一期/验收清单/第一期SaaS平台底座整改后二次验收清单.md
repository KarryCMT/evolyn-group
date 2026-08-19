# Evolyn Group 第一期 SaaS 平台底座整改后二次验收清单

> 项目：`KarryCMT/evolyn-group`  
> 分支：`1.0.0`  
> 二次验收基线：`86666a8`（`Add: 整改`，2026-08-19）  
> 对照文档：`docs/低代码平台/第一期/第一期SaaS平台底座整改文档.md`  
> 审计范围：后端代码 + 数据库模型 + Migration + 多租户安全 + SaaS 生命周期  
> 不包含：任何前端实现；应用、表单、表单设计、表单数据、Schema Engine、动态 DDL、Data Engine、Workflow、Dashboard 等后续低代码业务能力。

---

## 1. 二次验收结论

本轮整改已解决第一期 SaaS Foundation 中绝大多数数据库模型、多租户边界和生命周期基础问题。

当前不建议继续对已完成的 FIX-001～FIX-017 进行重复整改，后续开发应集中在以下四个真正影响“第一期正式验收”的收口项：

1. Tenant Open / Provisioning 全流程事务化。
2. Member 加入 Tenant 全流程事务化。
3. 补齐跨租户攻击与隔离集成测试。
4. 修正 Migration Parser 单元测试并跑通后端完整 CI/Test。

上述四项完成后，可将第一期 SaaS 平台底座状态由：

> 基本完成，待工程收口

调整为：

> 第一期 SaaS Foundation 整改验收完成，可进入下一阶段。

---

## 2. 原整改项二次验收状态

| ID | 二次验收状态 | 处理结论 |
|---|---|---|
| FIX-001 | ✅ 已验收 | Role Model 与数据库生命周期字段已统一 |
| FIX-002 | ✅ 已验收 | Role 已实现租户内部分唯一 |
| FIX-003 | ✅ 已验收 | Group 已实现租户内部分唯一 |
| FIX-004 | ✅ 已验收 | Member `(tenant_id, account_id)` 唯一约束已落地 |
| FIX-005 | ✅ 已验收 | `users.account_id -> accounts.id` FK 已落地 |
| FIX-006 | 🟡 待最终验收 | Service 层同租户校验已实现，但缺攻击/集成测试 |
| FIX-007 | ✅ 已验收 | frozen/deleted Tenant 请求级拦截已实现 |
| FIX-008 | ✅ 一期验收 | Platform API 与 Tenant Middleware 已完成路由域隔离 |
| FIX-009 | 🟡 待最终验收 | Migration 主体完成，但 Migration Parser 测试需修正并实际跑通 |
| FIX-010 | 🟡 待最终验收 | AddMember 流程已具备，但缺少事务闭环 |
| FIX-011 | ✅ 已验收 | Member Quota 一期执行闭环已完成 |
| FIX-012 | ✅ 已验收 | Tenant retention / purge 一期闭环已实现 |
| FIX-013 | ✅ 已验收 | 基础 Audit Log 持久化与业务接入已实现 |
| FIX-014 | ✅ 已验收 | PlatformBaseModel / TenantBaseModel 已分层 |
| FIX-015 | ✅ 已验收 | Department parent_id 完整性与循环检测已实现 |
| FIX-016 | ✅ 已验收 | Tenant Owner Account FK 已建立 |
| FIX-017 | ✅ 已验收 | AuthInfo 外部身份唯一约束已建立 |
| FIX-018 | ⏸ 后置专项 | 关系表 tenant_id 是否冗余存储，不阻塞第一期 |
| FIX-019 | ⏸ 后置专项 | Casbin 与自研 RBAC 选型，不阻塞第一期 |

---

# 3. 第一优先级：FIX-020 Tenant Provisioning 事务闭环

## 3.1 问题

当前 Tenant Open / Provisioning 已经包含：

```text
Account
  ↓
Tenant
  ↓
Quota
  ↓
Owner Member
  ↓
Baseline Roles
  ↓
Baseline Groups
  ↓
GroupRole
  ↓
OwnerRole
  ↓
Audit
```

业务流程已经基本完整，但这些数据库写操作尚未形成一个完整原子事务。

如果中间任意步骤失败，可能形成：

```text
Tenant 已创建
Owner Member 已创建
Role 创建了一部分
Group / RoleBinding 创建失败
API 返回失败
```

此时调用者看到“开通失败”，数据库却存在一个半初始化 Tenant。

这类状态会直接污染后续：Tenant 列表、登录租户列表、配额统计、RBAC 初始化、Tenant 重试开通、数据清理。

## 3.2 整改目标

Tenant Provisioning 需要满足：

> 要么完整开通成功，要么本次 Provisioning 所产生的租户域数据全部回滚。

推荐原子边界：

```text
BEGIN
Create Tenant
Create Owner Member
Create Baseline Roles
Create Baseline Groups
Create GroupRole
Create OwnerRole
COMMIT
```

Audit Log 推荐与业务事务一起写入；如果独立写入，Audit 失败不得反向导致租户开通被判失败，但必须有日志/监控捕获。

## 3.3 Account 创建边界

Account 已存在时直接复用。Account 本次新建时，优先与 Tenant Provisioning 同事务创建；若架构暂时无法跨域共享事务，则必须明确失败补偿规则，禁止自然留下无法解释的孤立 Account。

## 3.4 建议代码改造

重点检查：

```text
apps/evolyn-core/internal/platform/tenant/service/tenant.go
```

建议引入统一事务边界，让同一 `txCtx / tx` 向下贯穿 Tenant、IAM、Audit Repository，而不是让各 Repository 各自提交。

## 3.5 必须补充的测试

- `TX-TENANT-001`：正常开通，Tenant/Owner/Role/Group/关系全部完整。
- `TX-TENANT-002`：Role 创建失败，所有本次数据回滚。
- `TX-TENANT-003`：Group 创建失败，所有本次数据回滚。
- `TX-TENANT-004`：MemberRole 绑定失败，所有本次数据回滚。
- `TX-TENANT-005`：重复请求/重试，不得产生重复 Tenant、Owner、baseline Role/Group。

## 3.6 完成标准

- [ ] Tenant Open 使用明确 Transaction。
- [ ] 所有 Provisioning Repository 使用同一 Transaction。
- [ ] 任意中间失败均可完整回滚。
- [ ] 新 Account 的失败补偿策略明确。
- [ ] 正常开通测试通过。
- [ ] 至少 3 个不同失败点的回滚测试通过。
- [ ] 重试/重复请求测试通过。

---

# 4. 第二优先级：FIX-021 AddMember 原子事务闭环

## 4.1 问题

当前成员加入租户流程已经实现：

```text
Account → 重复检查 → Quota Check → Create Member → Bind Departments → Bind Roles → Audit
```

但 Member 创建、Department 绑定和 Role 绑定不是统一事务，可能形成“半完成 Member”。

## 4.2 整改目标

以下操作必须形成一个原子业务事务：

```text
BEGIN
Check Member uniqueness
Check quota
Create Member
Bind Departments
Bind Roles
COMMIT
```

Quota Check 还需考虑并发，否则可能出现多个请求同时通过配额检查后超配额创建。

## 4.3 建议代码改造

重点：

```text
apps/evolyn-core/internal/platform/iam/service/user.go
```

建议由 `UserService.AddMember` 控制统一事务，Member/Department/Role Repository 共享当前事务。

## 4.4 必须补充的测试

- `TX-MEMBER-001`：正常新增 Member，关系完整。
- `TX-MEMBER-002`：Department 绑定失败，Member 与关系全部回滚。
- `TX-MEMBER-003`：Role 绑定中途失败，Member/Department/前序 Role 绑定全部回滚。
- `TX-MEMBER-004`：Quota 超限，不产生任何业务数据。
- `TX-MEMBER-005`：同 Account 并发加入同 Tenant，最终只有一个有效 Member。

## 4.5 完成标准

- [ ] AddMember 有统一 Transaction。
- [ ] Member/Department/Role 绑定共享 Transaction。
- [ ] 任一关系绑定失败时 Member 自动回滚。
- [ ] Quota 失败不产生任何业务数据。
- [ ] 并发重复 Member 不会突破唯一约束。
- [ ] 回滚测试通过。

---

# 5. 第三优先级：FIX-022 多租户安全攻击测试矩阵

## 5.1 目标

FIX-006 的 Service 同租户校验已经具备，但必须用自动化测试把租户隔离从“代码约定”变成“回归保护”。

## 5.2 基础测试数据

准备 Tenant A/B，各自包含 Account、Member、Role、Group、Department。

## 5.3 写攻击测试

- `SEC-TENANT-001`：Tenant A Member 绑定 Tenant B Role，必须拒绝。
- `SEC-TENANT-002`：Tenant A Member 加入 Tenant B Group，必须拒绝。
- `SEC-TENANT-003`：Tenant A Member 设置 Tenant B Department，必须拒绝。
- `SEC-TENANT-004`：Tenant A Group 绑定 Tenant B Role，必须拒绝。
- `SEC-TENANT-005`：Tenant A Department 使用 Tenant B Department 作为 parent，必须拒绝。
- `SEC-TENANT-006`：请求上下文是 Tenant A，但伪造 Tenant B Entity ID，必须不可访问。

## 5.4 读攻击测试

- `SEC-TENANT-007`：Tenant A 查询 Tenant B Member，拒绝或不可见。
- `SEC-TENANT-008`：Tenant A 查询 Tenant B Role，拒绝或不可见。
- `SEC-TENANT-009`：Tenant A 查询 Tenant B Group，拒绝或不可见。
- `SEC-TENANT-010`：Tenant A 查询 Tenant B Department，拒绝或不可见。

## 5.5 更新/删除攻击测试

至少覆盖 Member、Role、Group、Department 的 Update/Delete。原则是所有 ID 查询首先进入 Tenant Scope，再执行后续修改。

## 5.6 Tenant 状态测试

- `SEC-TENANT-STATE-001`：active Tenant 正常访问。
- `SEC-TENANT-STATE-002`：frozen Tenant 被 TenantStatusMiddleware 拦截。
- `SEC-TENANT-STATE-003`：deleted Tenant 被拒绝。
- `SEC-TENANT-STATE-004`：Platform API 不错误依赖 TenantContext。

## 5.7 完成标准

- [ ] 至少 10 个跨租户攻击测试。
- [ ] Role / Group / Department / Member 全覆盖。
- [ ] 写、读、Update/Delete 攻击测试覆盖。
- [ ] frozen/deleted Tenant 回归测试覆盖。
- [ ] CI 中自动执行。

---

# 6. 第四优先级：FIX-023 Migration 测试与 CI 收口

## 6.1 当前问题

Migration 框架已经具备 embedded migrations、schema_migrations、version、checksum、up/down、transaction，但 `SplitSQLStatements` 的实现行为与 `TestSplitSQLStatements` 断言存在不一致风险。

重点文件：

```text
apps/evolyn-core/internal/infrastructure/migrate.go
apps/evolyn-core/internal/infrastructure/migrate_test.go
```

推荐明确 Parser 契约：SQL comment 不属于可执行 statement，解析时允许丢弃。

## 6.2 必须增加的 Parser 测试

- `MIGRATE-PARSER-001`：普通多条 SQL 正确切分。
- `MIGRATE-PARSER-002`：单行注释内分号不切分。
- `MIGRATE-PARSER-003`：字符串内部的分号不切分。
- `MIGRATE-PARSER-004`：PostgreSQL `$$...$$` Function 正确识别为单条。
- `MIGRATE-PARSER-005`：自定义 `$func$...$func$` 正确识别。
- `MIGRATE-PARSER-006`：Block Comment 若声明支持，则必须正确处理；否则明确限制。

## 6.3 Migration 集成测试

- `MIGRATE-INT-001`：空数据库执行全部 Migration 成功。
- `MIGRATE-INT-002`：第二次执行不重复迁移。
- `MIGRATE-INT-003`：checksum 被修改时明确拒绝或报警。
- `MIGRATE-INT-004`：中间 Migration 故意失败时事务完整回滚。
- `MIGRATE-INT-005`：最终 Schema 与 `db.sql` 关键约束一致。

至少核对：users account FK、tenant/account unique、role/group tenant unique、department parent FK、auth_info unique、tenant lifecycle 字段、audit_logs。

## 6.4 CI 最终执行项

```bash
go test ./...
go vet ./...
gofmt -l .
```

建议额外执行：

```bash
go test -race ./...
```

如果全仓成本较高，至少覆盖 tenant、iam、middleware、infrastructure/migrate。

## 6.5 完成标准

- [ ] `TestSplitSQLStatements` 与 Parser 契约一致。
- [ ] Parser 边界测试补齐。
- [ ] Migration 真实 PostgreSQL 集成测试通过。
- [ ] Migration 失败可完整回滚。
- [ ] checksum 防篡改测试通过。
- [ ] `go test ./...` 通过。
- [ ] `go vet ./...` 通过。
- [ ] gofmt 检查通过。
- [ ] CI 全绿。

---

# 7. 不阻塞第一期的后置事项

## 7.1 FIX-018：关系表 tenant_id

当前关系表是否冗余保存 `tenant_id` 应作为后续数据库隔离强化专题。第一期已有 TenantContext + Tenant Scope Repository + Service SameTenant Check + FK 的基础隔离，因此不应阻塞当前验收。进入应用/表单/Data Engine 前，再评审 composite FK、RLS、partition 等数据库层隔离方案。

## 7.2 FIX-019：Casbin

当前项目已经存在自研 Authorizer / Role / Resource / Operation / Rules / UserRole / GroupRole，因此“没有 Casbin”不等于“没有权限系统”。后续基于 RBAC 复杂度、数据权限、ABAC、策略更新、规模和性能再做专项选型。

## 7.3 Platform Admin 身份域

Platform API 已与 Tenant Middleware 解耦，但 Platform Admin 身份后续建议继续演进为 PlatformIdentity / PlatformRole 与 TenantMembership / TenantRole 两套权限域。该项不阻塞第一期。

---

# 8. 建议开发顺序

```text
FIX-020 Tenant Provisioning Transaction
        ↓
FIX-021 AddMember Transaction
        ↓
FIX-022 Cross-Tenant Security Tests
        ↓
FIX-023 Migration + CI Finalization
        ↓
第一期最终验收
```

事务整改可能改变 Repository 调用方式，因此建议先改事务，再补完整测试矩阵。

---

# 9. 事务基础设施建议

如果当前没有统一事务管理器，建议建立轻量 TxManager，而不是引入复杂 Unit of Work：

```go
type TxManager interface {
    WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
```

事务开始后将 tx 注入 context。Repository 获取 DB 时遵循：有 tx 使用 tx，无 tx 使用默认 pool/db。这样 TenantService.Open、UserService.AddMember、TenantPurge 可以统一控制事务，而不需要大量 `CreateWithTx/UpdateWithTx/DeleteWithTx` API。

---

# 10. 最终验收 Gate

## Gate A：Schema

- [ ] Migration 从空库可以完整初始化。
- [ ] Schema 与 db.sql 终态快照一致。
- [ ] 核心 FK / Unique / Partial Index 生效。

## Gate B：Tenant Boundary

- [ ] Tenant A 无法读取 Tenant B 数据。
- [ ] Tenant A 无法修改 Tenant B 数据。
- [ ] Tenant A 无法创建跨租户关系。
- [ ] frozen Tenant 无法访问租户业务 API。
- [ ] deleted Tenant 无法访问租户业务 API。

## Gate C：Lifecycle

- [ ] Tenant Open 原子化。
- [ ] AddMember 原子化。
- [ ] Freeze 生效。
- [ ] Delete/Retention/Purge 生效。
- [ ] Purge 不产生半清理状态。

## Gate D：Quota

- [ ] members quota 正常限制。
- [ ] quota 超限不会留下 Member 数据。
- [ ] 并发场景不会明显突破限制。

## Gate E：Audit

- [ ] Tenant Open 有审计。
- [ ] Tenant Freeze/Delete 有审计。
- [ ] Member Add/Remove 有审计。
- [ ] Role/权限关键变化有审计。
- [ ] 审计记录包含 Actor / Tenant / Action / RequestMeta。

## Gate F：Engineering

- [ ] Migration Parser Test 通过。
- [ ] Migration Integration Test 通过。
- [ ] `go test ./...` 通过。
- [ ] `go vet ./...` 通过。
- [ ] gofmt 检查通过。
- [ ] CI 全绿。

---

# 11. 第一期结束判定

当 FIX-020～FIX-023 与上述 Gate 全部完成后，可认为 Evolyn Group 已具备进入下一阶段低代码业务内核开发的 SaaS 平台基础。

此时不建议继续在第一期无限追加“可能以后会用到”的能力。Application、Form、Schema Engine、DDL Engine、Data Engine、Permission Engine、Relation Engine、Formula Engine、Workflow、Dashboard 等，应进入新的里程碑管理，不再作为第一期缺陷。

---

# 12. 当前剩余任务最终清单

## P0：必须完成

- [ ] FIX-020 Tenant Provisioning 全事务。
- [ ] FIX-021 AddMember 全事务。
- [ ] FIX-022 Cross-Tenant Attack Tests。
- [ ] FIX-023 Migration Test / Integration Test / CI 全绿。

## 后置，不阻塞一期

- [ ] FIX-018 关系表 tenant_id / composite FK / RLS 专项评审。
- [ ] FIX-019 自研 RBAC / Casbin 专项评审。
- [ ] PlatformIdentity 与 TenantMembership 权限域进一步拆分。
- [ ] Quota 并发强一致策略增强。
- [ ] Audit 企业级检索、归档、长期留存策略。

---

## 13. 推荐验收状态

当前：

> **第一期 SaaS 平台底座整改完成度：主体完成，工程收口中。**

完成 FIX-020 ～ FIX-023 后：

> **第一期 SaaS 平台底座：验收完成。**
