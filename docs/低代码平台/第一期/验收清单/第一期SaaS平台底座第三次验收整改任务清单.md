# 第一期 SaaS 平台底座第三次验收整改任务清单

**验收基线：** `ec01adacfd8d2335fa6af9573b110a1364cfb439`  
**适用范围：** 第一期 SaaS 平台底座后端，不包含应用、表单、表单设计、表单数据及 Data Engine 等低代码业务内核能力。

## 1. 整改结论

基于第三次代码级验收，当前 SaaS 平台底座核心能力已经基本成型，`ec01adac` 对跨租户安全测试、Tenant 状态测试和 Migration Parser 等进行了实质性补强。但二次验收清单中的 P0 收口项尚未全部满足，当前不建议将第一期状态标记为“正式验收完成”。

**本轮必须优先完成：FIX-020、FIX-021；随后完成 FIX-022、FIX-023 的工程验收闭环。**

## 2. 本轮整改项总览

| 整改项 | 问题 | 当前状态 | 优先级 | 完成标准 |
|---|---|---|---|---|
| FIX-020 | Tenant Provisioning 非原子事务 | ❌ 未通过 | P0 | `Open` 全流程同一事务 + 回滚测试 |
| FIX-021 | AddMember 非原子事务 | ❌ 未通过 | P0 | 成员/部门/角色绑定同一事务 + 回滚测试 |
| FIX-022 | 跨租户攻击测试矩阵不完整 | 🟡 部分通过 | P0 | 真实 Repository/PostgreSQL 集成攻击测试闭环 |
| FIX-023 | Migration 工程验收不完整 | 🟡 部分通过 | P0 | `$tag$` + PostgreSQL 集成测试 + CI Gate |

## 3. FIX-020：Tenant Provisioning 全事务整改

`TenantService.Open` 当前包含 Account、Tenant、Owner Member、默认 Role/Group/Binding 等多步写操作，但这些写操作没有形成统一事务边界。后续步骤失败时，前序数据可能已经提交，造成半初始化租户。

### 3.1 整改目标

- Tenant Open 要么全部成功，要么全部回滚，不允许留下半初始化数据。
- Account/Tenant/Owner Member/Role/Group/Binding 等写操作必须共享同一个数据库事务。
- 事务能力应通过统一 `TxManager` / `WithinTransaction` / Repository `WithTx` 等方式传播，避免 Service 内部各 Repository 使用独立连接。
- 审计日志若属于业务原子性的一部分，应明确纳入事务；若采用异步/独立审计，需要明确失败策略。

### 3.2 推荐事务边界

```text
BEGIN
  校验租户编码/输入
  获取或创建 Account
  创建 Tenant
  Quota 校验
  创建 Owner Member
  创建默认 Role
  创建默认 Group
  创建 Group-Role / Member-Role 等基线绑定
COMMIT

任意步骤失败 -> ROLLBACK
```

### 3.3 必须补充的测试

- `TX-TENANT-001`：创建 Tenant 失败，Account/Tenant/Member/Role/Group 不产生脏数据。
- `TX-TENANT-002`：创建 Owner Member 失败，Tenant 等前序写入全部回滚。
- `TX-TENANT-003`：`seedTenantBaseline` 创建 Role 失败，前序数据全部回滚。
- `TX-TENANT-004`：创建 Group 或 Group-Role Binding 失败，所有 Provisioning 数据全部回滚。
- `TX-TENANT-005`：正常流程成功后，所有基线数据一次性可见且关系完整。

## 4. FIX-021：AddMember 全事务整改

`UserService.AddMember` 当前流程为创建 Member 后再依次绑定 Department、Role。当部门或角色绑定失败时，已创建的 Member 或部分关系可能保留，导致成员处于不完整状态。

### 4.1 整改目标

- Member 创建、Department Binding、Role Binding 必须处于同一个事务。
- 任何 Department/Role 校验或写入失败时，Member 及已完成的绑定全部回滚。
- 跨租户校验、唯一性校验、Quota 校验必须在事务流程中保持一致性。
- 批量绑定时禁止出现 Role1 成功、Role2 失败后 Role1 仍保留的部分成功状态。

### 4.2 推荐事务边界

```text
BEGIN
  resolveAccount
  校验 Member 唯一性
  Quota.Check
  校验 Departments / Roles 属于当前 Tenant
  Create Member
  Bind Departments
  Bind Roles
COMMIT

任意步骤失败 -> ROLLBACK
```

### 4.3 必须补充的测试

- `TX-MEMBER-001`：Member Create 失败，不产生任何关系数据。
- `TX-MEMBER-002`：Department Binding 失败，Member 回滚。
- `TX-MEMBER-003`：Role Binding 失败，Member + Department Binding 全部回滚。
- `TX-MEMBER-004`：多个 Role 中后一个绑定失败，前面已经绑定的 Role 同样回滚。
- `TX-MEMBER-005`：正常成功路径下 Member/Department/Role 关系完整。

## 5. FIX-022：跨租户攻击测试闭环

`ec01adac` 已增加 Member→Role、Member→Group、Group→Role 等跨租户限制测试，并补充 Department/TenantStatus 相关测试。这些改动有效，但当前更多属于 Service + Fake Repository 单元测试，尚不足以证明真实 GORM Tenant Scope、Repository 和 PostgreSQL 链路不会发生越权。

### 5.1 必须补齐的攻击维度

- Member：跨租户 Read / Update / Delete / Role Binding / Group Binding。
- Role：伪造其他 Tenant Role ID 进行 Read / Update / Delete / Binding。
- Group：跨租户 Read / Update / Delete / AddMember / AddRole。
- Department：跨租户 Read / Update / Delete / Member Binding。
- Tenant Status：Disabled/Closed 等状态下禁止继续访问受保护业务能力。
- 所有非法操作除返回拒绝结果外，还必须验证数据库中没有发生实际写入或关系污染。

### 5.2 测试层级要求

保留现有 Service 单元测试，同时新增真实 Repository + PostgreSQL Integration Test。至少应覆盖：

```text
Tenant Context
    ↓
Service
    ↓
Repository
    ↓
GORM Callback / Tenant Scope
    ↓
PostgreSQL
```

建议最终形成不少于 **10 个 `SEC-TENANT-*` 攻击用例**，并纳入 CI。

## 6. FIX-023：Migration Parser / Integration / CI 收口

`ec01adac` 已补强 SQL 分割逻辑，可处理行注释、块注释、字符串中的分号及 `$$...$$` PostgreSQL Dollar Quote。Migration apply 本身也使用事务并具备 checksum 校验。但仍需补齐通用 `$tag$` Dollar Quote 和真实 PostgreSQL Migration 集成验收。

### 6.1 Parser 必须整改

当前不能只识别 `$$...$$`，应支持 PostgreSQL 通用 Dollar Quote，例如：

```sql
$func$ ... $func$
$body$ ... $body$
$procedure$ ... $procedure$
```

Parser 应记录当前 dollar tag，并且只有遇到完全相同的结束 tag 时才退出 Dollar Quote 状态。

### 6.2 Migration Integration Tests

- `MIGRATE-INT-001`：空 PostgreSQL 数据库执行全部 Migration，全部成功。
- `MIGRATE-INT-002`：Migration 全部完成后再次执行，保持幂等。
- `MIGRATE-INT-003`：修改已执行 Migration 内容导致 checksum 改变时必须拒绝启动/迁移。
- `MIGRATE-INT-004`：构造中途失败 Migration，验证 SQL 与 `schema_migrations` 记录全部回滚。
- `MIGRATE-INT-005`：最终 Schema 的关键表、索引、唯一约束、外键与 `db.sql` 设计一致。

### 6.3 CI Gate

建议合并/发布前至少强制执行：

```bash
gofmt check
go vet ./...
go test ./...
# Migration Integration Tests
# Cross-Tenant Integration Tests
```

任何 P0 测试失败均不得进入第一期正式验收完成状态。

## 7. 建议实施顺序

| 顺序 | 任务 | 说明 |
|---|---|---|
| 第一步 | 建立统一事务基础设施 | 先解决 TxManager / Repository WithTx / 事务 Context 传播，否则 FIX-020、FIX-021 会重复改造 |
| 第二步 | 整改 TenantService.Open | 完成 Tenant Provisioning 原子化并补 `TX-TENANT-*` 回滚测试 |
| 第三步 | 整改 UserService.AddMember | 完成 Member + Department + Role 原子化并补 `TX-MEMBER-*` 测试 |
| 第四步 | 补 Cross-Tenant Integration Test | 在真实 PostgreSQL 下验证 GORM Scope/Callback 和 Repository 隔离 |
| 第五步 | 补 Migration `$tag$` + Integration | 完成 Parser 边界、空库迁移、幂等、checksum、rollback、Schema 验证 |
| 第六步 | 接入 CI Gate | 确保 gofmt/go vet/go test/安全与 Migration 集成测试全部通过 |

## 8. 第四次验收通过标准

- [ ] **FIX-020**：Tenant Open 原子事务通过，失败注入测试能够证明无半初始化数据。
- [ ] **FIX-021**：AddMember 原子事务通过，Department/Role 任意失败均无残留 Member/Binding。
- [ ] **FIX-022**：不少于 10 个跨租户攻击用例，且真实 PostgreSQL 集成链路通过。
- [ ] **FIX-023**：Parser 支持通用 `$tag$`；Migration 五类集成场景全部通过。
- [ ] `go test ./...` 全绿。
- [ ] `go vet ./...` 无阻塞问题。
- [ ] `gofmt` 检查通过。
- [ ] CI 将上述 P0 测试设为强制 Gate。

## 9. 本轮不建议继续扩大的范围

当前应优先完成 SaaS Foundation 工程收口，不建议在本轮继续扩大到应用、表单、表单设计、表单数据、Schema/DDL/Data Engine 等低代码业务内核。上述能力应进入后续阶段独立设计和验收，避免影响第一期底座正式结项。

## 10. 最终整改目标

> **整改完成后的目标状态：第一期 SaaS 平台底座 = 正式验收完成。**

其中最重要的验收原则不是“代码存在”或“单元测试数量足够”，而是：

1. 核心写流程具备事务原子性；
2. 多租户边界在真实数据访问链路中不可绕过；
3. Migration 可以在真实 PostgreSQL 环境稳定执行；
4. CI 能持续阻止上述能力发生回归。
