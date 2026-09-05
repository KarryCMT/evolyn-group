# evolyn-group Agent Guide

本文件适用于整个仓库。进入子目录工作时，先检查该子目录是否另有更近的 `AGENTS.md`；若没有，则遵守本文件。

## 项目总览

本仓库是企业级低代码平台（对标灵衍云形态）的单仓工程，由 Kubernetes 管理平台 weave 二次演进而来。整体架构与演进路线见 `docs/低代码平台/企业级低代码平台技术架构设计.md`（M0–M7 里程碑；当前处于 M1 后段：账号×成员拆分与租户体系补齐的后端全链路已落地，见第 26/27 章与 ADR-006/007）。

- `apps/evolyn-core/`: Go 1.25 + Gin 后端，模块名保持 `evolyn`。当前承载认证（JWT + OAuth）、平台账号/租户成员、部门、分组、自定义 RBAC、租户与套餐配额；低代码引擎（Schema/Query/Permission 等）按里程碑逐步落地。
- `apps/evolyn-web/`: pnpm + Turborepo 前端 monorepo（Vue 3.5 + TypeScript）。主应用在 `apps/web/`（脚手架阶段），共享库在 `packages/`，文档站在 `apps/docs/`。
- `services/`、`packages/`（仓库根）: 规划目录（OpenAPI 契约；原 Java/Flowable 工作流规划已由 ADR-012 取代——流程引擎以 Go 原生落在 `apps/evolyn-core`，见 `docs/低代码平台/流程引擎/`），落地前不要创建同名内容。
- `deploy/`: `docker-compose.yaml` 一键起 PostgreSQL 16/Redis 7/MinIO。
- 根目录 `.golangci.yml`、`.staticcheck-version` 等：仓库级 lint 配置；后端域内另有 `apps/evolyn-core/.golangci.yaml`。

分支约定：当前固定在 `1.0.0` 分支开发，未明确要求前不要切换到其他分支；`2.0.0` 为基线分支，`3.0.0` 为后续规划分支。

## 通用约束

- 项目内不得出现竞品品牌文案、示例名称、域名或注释；涉及品牌展示、产品名称、示例数据及说明时，统一使用“灵衍云”及对应 `lingyanyun` 域名。
- 不要跨子项目混装依赖。前端统一用 pnpm（`apps/evolyn-web` 为 workspace，`pnpm-lock.yaml` 为准，Node >= 22）；Go 命令在 `apps/evolyn-core` 内执行（根 `go.work` 已纳入）。
- 不要提交或手改生成/构建产物：`node_modules/`、`dist/`、`.turbo/`、`bin/`、`cover.out`、`coverage.txt`、swagger 生成的 `apps/evolyn-core/docs/`、`components.d.ts`。
- 证书与私钥不入库（`.gitignore` 已拦截 `certs/`、`*.key` 等）；本地证书用 `apps/evolyn-core/scripts/cert.sh` 生成。
- 真实密钥、token、生产连接串不写入代码或文档。`config/app.yaml` 是本地开发默认值，生产配置通过 `config/app.example.yaml` 复制后填写，不入库。
- 后端模块导入路径是 `evolyn/...`，应用名 evolyn 只是目录名，不要重命名 go module 或批量改导入路径。
- 业务代码一律放 `internal/`（编译器强制外部不可导入）；顶层 `pkg/` 已解散（ADR-007），不要再新建。后续引擎代码放 `internal/engine/`，禁止 import gin/gorm。
- 修改 API 字段、路由、权限模型时，同步检查 `apps/evolyn-web/apps/web/src`（`router/index.ts`、`pages/`）与后端 swagger。
- 遇到工作区已有未提交改动时，默认视为用户改动；只处理当前任务相关文件，不回滚无关文件。
- 当前仍处于开发阶段，实施方案以长期最优的架构、体验、可维护性与一致性为唯一优先级，不受既有实现、兼容包袱或临时过渡方案约束。发现历史设计、接口、数据结构、命名或实现不合理时，应在任务范围内直接重构为新方案，并同步清理已废弃的代码、配置、文档和调用方；仅保留迁移链完整性、数据安全与本文件明确规定的仓库约束。
- 改动的代码需要补充相关注释，简单逻辑简要注释，复杂逻辑详细注释。注释可使用中文。
<!-- - 开始执行或实施前，必须先执行git pull 确保代码是最新的。 -->

## evolyn-core 后端

技术栈与入口：

- Go 1.25，Gin + GORM（Postgres）+ go-redis + swaggo。
- 入口 `cmd/api/main.go`，默认配置 `config/app.yaml`（可用 `--config` 覆盖）。
- 关键配置段：`server`（端口 8080、jwtSecret、rateLimits）、`workflow`
  （service 节点出站调用安全策略：allowPrivateNetwork/allowedHosts）、`db`（库名 `evolyn`；`migrations: true` 启动时应用版本化 SQL 迁移【生产路径】，`migrate: true` 为 GORM AutoMigrate【仅开发/测试】，两者互斥）、`tenant`（注销保留期/清理周期）、`redis`、`oauth`。

目录职责（域模块化定版，ADR-007）：

```text
cmd/api/              入口，只做装配
internal/
  config/             配置解析
  contextx/           通用 context 键值存取（租户上下文/操作者/请求元数据、DetachTenant）
  model/              跨域共享内核（PlatformBaseModel / TenantBaseModel；
                      JSONTime 出网时间统一载体：API 时间字段一律用它，
                      秒级「2006-01-02 15:04:05」东八区，GORM 经 Value/Scan 透明转换）
  metrics/            监控指标
  infrastructure/     postgres/redis/pgx 客户端、GORM 租户 Callback、
                      SQL Migration 执行器（migrate.go）、统一事务
                      （tx.go：TxManager/ResolveDB，ctx 传播事务 session）、
                      ipregion（IP 归属地离线解析：内嵌 ip2region v4 xdb，
                      登录日志「登录地」数据源，IPv6 暂回退「未知」）、
                      生命周期
  testsupport/        集成测试基础设施（TEST_PG_DSN 按需建库/迁移/清理）
  utils/              ratelimit/request/set/trace
  version/            版本信息（Makefile ldflags 注入，路径勿动）
  engine/
    workflow/         流程引擎纯内核（ADR-012，docs/低代码平台/流程引擎/）：
                      禁依赖 gin/gorm/redis/http 及任何 platform 具体实现；
                      DSL v1 协议（model/dsl.go：单文档 JSONB 事实源，
                      不建 wf_node/wf_edge 表）、严格校验器（definition/）、
                      状态机迁移表（task/state_machine.go：状态语义唯一
                      事实源，Phase 0 冻结）、全部 SPI 契约（repository/
                      provider/executor/assignment/expression/event）；
                      平台能力一律经 provider/ 窄端口注入
  platform/
    workflow/         流程引擎平台适配层（ADR-012）：Controller/Service/
                      GORM Repository Adapter/Provider Adapter/Job Worker
                      薄壳装配，依赖方向唯一（platform/workflow →
                      engine/workflow）；事务复用 TxManager/ResolveDB，
                      事件经既有 notification 域 Outbox（禁新建
                      domain_outbox）；API 走 /api/v1 租户中间件链 +
                      httpx.BizError（错误码段见该包 doc.go，前端
                      errorCodes.ts 对齐维护）；Phase 1/2 已落地：
                      迁移 000048（wf_definition + wf_definition_version）
                      + 000049（运行态六表 + 运行实例业务/请求双幂等部分
                      唯一索引），/api/v1/workflows 与 /workflow-instances
                      （发起/详情/同意，行锁推进）全套接口，workflows 资源
                      基线管理员补授、workflow-instances/workflow-tasks
                      授全体成员（TaskActor 实例级校验兜底）；Phase 3 已
                      落地（迁移 000050 tn_departments.leader_member_id）：
                      身份/组织/业务数据三条窄端口适配器（identity/
                      organization/form_provider）+ 条件节点执行器与
                      Expr 发布预编译产物缓存 + form.* 表达式数据源与
                      starter.* 上下文填充 + role/form_field/
                      department_manager Resolver（解析失败租户管理员兜底）
                      + 审批编辑字段权限过滤（同事务经 Form Domain
                      WorkflowRecordStore 窄端口写回 tn_form_records）；
                      Phase 4 已落地（迁移 000051 wf_cc_record）：完整人工
                      Task Engine——Reject terminate 联动/ReturnToStarter+
                      发起人 Resubmit（WAITING_RESUBMIT）/Withdraw 撤回窗口/
                      Terminate 管理员独立权限/Transfer 转办回链/CC 节点
                      执行器（独立抄送记录表），/api/v1/workflow-tasks
                      （列表 scope=pending|completed|cc-to-me、详情上下文
                      含表单快照+字段权限+允许动作+时间线、reject/
                      return-to-starter/transfer）与 /api/v1/
                      workflow-instances（started-by-me、withdraw、
                      terminate、resubmit），WORKFLOW_ACTION_NOT_ALLOWED
                      稳定码；Phase 5 已落地（迁移 000052 wf_job）：
                      DSL approval 节点 timeout/reminder 显式配置（校验器
                      封顶 30 天）+ 任务创建排期/任务与实例终态联动取消 +
                      WorkflowJobWorker（FOR UPDATE SKIP LOCKED，claim+
                      执行同事务 crash 自动回滚，失败重试独立记账回队/
                      FAILED），超时自动 approve/reject 强制经 Task Engine
                      正常路径（AutoTimeout：操作人 0=系统、TIMEOUT 流水、
                      状态机不变）；Phase 6 已落地：workflow.* 事件接入
                      既有 notification 域 Outbox——事件适配器桥接
                      EventPublisher.PublishInTx（同一审批事务写
                      tn_notification_outbox_events，发布失败 best-effort
                      不回滚审批），通知目录追增「审批动态」分类与 7 个
                      消费事件（待办/转办/催办受众=任务参与人快照，
                      通过/驳回/终止/退回受众=发起人，动作
                      open_workflow_task / open_workflow_instance），事件
                      目录追增 workflow.instance.returned /
                      workflow.task.reminder，催办 Job 与 REMINDER 流水
                      同事务发通知事件；task 终态与节点流转事件 V1 仅日志
                      流水（Webhook 为 Phase 7 预留）；Phase 7 已落地：
                      迁移 000053（wf_variable (instance_id,var_key) 唯一
                      + wf_job 约束扩展 service.invoke），DSL service
                      节点 v1（http 动作 + {{expr}} 模板发布期预编译 +
                      responseMapping），节点 Async 挂起排期 service.invoke
                      Job、Worker 独立事务经 ServiceInvoker 窄端口出站
                      调用（SSRF 禁私网/禁重定向/allowedHosts 白名单/
                      Idempotency-Key/响应 1MB 限长/日志脱敏），2xx 响应
                      映射写流程变量（标量收敛）后节点完成并续跑推进，
                      失败经重试记账退避回队；校验器拒绝 authorization/
                      cookie 等敏感头明文入 DSL（密钥由平台侧注入）；
                      Phase 8 已落地：启用 Phase 0 预留的 Execution Tree
                      ——DSL parallel 节点（config.parallel.role=
                      split/join，分支数封顶 10），发布期并行区域分析
                      （definition/parallel.go：分支封闭汇聚同一 join、
                      禁 End/嵌套 parallel/外部入口/分支泄漏，校验器与
                      编译器共用实现并冻结为 CompiledDefinition
                      SplitRegions/JoinRegions 预编译产物），Runtime
                      推进环按执行路径编排（advance 显式携带 execution：
                      发起=根路径、审批/服务续跑/重提交=节点实例所属
                      路径），split 瞬时完成即按出边声明顺序扇出子执行
                      路径（wf_execution.parent_execution_id 执行树，
                      分支内串行推进互不阻塞）、分支进入 join 落到达
                      token（join 节点实例挂分支路径 COMPLETED+分支路径
                      收口）并以实例行锁串行化到达计数（并发推进锁），
                      到达数==分支数由最后到达分支在自身事务内放行回父
                      路径续跑；并行下 Reject/Withdraw/Terminate 既有
                      cancelInstance 链路已覆盖全执行路径取消，含并行
                      定义冻结不支持退回发起人（重提交会二次扇出致
                      join 计数失真，Runtime 拒绝 + 任务详情允许动作
                      投影剔除 return-to-starter 同口径），无新增迁移
                      （000049 已预留 parent_execution_id）；Phase 9 已落地
                      （迁移 000060 wf_definition.form_code 表单绑定 + 引擎
                      Settings.designer 显式声明设计器私有坐标）：LogicFlow 可视化
                      流程设计器——前端 @evolyn.do/workflow 包（DSL v1 协议层/图
                      适配/六类节点画布/属性面板/条件出边编辑/发布 issues 画布定位
                      高亮/版本历史与只读快照预览），主应用流程型表单工作区
                      「流程设计」页（pages/form/workflow.vue）按 formCode 定位或
                      懒建定义并接保存草稿/发布全链路
    server/           HTTP 服务器装配与路由注册（依赖注入汇聚点）
    controller/       Controller 注册契约（RegisterRoute/Name；
                      PlatformController 标记平台运营域归属）与 AppConf
                      应用配置控制器（GET /app/conf：区号列表/能力开关，
                      公开引导端点直挂 api 组，不经租户域 RBAC）
    ginctx/           gin.Context 会话/上下文存取助手
    httpx/            统一响应封装
    middleware/       认证/租户/租户状态拦截/鉴权（平台与租户两条链）/
                      限流/CORS/日志/trace/监控
    auth/             认证域：JWT（claims 含 accountId/memberId/tenantId）、
                       OAuth（github/wechat）、短信验证码（sms/ 子包：Redis
                       存码 + 冷却/试错上限，场景 login/register；开发通道
                       provider=dev 固定码 666666 + devEcho 回显，其他
                       provider 启动拦截）、邮箱绑定验证码（email/ 子包：当前
                       手机号 rebind 验证后签发短时身份凭证，再向新邮箱发码；
                       Redis 原子消费凭证与验证码，dev 固定码 666666，release
                       强制 smtp 通道）、登录口令加密（pki/ 子包：RSA
                       密钥对，公钥经 /app/conf 下发前端 jsencrypt 加密、
                       服务端私钥解密，私钥配置注入或开发态启动随机生成）、
                       service/ 子包（注册编排 RegistrationService：单事务
                       组合 iam 免密注册/账号画像与 tenant 事务内自助开通，
                       复用既有租户保证重试幂等）、auth 控制器（POST
                       /auth/register 为注册向导最终提交「进入产品」：三步
                       纯前端采集的全量数据一次性上送，单事务完成免密注册
                       账号（不设密码，后续个人中心首设）、落账号画像、
                       自助开通租户并绑定 tenant-admin，签发绑定新租户的
                       令牌；已注册手机号等价短信登录（created=false）；
                       POST /auth/tenant 登录态自助开通租户：创建者即所有者
                       并绑定 tenant-admin，编码服务端生成）、loginlog/ 子包
                       （登录日志 000013：会话建立流水，账号维度平台级追加写
                       ——登录/注册即登录各路径令牌签发后 best-effort 落库，
                       登录地写时经 ipregion 离线解析；账号自查分页查询经
                       iam 账号控制器 GET /accounts/me/login-logs 暴露，
                       与 tn_audit_logs（业务操作审计）职责互斥）
    iam/              身份域（域内 controller→service→repository 小三层）：
                       account 平台账号（PUT /accounts/me 自助资料（个人
                       中心入口）含 onboarding JSONB 画像，昵称变更同事务
                       同步当前成员昵称；PUT /accounts/me/password 密码
                       修改——免密注册账号（注册向导不设密码）
                       password_initialized=false（迁移 000012）首设免旧
                       密码；迁移 000010/000011）/ user 租户成员 /
                       group / rbac / department；memberfield 成员信息管理
                       （docs/低代码平台/成员信息管理/，迁移 000031：
                       tn_member_field_settings 租户字段显示策略
                       ——服务端字段注册表 + 乐观锁单字段即时 PATCH，租户
                       开通事务预置默认配置、读取侧幂等兜底；tn_member_profiles
                       正式成员扩展档案——本人按 personalVisible/personalEditable
                       裁剪读写、管理员全量 + cardVisible 服务端裁剪卡片视图；
                       单人邀请 token 接受事务（建成员/迁档案/绑部门/置
                       accepted，注册编排与 POST /auth/invitations/accept
                       两入口））；admingroup 管理组（权限中心-管理员模块，
                       迁移 000032：tn_admin_groups + tn_admin_group_members——
                       内置系统管理员组由 tenant-admin 角色绑定实时推导成员
                       （单一事实源），自定义组经 scope_config JSONB 承载
                       部门/角色/互联组织/应用带范围委托；/admin-groups CRUD
                       + 分区块即时 PATCH + /auth/admin-scopes 身份自查；
                       authorization 拒绝后回落管理组保守门：仅 all 全量范围
                       放行、partial 待各域数据过滤批，admin-groups 资源永
                       不经管理组授予）；authorization/ 自研 RBAC 鉴权
    tenant/           租户域（小三层）：租户 CRUD 与生命周期（注销保留期/
                      Purge Worker）、plan 套餐与配额、QuotaService 配额执行；
                      SelfOpenInTx 供认证域注册编排在外层事务内组合开通
                      （不记审计，由调用方提交后补记）
    audit/            审计域：业务操作审计日志（Recorder，追加写流水）
    application/      应用管理域（M2，小三层，docs/低代码平台/应用管理/）：
                      空白应用创建/列表/详情/更新/软删 + apps 配额（M2-A）；
                      应用菜单只读接口 GET /applications/code/:code/menu
                      （M2-菜单-1，迁移 000016：tn_application_menu_entries
                      节点表 + applications.menu_revision 菜单乐观并发口令，
                      读取走单条 SQL 快照保证修订号与节点同快照；分组创建
                      POST /applications/code/:code/menu/groups 已接入事务、
                      乐观锁、层级校验与审计，其余菜单写管理接口随 M2-菜单-3 落地）
    edition/          版本信息域（一期，小三层，docs/低代码平台/版本信息/）：
                      套餐目录/不可变套餐版本快照/租户订阅/特批权益覆盖
                      （迁移 000030，四表 + 三档 seed + 存量回填）；活动订阅
                      是权益事实源，pf_tenants.plan/quotas 退为 QuotaService
                      过渡兼容投影（同事务同步）；租户侧 GET /editions/current
                      （editions:get 仅授租户管理员）+ 平台侧人工授予/取消/
                      可授予版本列表；读时到期投影 expired + EditionWorker
                      幂等降级免费版；QuotaService 经到期守卫 GuardLimit 在
                      降级落库前按「免费快照+仅 manual 覆盖」拦截；租户开通
                      经 SubscriptionSeeder 同事务补种初始订阅
    form/             表单资产域（ADR-010，docs/低代码平台/表单设计器/）：表单资产与
                      草稿（forms.draft_content 目标保存协议全文 + draft_revision
                      乐观锁）、不可变发布快照（tn_form_versions：version_no +
                      schema_revision 双口令）、记录提交（tn_form_records：按快照终审，
                      错误按 widgetName 回填）；服务层含与前端逐字一致的协议校验器
                      与基础字段值校验器（schema.go/value.go 镜像 TS 字典）；
                      权限资源 forms（管理员）/form-records（全体成员提交）、
                      forms 配额键（QuotaService 计数器注入）；M2-资产-1 最小
                      纵切：创建/改名/删除同事务维护 tn_application_menu_entries
                      的 form 节点并递增 menu_revision，菜单读侧经 FormDirectory
                      窄端口做存在性裁剪与 target 投影（跨域双向窄端口装配）；
                      000044 将 form_type（standard/workflow）固化为创建后不可变
                      的表单类型事实源；000045 将 form_ 前缀 code 固化为路由/API/
                      菜单 target 的稳定公开标识，内部自增 ID 不出网；ADR-011
                      放宽 form_type 可切换（form-actions:switch-type，流程数据
                      保留）并落菜单按钮动作授权（动作注册表
                      iam/authorization/menuactions.go + form-actions/menu-
                      favorites 资源，菜单读侧投影 actions 按钮图）；迁移
                      000037/000038/000044/000045/000046/000047；记录列表
                      （POST /forms/:code/records，Query DSL JSON body）：服务端
                      编译参数化谓词合并行级 view 范围，字段白名单=发布快照
                      field_mappings ∪ 系统字段命名空间 sys.submittedBy/
                      sys.submittedAt/sys.updatedAt（record_system_fields.go，
                      操作符矩阵与前端 @evolyn.do/query 镜像；排序仅开放系统
                      字段，仓储恒定追加 id DESC 稳定尾排序）；迁移 000067 为
                      tn_form_records 落系统字段列（submitted_by_name 提交时
                      固化展示名快照 + updated_at 最后写回时间，存量回填；
                      UpdateValues 同语句刷新 updated_at）；表单权限组
                      （P1，docs/低代码平台/表单权限/，迁移 000058）：
                      tn_asset_permission_groups + subjects 两表承载主体×操作集×
                      字段矩阵×数据范围整体授权单元，FormPermissionEvaluator
                      组绑定判定（组内合取防串联越权、form-data:admin 显式旁
                      路、禁用组收口、deny-by-default 字段默认、入口判定
                      view ∨ add）+ 权限感知提交管线 + 运行时 permissions 投
                      影（viewFields/addFields 双矩阵）+ 菜单裁剪窄端口
                      FormPermissionDirectory + switch-type/发布阻塞；配置面
                      form-permissions 与数据面 form-data 动作资源注册表
                      （iam/authorization/actionresources.go）按管理员规则签
                      名补授；字段显隐规则（v5，本目录
                      字段显隐规则设计方案.md）：content.fieldShowRules 表单级
                      唯一事实源（条件树×目标字段，12 种值语义控件作条件源，
                      方法×值形态矩阵封闭），前后端镜像结构校验器+依赖图环检测
                      （schema.go/validate.ts）、纯求值器（service/rules.go 与
                      packages/form schema/rules.ts，条件源不可见即条件不成立），
                      提交终审按动态可见性复核信封 visible（含权限管线合成），
                      运行时 setValue 沿编译产物定向重算下游可见性；不可见字段
                      赋值（v6，本目录 不可见字段赋值前后端设计方案.md）：
                      content.submitRule 默认策略 + widget_submit_rules 字段级
                      例外（键=顶层值语义 widgetName，值 1 保持原值/2 空值/
                      3 始终重新计算，执行器交付前 3 不可配置），前后端镜像
                      校验器（schema.go/validate.ts）+ 策略/空值/值决议纯服务
                      （invisible_value_policy.go/visibility.go/
                      value_resolution.go，与前端 schema/invisible-value-policy.ts
                      同语义）；信封 visible 升级为有效可见性（静态∧权限∧规则），
                      有效不可见字段禁携 data、按 clear 类型化空值/preserve 锁定
                      基线/recompute 执行器决议，新建与流程写回（记录基线+受信
                      patch）共用 ResolveSubmittedValues/ResolveMergedRecordValues，
                      v6 前快照保持旧静态可见语义
  tenantproduct/    产品中心域（一期，小三层，docs/低代码平台/产品中心/）：
                      平台产品目录/租户产品配置/部门与成员范围关联（迁移
                      000033，四表 + lingyanyun seed + 存量租户回填，目录是
                      平台级资源）；租户侧 GET /tenant-products 卡片列表 +
                      PUT /:code/enabled 启停 + PUT /:code/access-scope 范围
                      全量替换（revision 乐观锁 + 统一事务 + 提交后审计；
                      tenant-products:view/update 仅授租户管理员）；版本投影
                      经窄端口只读消费 edition 域；TenantProductAccessEvaluator
                      服务端访问判定（目录 active → 租户启用 → 有效成员 →
                      all/命中直接成员或部门含子部门），产品受保护路径接入
                      时构造使用；租户开通经 ProductConfigSeeder 同事务初始
                      化默认配置（enabled=true、scope=all）；000034/000035 权限
                      补授兜底：基线管理员角色名可被改名（历史 000030-000033
                      均按名补授会漏），改按「members:*+roles:*+departments:*」
                      规则签名补授 tenant-products/editions/member-field-
                      settings/admin-groups，与角色名无关
    enterpriselog/    企业日志域（一期，小三层，docs/低代码平台/企业日志/）：
                      管理后台 /tenant/enterprise-logs 的登录日志/操作日志
                      只读查询与导出编排（迁移 000036：pf_login_logs 补
                      actor_name_snapshot、tn_audit_logs 补 event_code/
                      category_code/快照/summary 展示投影 + 租户维度索引 +
                      tn_enterprise_log_exports 导出任务表）；不接管登录/审计
                      写入（分属 auth/loginlog 与 audit 域），仓储显式租户
                      条件 + JOIN tn_users/accounts 显示名兜底 + keyset 导出
                      扫描；GET /enterprise-logs/login|operations|operation-
                      categories + POST/GET /enterprise-logs/exports（同步
                      生成 CSV、24h 有效、单次上限 5 万行；enterprise-logs:
                      view/create 仅授租户管理员，下载路径复核 create 动词，
                      不经管理组放行）；既有审计调用方零改动——audit 域事件
                      注册表（service/events.go）按 module+resourceType+action
                      推导事件码/分类/脱敏摘要，存量历史行读取侧降级
                      「历史操作记录」
  productlog/       产品日志域（一期，小三层，docs/低代码平台/产品日志/，
                      迁移 000064）：管理后台「产品日志」只读与导出编排——
                      tn_audit_logs 按产品分类白名单（audit 注册表
                      product_events.go：application/application_menu/form/
                      workflow/data/app_permission 六分类，与企业日志目录
                      category_code 互斥、企业日志查询侧排除产品分类）受控
                      投影读取 + tn_product_log_exports 导出任务表（一期同步
                      生成 CSV、24h 有效、上限 5 万行）；审计事实源仍在 audit
                      域：Entry 扩展应用维度快照三元组（application_id/code/
                      name_snapshot 写时固化，应用删除后历史展示不失真），
                      application/menu/form（含权限组与记录提交）/workflow
                      写路径已接入产品事件（资源级动词覆盖，如「创建表单」）；
                      API：GET /product-logs（分类/事件/成员/应用/关键词/
                      日期筛选，keyset 导出扫描）+ /product-logs/options
                      （分类事件码+操作人+有效应用，服务端下发不硬编码）+
                      /product-logs/exports（创建/状态/下载复核
                      product-logs:export）；product-logs:view/export 仅授
                      租户管理员（规则签名补授，不经管理组放行）；导出行为
                      落企业治理类审计（productlog.export.create）；前端：
                      api/productLog.ts + pages/tenant/product-logs.vue
                      （VTable 列表接真实接口）
  notification/     消息中心域（P1+P2，小三层，docs/低代码平台/消息中心/）：
                      租户×成员站内收件箱与租户通知设置（迁移 000039 七表：
                      不可变 tn_notification_messages（纯文本快照物化时固化，
                      (tenant_id,event_id) 唯一幂等）+ notification_member_
                      inboxes（扇出/已读，查询显式 tenant_id+member_id 双
                      条件，过期消息 SQL 侧排除）+ 设置聚合/偏好覆盖/接收
                      规则/自定义提醒对象 + tn_notification_outbox_events）；
                      业务域经 EventPublisher.PublishInTx 在自身事务内写
                      Outbox（application 域创建/删除应用已发布
                      application.asset.changed 真实事件），Dispatcher 以
                      FOR UPDATE SKIP LOCKED 小批领取、按事件目录（八个
                      稳定分类 + 首批五个 app-log 事件，模板纯文本渲染 +
                      受控动作码）解析接收人（event_actor/event_audience/
                      tenant_admin 实时推导，custom 仅外部渠道）并同事务
                      扇出；Outbox 物化 Worker + 六个月保留清理 Worker 随
                      服务启停；API：GET /notifications/unread-summary、
                      GET /notifications（游标分页）、PUT /notifications/
                      :id/read|read-all（through 口令不误伤新消息）与
                      GET/PATCH /notification-settings*、recipients CRUD
                      （revision 乐观锁 409、联系人被引用 409 +
                      usedByEventCodes、上限 200 配置）；权限 notifications:
                      view/update 授全体成员（只能读写本人收件箱）、
                      notification-settings:* 按管理员规则签名补授（不经
                      管理组放行）；租户开通经 NotificationSettingSeeder
                      预置聚合根；前端：api/notifications.ts、api/
                      notificationSettings.ts、stores/notification.ts（未读
                      摘要会话级单一事实源，两个顶栏共读）、消息中心抽屉
                      全量接入（游标增量分页、事件筛选目录、action 白名单
                      跳转、接收对象选择器、409 冲突重载）；审批动态分类
                      （approval）+ workflow.* 事件随流程引擎 Phase 6 接入
                      （目录注册 + Dispatcher 扇出，受众由 workflow 事件
                      适配器解析显式成员）；邮件/短信外部渠道与云币计费
                      为 P3
migrations/           版本化 SQL Migration（Schema 唯一事实来源，嵌入二进制；
                      命名 NNNNNN_name.(up|down).sql，版本号只增不复用。
                      000063 起业务表名统一命名空间前缀：pf_ 平台 / sys_ 系统 /
                      tn_ 租户 / wf_ 流程引擎（方案见 docs/低代码平台/数据库表
                      命名空间前缀调整方案.md）；000063 之前迁移文件内保留当时
                      旧表名，勿据其回改代码；索引/约束/序列名同规则带前缀）
scripts/              db.sql（终态快照，与迁移链一致）、cert.sh（本地证书）
```

常用命令（在 `apps/evolyn-core` 内执行）：

```bash
make run            # 本地运行（需 postgres/redis 就绪，读 config/app.yaml）
make build          # 构建到 bin/evolyn-core（注入版本 ldflags）
make test           # 单测 + 覆盖率（离线全绿，真库集测自动跳过）
make test-integration  # 复用本地 postgres（见下方「本地开发/测试数据库约定」）
                    # 以 TEST_PG_DSN 跑全部测试（含 SEC-TENANT-*/MIGRATE-INT-* 真库集成用例）
make vet            # go vet ./...
make fmt            # gofmt -s 格式化
make lint           # golangci-lint（需已安装）
make swagger        # 重新生成 swagger 到 ./docs（gitignored，本地查阅用）
make postgres       # 复用本地 postgres 容器并导入 scripts/db.sql
make redis          # 起本地 redis 容器
```

本地开发/测试数据库约定（固定，勿改）：统一复用本机已存在的 `postgres`
容器（`postgres:14-alpine`，`127.0.0.1:5432`，账号/密码 `postgres/postgres`）。
Makefile 的 `PG_CONTAINER`/`PG_IMAGE`/`PG_HOST`/`PG_PORT`/`TEST_PG_DSN`
默认已指向它，端口已有实例监听时直接复用，不要另建容器或拉其他镜像 tag。
集成测试 DSN 连 `postgres` 库（testsupport 按需建/清临时测试库，零残留）；
业务库 `evolyn` 由 `make postgres` 导入 `scripts/db.sql` 快照。该连接仅用于
开发与测试，生产连接串经 `config/app.example.yaml` 复制填写，不入库。

后端改动规则：

- 域模块化：新资源归属既有域（iam/tenant/auth/audit）或按第 27 章 ADR-007 的结构新
  建域，域内遵循 controller → service → repository 小三层，同步补各层
  `interface.go` 与 `internal/platform/server/server.go` 装配。
- 多步写流程必须走统一事务边界（整改 FIX-020/021）：Service 依赖 TxManager
  接口（`WithinTransaction`），Repository 一律经 `infrastructure.ResolveDB`
  取连接以加入 ctx 传播的事务；跨租户 Update/Delete 必须先加载再写，
  禁止「租户过滤 0 行影响却返回成功」；审计在事务提交后独立写入（best-effort）。
- 账号×成员拆分（ADR-006）是现行模型基线：登录身份（name/phone/password/
  OAuth 凭证）只挂 `pf_accounts`；租户内身份（昵称/部门/分组/角色）挂 `tn_users`。
  迁移期字段残留由启动幂等回填处理，新代码不要声明旧列。
- 路由双域隔离（整改 FIX-008）：`/api/v1/platform/*` 只走
  Authentication + PlatformAuthorization（无租户上下文），平台控制器实现
  `Platform() bool` 标记；其余租户域路由走 Tenant → TenantStatus →
  Authorization 链。关系绑定（角色/分组/部门）必须经 Service 层同租户校验，
  不允许裸 ID 盲写关系表。
- 数据库结构变更以 `migrations/` 版本化 SQL 为唯一事实来源（整改 FIX-009）：
  改 GORM Model 必须同时提交 up/down 迁移并同步 `scripts/db.sql` 快照；
  新增数据库表及新增字段必须在迁移 SQL 中添加中文注释；约束/索引名与迁移文件保持一致以保证快照库重放幂等。AutoMigrate 仅限
  开发/测试（`db.migrate`），禁止依赖其建生产 Schema。
- swagger 注释一律中文：新增/修改接口时 `@Tags`（模块名）、`@Summary`、
  `@Description` 等注释统一写中文，不得新写英文；模块 tag 沿用既有中文名
  （首页/认证/账号/成员/部门/分组/角色权限/平台管理），新模块也起中文名。
  改完执行 `make swagger` 重新生成文档；docs 包经 server.go 的 blank import
  编译进二进制，联调需重启服务生效。
- 业务错误必须走 `httpx.BizError`（ADR-008）：域服务用 `httpx.NewBiz`
  定义稳定码常量（码表注释即文档）、`httpx.Wrap` 附加原始错误（只入日志）；
  禁止裸 `errors.New` 的错误文本出网，禁止在 msg 携带内部数据（租户 ID/
  SQL/用量数值）。前端按 `errCode` 分支（`packages/utils/src/request/errorCodes.ts`
  对齐维护，经 `@evolyn.do/utils` 导入），禁止 message 文本匹配；新接口 swagger
  `@Failure` 注明 errCode。
- 提交前至少运行 `go test ./...`（或说明未运行原因），格式化使用 `make fmt`。

## evolyn-web 前端

技术栈与结构（pnpm + Turborepo monorepo，TypeScript 全量）：

- 主应用 `apps/web/`（`@evolyn.do/web`）：Vue 3.5 + Vite + Element Plus
  （`unplugin-vue-components` 按需自动导入，`components.d.ts` 勿手改）+ Sass，
  路径别名 `~/` 指向 `src/`。
- 共享库 `packages/`：`ui`（组件库）、`form`（表单三入口，ADR-010 目标保存协议
  `content.items[].widget` 为唯一事实结构：`@evolyn.do/form/schema` 纯 TS 协议层
  （27 种 widget.type 字典/严格校验器 JSON Path 级错误/深拷贝/迁移器/基础字段值
  编解码，与后端 internal/platform/form 校验器逐字一致）、`@evolyn.do/form/runtime`
  最终渲染器（按 `widget.type` 注册组件、值按 `widgetName` 取键、enable/visible/
  labelHidden/lineWidth 静态执行、提交双口令，P2 基础 9 类已落地）、
  `@evolyn.do/form/designer` 设计器入口（素材/画布/属性面板组件与
  useFormSchemaEditor，页面状态即协议文档），主入口 `@evolyn.do/form` 为兼容期
  别名；运行时样式独立走 `@evolyn.do/form/runtime/style.css`，不得静态引入设计器
  或重型字段模块；发布白名单 PUBLISHABLE_WIDGET_TYPES 前后端各一份保持一致，
  tsdown 构建必须保持单图（见 tsdown.config.ts 注释，分组双写会丢公共导出）、
  `schema/invisible-value-policy.ts` v6 不可见字段赋值策略解析与客户端预演
  （设计器区块/对话框与运行时信封共用，禁在组件内分散写策略判断））、
  `workflow`（流程可视化设计器，Phase 9：`schema/` 为 Workflow DSL v1 前端协议层
  （types 与后端 internal/engine/workflow/model/dsl.go 逐字对齐、lifecycle 不可变
  文档操作、validate 即时校验镜像后端校验器错误码）、`adapters/graph.ts` DSL↔
  LogicFlow 投影（画布坐标持久化于 settings.designer.layout，分层自动布局兜底）、
  `designer/` LogicFlow 画布 + 素材面板 + 按节点类型属性面板（审批人/会签/字段
  权限/条件出边/服务调用）+ 只读预览与校验错误高亮；铁律 LogicFlow Graph !=
  Workflow Runtime Model，LogicFlow 只负责编辑体验）、
  `utils`、`hooks`、`directives`、
  `lint-configs`（eslint/prettier/stylelint/commitlint/typescript 配置）。
- 文档站 `apps/docs/`（VitePress）。依赖版本走 `pnpm-workspace.yaml` 的
  `catalog:` 统一管理。
- 主应用目录：`src/main.ts`（入口）；`src/router/index.ts`（**手动路由表**，
  与 `src/pages/` 目录一一对应，新增页面在此登记；`meta.public` 标记公开页，
  全局守卫拦截未登录访问）；`src/pages/`（页面，`auth/` 为登录/注册/找回密码）；
  `src/api/`（HTTP 层：`http.ts` 统一 fetch 封装 + 按域拆分的接口模块，
  对齐后端统一响应 `{code,msg,data}`）；`src/components/auth/`（认证域业务组件：
  AuthLayout 骨架、登录/注册表单等）；`src/stores/`（pinia 全局 store：`auth.ts`
  会话 store 为 token/登录聚合信息唯一事实来源，pinia 在 main.ts 中先于 router
  装配）；`src/composables/`（消费入口与局部状态：`auth.ts` 为 auth store 的
  useAuth() 只读适配、dark 暗色）；
  `src/styles/`（element 主题覆盖，默认 `el` 命名空间）。布局与菜单组件
  （侧栏/顶栏）随后续前端业务批落地。

常用命令（在 `apps/evolyn-web` 内执行，需 pnpm >= 10）：

```bash
pnpm install                        # workspace 全量安装（only-allow pnpm）
pnpm -F @evolyn.do/web dev          # 主应用本地开发
pnpm -F @evolyn.do/web typecheck    # vue-tsc 类型检查
pnpm -F @evolyn.do/web build        # 生产构建
```

前端改动规则：

- 使用 Composition API（`<script setup>`）+ TypeScript；`typecheck` 必须通过。
- 前端界面还原截图时，统一以 1080P（1920×1080）画布尺寸为基准进行还原。
- 前端代码调整时，不进行任何build，由用户手动执行，只检查类型语法代码错误。
- 前端界面还原必须优先使用element-plus组件库提供的组件，utils方法。
- 前端在进行页面开发时，需要善于发现某些组件可以复用就做成通用组件，并放到@evolyn.do/ui里面，统一管理维护。
- 前端在进行页面开发时，必须需要适配暗黑主题模式，避免出现写死的颜色值。
- 前端在进行页面开发时，页面样式中文本边框、填充、基础、背景、字体大小、间距、阴影、圆角，必须从`docs/低代码平台/前端中性色变量/变量表.md`取对应值CSS变量。
- 代码统一以 Prettier 格式化（约束）：以 `apps/evolyn-web/prettier.config.mjs`
  （复用 `@evolyn.do/prettier-config`）为准，改完前端代码在 `apps/evolyn-web`
  内执行 `pnpm run lint:format`（等价于编辑器 Prettier - Code formatter 插件
  保存格式化），不自定义其他格式化风格或手工对齐。
- 样式统一引用 Element Plus 默认命名空间 CSS 变量（`--el-*`），不自定义 CSS
  token；类名采用 BEM（块\_\_元素--修饰符），优先组件内 scoped 样式。
- 管理后台的选中态、操作态、状态标签和强调元素必须使用当前主题色相关变量
  （如 `--el-color-primary` 及其 `light-*` 色阶），禁止在后台页面局部硬编码品牌色或覆盖主题色变量。
- 前端样式统一使用 SCSS 编写；新增或修改样式时不要引入 CSS、Less 等其他预处理器。
- 页面存在列表滚动时，只允许列表内容区域滚动；页面标题、筛选/搜索栏、工具栏、分页等上下文操作保持固定。必须使用 Element Plus 的 `el-scrollbar` 实现列表滚动，禁止以页面容器或原生 `overflow: auto/scroll` 代替。
- 所有可点击的图标、文字和原生 `button` 都必须提供清晰的鼠标悬停背景效果，并设置
  `cursor: pointer`；禁用态元素不适用此规则。
- 图标优先使用 `@remixicon/vue` 提供的 Fill 类型图标，禁止使用 Line 类型图标。
- 开发时优先从 `apps/evolyn-web/packages/` 查找可复用的模块、方法与 hooks；若没有合适实现，
  先由用户决定是否新增实现，禁止自行造轮子。
- 抽屉标题栏统一采用标准规格：桌面端高 `56px`、移动端高 `52px`，标题为 `18px/26px`，
  关闭按钮为 `32px`（图标 `22px`）；自定义 header 与 Element Plus 默认 header 均须遵守。
  抽屉内容若传送至 `body`，以组件唯一块类限定全局覆盖，禁止影响其他抽屉。
- 新增页面在 `src/pages/` 建文件并到 `src/router/index.ts` 登记（不用文件式
  自动路由）；布局/菜单组件落地后，菜单入口同步维护对应布局组件。
- 数据表格统一采用 VisActor VTable 生态（见架构文档第 3 章），新列表页不要
  引入其他表格库。
- 组件库改动在 `packages/ui` 内进行并通过 changeset 发版，主应用优先复用
  `@evolyn.do/*` workspace 包，不重复引第三方实现。
- 前端低代码基础能力的分层、职责与引用以
  `docs/低代码平台/基础引擎能力/基础引擎能力职责边界与引用规范V1.md` 为准：
  `packages/engines/*` 是无业务页面、尽量无 UI、可独立测试的基础 Engine Layer；
  `form`、`dashboard`、`workflow` 是组合 Engine 的领域层，Engine 严禁反向引用领域层。
- Engine 只能通过包公开入口和稳定的 types/DTO/AST/Context/Result/Diagnostic 协作，
  禁止跨包深层导入 `src/internal`、`src/private` 等未公开实现；新增或修改跨 Engine
  协议必须确认唯一权威来源、版本/迁移策略，并同步维护前后端一致性 Fixture。
- `packages/engines/*` 原则上不得依赖 Vue SFC、Element Plus、Router、页面级 Pinia
  Store 或 `@evolyn.do/ui`；错误以结构化 Result/Diagnostic 返回，禁止在 Engine 内直接
  Toast、操作 DOM/组件实例或 monkey patch 内部状态。领域包相互协作默认走 Engine 协议、
  共享 DTO 或上层 Orchestrator，禁止形成任何直接或间接循环依赖。
- 浏览器侧的 Permission Engine 仅负责授权结果投影和交互控制，后端仍是数据、字段、
  记录权限的最终授权边界；Physical Engine 仅描述存储模型和迁移意图，严禁在浏览器生成、
  连接或执行 DDL；Query Engine 只构造和校验 Query DSL，严禁生成或执行最终 SQL。
- Rule、Formula、Validator、Query 等核心语义必须保持确定性；涉及规则、公式、校验、
  权限或查询协议时，应设计为可与 Go 后端镜像校验，不能因前端存在同名 Engine 而转移
  后端权威职责。

## 验证建议

- 前端改动验证以 `pnpm -F @evolyn.do/web typecheck && build` 为准，后端以 `go test ./...` 为准；用户约定**无需用浏览器执行 E2E 走查**，联调由用户自行进行。
- 后端：`cd apps/evolyn-core && go test ./...`，必要时 `make lint`、`make run`。
- 前端：`cd apps/evolyn-web && pnpm -F @evolyn.do/web typecheck && pnpm -F @evolyn.do/web build`；联调用 `deploy/docker-compose.yaml` 起依赖。
- 跨端接口：字段名、错误码、鉴权头在 evolyn-core 控制器与前端调用侧两侧核对（前端 HTTP 层已随登录批落地：`apps/web/src/api/`，开发期 Vite 将 `/api` 代理到本地 8080）。

## 文档约定

- `docs/低代码平台/`：平台架构设计专题，含设计基线（第 1–21 章选型、第 22 章起顶层定版）与 README 导航；重大技术决策以追加 ADR 方式记录在第 27 章，不回改已定版结论原文，被取代表述在原章节就地标注指向关系。
- 其他业务文档按子目录组织并配 `README.md` 导航，不散落根目录。
- 移动或改名文档用 `git mv`，同一次改动内修复引用；整理只调结构，不改写业务结论。

## 文档维护

- 新增子项目、脚本、关键目录或里程碑落地时，同步更新本文件与 `docs/低代码平台/README.md` 的实现位置映射。
- 若某个子目录需要更细的约束，可在该子目录新增 `AGENTS.md`，范围只覆盖该目录及其子目录。
