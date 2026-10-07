## Context

当前仓库已经具备三块可以复用但职责不同的基础：应用菜单和表单资产已经实现“先创建资产、再进入设计器”的事务链路；`@evolyn.do/dashboard` 已提供 GridStack 布局、组件描述和 VChart/VTable 渲染壳；`@evolyn.do/query` 与表单记录列表已经定义部分查询语义。缺口在于业务仪表盘尚无独立资产、草稿、发布、权限和服务端查询模型，现有 `/workbench` 则是租户级企业工作台，不能作为业务仪表盘事实源。

本变更覆盖从预创建到高级交互的完整纵向链路。实现按五个阶段推进，每个阶段都必须保持前一阶段可运行，并以服务端资产与不可变版本作为最终事实源。前后端协议、权限资源和菜单投影同时演进，不引入只在浏览器生效的临时数据能力。

## Goals / Non-Goals

**Goals:**

- 让仪表盘成为与表单同级的应用资产，具备稳定公开编码、草稿、菜单节点、权限、配额、审计和软删除生命周期。
- 提供独立、可刷新恢复的设计工作区，可靠处理空画布、保存、并发冲突、预览和未保存离开保护。
- 定义前后端镜像的版本化仪表盘协议与 Dataset/Query DSL，首批可靠支持统计图和明细表。
- 让预览和运行时只通过服务端权威查询访问数据，并在每次查询重新应用发布范围、表单权限、字段权限和行级数据范围。
- 通过不可变发布版本稳定线上行为，并在应用菜单和通用资产路由中完成运行时接入。
- 在同一协议上扩展全局筛选、组件联动、移动端布局、应用内复制和发布范围。

**Non-Goals:**

- 本期不支持跨表关联、跨应用数据集、多数据源 Join、用户编写 SQL 或第三方图表原生配置透传。
- 本期不建设 OLAP/ClickHouse、查询结果缓存、异步查询、数据导出、订阅推送或实时流式刷新。
- 本期不提供匿名公开分享、跨租户分享、跨应用复制或外部嵌入认证协议。
- 本期不把业务仪表盘并入 `/workbench`，也不迁移或改变企业工作台已有文档和 revision。
- 首批组件不追求完整图表品类；统计图与明细表通过统一描述符扩展，后续组件沿同一注册表接入。

## Decisions

### 1. 业务仪表盘使用独立平台域

新增 `internal/platform/dashboard`，采用 controller → service → repository 的域内分层。控制器只处理 HTTP DTO、会话与错误映射；服务层负责事务、授权后的业务不变量、协议终审和跨域窄端口；仓储只处理仪表盘表。该域通过装配层消费以下端口：

- `MenuMaintenance`：创建、更新、移动和删除仪表盘菜单节点，并推进应用菜单 revision。
- `DashboardDirectory`：供菜单读取侧批量确认资产存在、发布状态、目标路由和当前成员可见性。
- `FormDataCatalog`：读取表单发布字段目录与字段能力，不直接引用 form 仓储。
- `DashboardQueryExecutor`：执行已经校验的查询计划，并复用表单记录数据范围与物理存储能力。
- 既有 `TxManager`、`QuotaService`、审计 Recorder 和身份/组织主体解析端口。

依赖方向保持 `platform/dashboard → engine/query`，纯查询内核不得依赖 Gin、GORM、Redis 或具体 platform 域。`/workbench` 继续使用当前工作台服务、权限和存储，dashboard 域不得读写其配置。

备选方案是把仪表盘作为表单子类型或继续存入工作台配置。前者会把无记录语义的资产错误绑定到表单发布模型，后者无法提供应用级菜单、版本和数据授权，因此不采用。

### 2. 资产、草稿和发布版本的数据模型

使用版本化 SQL migration 新增：

- `tn_dashboards`：租户、应用、稳定公开 `code`、名称/图标/颜色、协议版本、全量 `draft_content` JSONB、`draft_revision`、当前发布版本指针、创建人、软删除与标准审计时间。公开 API 永不暴露内部自增 ID。
- `tn_dashboard_versions`：仪表盘 ID、严格递增 `version_no`、来源草稿 revision、协议版本、规范化文档、内容 checksum、发布人和发布时间；`(dashboard_id, version_no)` 唯一，创建后禁止更新内容。
- `tn_dashboard_version_subjects`：按发布版本存储 `all` 或 member/department/role/group 主体。主体定义随版本冻结，但访问时仍按最新有效组织关系求值。
- 持久化创建幂等绑定：在租户、操作者和 `request_id` 范围唯一，同时记录规范化请求 hash 与结果 dashboard ID。相同请求返回原结果，不同 payload 返回幂等冲突。

空草稿不是 `NULL`，而是当前协议版本的最小合法文档，包含空 datasets/widgets/filters/interactions 和默认桌面设置。移动端布局可以缺省。草稿保存是携带 `expectedRevision` 的全量替换，SQL 更新条件包含当前 revision；受影响行数为 0 即返回 HTTP 409。发布在同一事务内再次锁定资产并比较 revision，写不可变版本后更新当前版本指针。

迁移为现有租户套餐目录增加 `dashboards` 配额键；初始权益建议为免费版 3、试用版 20、专业版无限，最终 seed 值应与产品权益目录在实施时一次确认。计数只统计未删除资产，创建和复制在事务中执行并发安全的 quota guard。

### 3. 预创建与菜单维护必须共事务

`POST /api/v1/apps/code/:appCode/dashboards` 接收 `requestId`、名称和可选 `parentMenuCode`。服务层在一个 `TxManager` 事务中：

1. 校验应用、父分组和创建权限；
2. 解析或创建幂等绑定并执行 quota guard；
3. 分配稳定 `dashboard_` 公开编码并插入资产与最小草稿；
4. 通过 `MenuMaintenance.AttachDashboardNode` 插入菜单节点；
5. 只推进一次应用菜单 revision。

任一步失败都回滚资产、菜单、幂等结果和配额占用。改名、图标/颜色更新、移动和软删除沿用相同事务原则。删除同时摘除菜单节点和该节点个人收藏；发布历史保留用于审计但不再可访问。审计在成功提交后记录，发布和查询等需要同事务证据的事件使用既有可靠事件机制。

### 4. API 按管理态、预览态和运行态分面

管理态接口位于租户 API 链并要求 dashboard 资源授权：

- `POST /apps/code/:appCode/dashboards`：预创建。
- `GET /dashboards/:code`：详情、草稿、revision 与发布摘要。
- `PATCH /dashboards/:code`、`DELETE /dashboards/:code`：展示信息与软删除。
- `PUT /dashboards/:code/draft`：全量草稿乐观锁保存。
- `POST /dashboards/:code/publish`：基于 `expectedDraftRevision` 发布。
- `POST /dashboards/:code/copy`：同应用复制到根级或目标分组。
- `GET /dashboards/:code/data-sources/forms` 与 `GET /dashboards/:code/data-sources/forms/:formCode/fields`：设计态数据源和字段目录。
- `POST /dashboards/:code/widgets/:widgetId/preview-query`：按指定已保存草稿 revision 查询。

运行态接口按应用、资产和发布版本定位：

- `GET /apps/code/:appCode/dashboards/:code/runtime`：返回统一 bootstrap。
- `POST /apps/code/:appCode/dashboards/:code/widgets/:widgetId/query`：只接受发布版本、组件 ID、声明过的筛选/联动参数和分页游标。

查询接口不接受客户端提交 Dataset、字段列表、聚合或任意 Query AST；服务端从已保存草稿或不可变发布版本恢复可信语义。稳定错误码至少覆盖 `DASHBOARD_NOT_FOUND`、`DASHBOARD_SCHEMA_INVALID`、`DASHBOARD_DRAFT_CONFLICT`、`DASHBOARD_PUBLISH_CONFLICT`、`DASHBOARD_IDEMPOTENCY_CONFLICT`、`DASHBOARD_UNAVAILABLE`、`DASHBOARD_QUERY_INVALID` 和 `DASHBOARD_QUERY_LIMIT_EXCEEDED`，并复用统一禁止访问、菜单父节点无效和配额错误。所有 DTO 同步到 Swagger、前端类型和 `errorCodes.ts`。

### 5. 仪表盘文档保存平台语义，不保存渲染器私有对象

`@evolyn.do/dashboard` 定义可判别联合的版本化文档：

- `settings`：主题、画布、刷新和渲染约束。
- `datasets`：稳定 ID、表单公开编码、基于 `@evolyn.do/query` 的 filters/groupBy/aggregates/sorts/projection 与受控限制。
- `widgets`：稳定 ID、类型、标题、桌面布局、可选移动布局、Dataset 引用、字段 encoding 和展示属性。
- `filters`：类型化筛选器、默认值和显式目标映射。
- `interactions`：源组件事件、输出字段与目标组件参数/筛选映射。
- `publishScope`：all 或结构化主体集合；发布时同步规范化到版本主体表。

统计图首批支持一个或多个维度、一个或多个聚合指标以及平台限定的图形表现；明细表支持列、受控排序、分页大小和显示格式。字段引用使用表单发布快照中的不可变 `fieldId`/稳定字段编码，不以 label 为身份。协议不得包含函数、DOM、VChart/VTable 实例、原始 SQL 或服务端无法重建的对象。

现有 dashboard 包中的通用布局能力下沉为不含业务数据语义的基础层；业务仪表盘编辑器在其上组合 Dataset、字段绑定和组件描述符。工作台继续只消费基础布局层，从包边界上阻止协议串用。

### 6. Query DSL 由纯内核校验和服务端适配执行

前端 `@evolyn.do/query` 保持用户配置的规范来源；后端新增镜像的 `internal/engine/query`，包含 AST、类型/操作符矩阵、复杂度预算、规范化器和逻辑执行计划，禁止依赖存储实现。前后端以共享测试向量校验相同文档得出相同问题路径和规范化结果。

dashboard 平台适配层把执行计划交给 form 域提供的窄端口。该端口复用表单发布 schema、物理字段映射、参数化 SQL、RLS/租户会话、字段权限和记录行级范围，避免 dashboard 域直接拼接表名、列名或绕过现有数据权限。现有表单记录列表编译器不支持 groupBy/aggregates，实施时把可共享 AST/谓词和字段能力提取到纯内核，再为仪表盘增加受控聚合计划；不以复制 SQL 编译逻辑作为长期方案。

每次查询按当前成员重新判定：仪表盘发布范围、仪表盘查看权限、表单入口权限、字段权限和行级数据范围。预览也使用设计者自己的有效数据范围，不提供“以管理员视角模拟全部数据”。高精度 decimal/money/percent 均以 canonical decimal string 出网；明细表在用户排序后追加记录 ID 稳定尾排序。

初始资源上限由服务端配置且协议校验不可绕过：单仪表盘最多 50 个 Dataset、100 个组件，查询条件深度与现有表单 DSL 保持一致，明细页大小默认 20、最大 100，聚合结果最大 5,000 组，单组件查询默认 10 秒。超限在预检或执行期以稳定错误结束，不返回未经完整权限处理的部分结果。

### 7. 前端使用独立工作区和通用应用资产路由

主应用新增 dashboard 设计工作区，建议路由：

- `/app/:appCode/dashboard/:dashboardCode/design`
- `/app/:appCode/dashboard/:dashboardCode/preview`
- `/app/:appCode/dashboard/:dashboardCode/versions`
- `/app/:appCode/dashboard/:dashboardCode/publish`

设计页 shell 负责详情、标题、标签页、保存状态和离开拦截；`@evolyn.do/dashboard` 负责纯编辑器、画布、属性面板、数据配置与渲染。工作区状态通过组合式函数管理 `serverSnapshot`、`editingDocument`、revision、dirty、saving 和 issues，服务端保存成功后才替换 snapshot。409 时保留本地文档并提供重新加载服务端版本；本期不做自动三方合并。

应用运行时采用通用资产路由 `/app/:appCode/assets/:assetType/:assetCode`，根路由 `/app/:appCode` 仍用于默认选中。菜单 target 投影统一为 `assetType + assetCode`，表单和仪表盘由运行时注册表选择 loader/renderer；旧的表单专用可选参数路由在本次主应用调用方迁移完成后删除，避免继续扩大歧义。设计路由不嵌入应用运行时 shell，以隔离编辑状态和普通成员访问语义。

Vue 实现统一使用 Vue 3 Composition API、`<script setup lang="ts">`、类型化 props/emits 和可测试的 composable。路由离开使用组件级 guard，浏览器关闭使用仅在 dirty 时注册的 `beforeunload`。预览复用运行时 renderer；dirty 状态先保存，成功后才打开包含草稿 revision 的预览 URL。

### 8. 发布版本决定线上语义，当前权限决定即时可访问性

发布流程对当前保存草稿做完整终审，在一个事务内写版本、版本主体和当前版本指针。草稿后续修改不影响线上版本。发布范围主体定义被冻结到该版本，但 member/department/role/group 的有效关系在菜单、bootstrap 和每次查询时动态解析，因此成员离组或部门关系变化无需重新发布即可收口。

应用菜单读取侧通过 `DashboardDirectory` 批量投影：有管理权限者可见未发布仪表盘并获得编辑动作；普通成员只有在存在当前版本、命中发布范围且拥有查看权限时看到节点。运行时对不存在、跨租户、已删除、未发布和不在范围统一返回 `DASHBOARD_UNAVAILABLE`，不泄露资产状态。查询仍会进一步与数据源权限取交集。

单组件查询独立管理 loading/error/retry，不因一个组件失败卸载整个页面。bootstrap、组件查询和发布记录 trace ID、租户/应用/仪表盘/版本维度、耗时、结果规模和稳定失败分类；日志不得记录筛选敏感值或完整业务行。查看事件按成员、仪表盘版本和短时间窗口去重，避免刷新造成审计/指标放大。

### 9. 高级交互保持显式、可验证和可冻结

全局筛选只向显式声明的 Dataset/组件目标传值，目标字段必须与筛选值类型兼容。组件联动以受控事件类型和目标映射表示；保存与发布时构建有向图并拒绝环、未知目标和类型不兼容。运行时只在交互上下文中合并参数，不改写发布文档。

桌面和移动端布局分别保存。没有移动布局时，renderer 按桌面位置的 `y → x → widgetId` 稳定排序生成单列只读回退；回退不写回草稿。应用内复制在单个事务中复制当前草稿和展示信息，重新生成 code、组件/Dataset 是否需要重编号由规范化器统一处理，并创建未发布菜单节点；不复制发布历史、查看事件或个人收藏。

### 10. 权限资源和动作一次收敛

新增或落地 `dashboards` 管理资源以及 `dashboard-actions` 动作资源。至少区分 create、get、update、delete、design、preview、publish、copy、view；字段目录和预览查询要求 design/preview，运行时要求 view 且命中发布范围。租户管理员获得管理基线，普通成员只获得运行入口基线，最终是否可见由发布范围与数据源权限继续裁剪。

菜单动作注册表中 dashboard 的 `Landed` 只在对应端点和前端行为完成且测试通过后切换为 true，不允许先暴露无实现按钮。权限拒绝发生在业务服务之前；跨租户资源查询使用带 tenant 条件的仓储并统一按不存在处理。

## Risks / Trade-offs

- **[聚合查询扩大数据库压力]** → 使用字段能力白名单、参数化查询、复杂度预算、组数/页大小/超时限制和按租户指标；本期不引入缓存，以先保证权限正确性和结果一致性。
- **[表单查询逻辑提取影响既有列表]** → 先以共享测试锁定现有 filter/sort 行为，再提取纯 AST 和谓词；表单列表继续运行原有回归套件，聚合走新增计划分支。
- **[JSONB 文档随能力增多而复杂]** → 协议强版本、保存与发布双重校验、路径级 issues、规范化输出和不可变版本；后续升级必须提供显式 migration，而不是在 renderer 中猜测旧结构。
- **[动态组织关系与版本快照语义不同]** → 冻结“声明的主体”，动态计算“主体当前成员”；这样能立即撤权，但历史版本回看不会重现当时成员集合。审计记录发布主体定义和访问判定版本以便追踪。
- **[通用资产路由会影响现有表单入口]** → 在同一阶段迁移菜单 target、选中态、刷新和前进后退测试；不保留两套长期路由状态，必要时仅提供短期显式 redirect 并在阶段结束前移除。
- **[并发保存无法自动合并]** → 以不丢数据为第一目标，409 保留本地编辑并允许复制内容/重新加载；自动合并留到有稳定组件级操作日志后评估。
- **[单次变更范围较大]** → 严格按五阶段提交，每阶段包含迁移、API、前端和测试的完整纵切；后一阶段不得成为前一阶段上线的必需条件。

## Migration Plan

1. 新增 dashboard 资产、版本、发布主体和创建幂等表，注册权限资源与配额键；先保持菜单 dashboard 动作未落地。
2. 上线资产 API 与菜单窄端口，完成预创建、详情、草稿、改名、删除和权限测试；确认回滚不会留下菜单孤儿后开放创建动作。
3. 上线独立设计工作区、通用资产路由和空画布保存链路；迁移表单运行时到通用资产路由并完成刷新/返回回归。
4. 提取查询纯内核并上线字段目录、Dataset、统计图、明细表和预览查询；先以受控功能开关限制租户，再观察慢查询和拒绝原因。
5. 上线发布版本、菜单发布状态投影、runtime bootstrap 与运行查询；当前无发布版本的菜单仅管理员可见。
6. 上线筛选、联动、移动布局、复制和发布范围；完成协议终审、版本冻结和动态撤权测试后开放对应动作。
7. 更新文档、Swagger 和前端模块现状，移除仪表盘占位 UI 与旧表单专用运行路由；确认 `/workbench` 回归无变化。

回滚时先关闭新建、发布和运行时功能开关，保留数据表和不可变版本避免数据丢失；应用菜单读取侧可在 DashboardDirectory 不可用时过滤 dashboard 节点，不回退为工作台协议。代码回滚不删除 migration 或业务数据，恢复后可继续读取原 revision 和版本。

## Verification Strategy

- 纯内核：前后端共享协议向量覆盖规范化、未知版本、重复 ID、字段能力、查询复杂度、交互环和移动布局回退。
- 后端单元/集成：事务回滚、幂等重试、配额并发、租户隔离、乐观锁、发布不可变、动态主体撤权、查询字段/行级裁剪和高精度结果。
- 菜单契约：创建/改名/移动/删除只推进一次 revision；未发布与发布范围在管理员/普通成员下投影正确；个人收藏失效正确。
- 前端：工作区状态机、409 保留本地文档、路径级 issue 定位、路由/浏览器离开保护、预览先保存、运行时单组件失败隔离。
- E2E：从“新建仪表盘”预创建进入空画布，配置 Dataset 和统计图/明细表，预览、发布、应用运行、刷新恢复、权限收紧和删除的完整主路径。
- 回归：表单创建/设计/运行、表单记录 Query DSL、应用菜单与 `/workbench` 保存和渲染行为不变。

## Open Questions

- `dashboards` 三档默认配额 seed 在实施 migration 前需要产品侧确认；若未确认，采用本设计建议的 3/20/无限并在权益目录文档中明确。
- 初始查看事件去重窗口和查询并发阈值应配置化；实现默认值分别采用 5 分钟和每成员每仪表盘 4 个并发查询，压测后只调整配置，不改变协议。
