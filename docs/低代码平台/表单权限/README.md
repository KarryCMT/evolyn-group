# 表单权限

应用后台「权限设置」的表单资产权限组专题：普通表单/流程表单的操作权限、字段权限与数据
权限的后端建模、判定与执行点设计。

## 文档

- [表单权限组后端设计方案.md](./表单权限组后端设计方案.md)：权限组四要素（主体 × 操作集 ×
  字段矩阵 × 数据范围）的数据模型（迁移 000058 `asset_permission_groups`）、操作权限字典
  （普通 9 项 / 流程 12 项）、API 与错误码、FormPermissionEvaluator 组绑定判定语义
  （组内合取防串联越权、`form-data:admin` 显式旁路与通配展开注册表、禁用组收口、
  deny-by-default 字段默认、入口判定 view ∨ add、写入路径范围判定、权限感知提交管线、
  守卫式 JSONB 编译与 datetime 四形状文本比较、`V(F)` 规范化取值精确 SQL 模板、
  viewFields/addFields 双矩阵投影、发布阻塞防授权漂移）与
  菜单/提交/运行时/记录查询执行点分期（v2.3 评审修订定版，实施基线）。

## 实现位置

P1 已落地（迁移 000058）：

- 模型/仓储：`apps/evolyn-core/internal/platform/form/model/permission.go`、
  `apps/evolyn-core/internal/platform/form/repository/permission.go`
- 校验器/服务：`apps/evolyn-core/internal/platform/form/service/permission_schema.go`（操作字典 ×
  表单类型、字段矩阵必填协调、数据范围类型分派）、`service/permission.go`（配置面 CRUD +
  字段清单接口）、`service/permission_evaluator.go`（FormPermissionEvaluator 组绑定判定 +
  S7 合并 + 内存版数据范围 NULL 语义匹配）、`service/permission_pipeline.go`
  （权限感知提交管线 `ValidateSubmittedRecordValuesWithPermission`）、
  `service/permission_readsource.go`（switch-type/发布阻塞只读端口）
- 接口：`apps/evolyn-core/internal/platform/form/controller/permission.go`
  （GET/POST/PUT/DELETE `/forms/:code/permission-groups` + GET `/forms/:code/permission-fields`）
- 执行点：SubmitRecord（add 判定 + 权限管线）、GetRuntime（入口 view ∨ add +
  `permissions` 投影 operations/viewFields/addFields）、SwitchType/ Publish 阻塞、
  application 域菜单裁剪窄端口 `FormPermissionDirectory`
- 授权：动作资源注册表（`iam/authorization/actionresources.go`）接入 `form-data`（admin），
  `form-permissions` 配置面资源按管理员规则签名补授（迁移 000058 + 租户开通基线）
- 测试：`service/permission_schema_test.go`、`service/permission_evaluator_test.go`、
  `service/sec_permission_integration_test.go`（SEC-FPERM-001~009 真库集成）、
  application 域 `menu_permission_test.go`
