// Package form 表单资产域（ADR-010 / docs/低代码平台/表单设计器/表单资产域后端契约.md）：
// 表单资产与草稿、不可变发布版本、记录提交。目标保存协议 content.items 是唯一
// 事实结构，草稿/发布/提交均按字段字典严格校验后落库。稳定业务错误码集中定义于
// 本包（ADR-008），调用方按 errCode 分支；内部细节经 httpx.Wrap 只入日志。
package form

import (
	"net/http"

	"evolyn/internal/platform/httpx"
)

var (
	// ErrFormNotFound 表单不存在或无权访问（租户过滤后的 NotFound 统一口径）
	ErrFormNotFound = httpx.NewBiz("FORM_NOT_FOUND", "表单不存在或无权访问", http.StatusNotFound)

	// ErrFormNameInvalid 表单名称不符合要求（trim 后 1–128 字符）
	ErrFormNameInvalid = httpx.NewBiz("FORM_NAME_INVALID", "表单名称不符合要求", http.StatusBadRequest)

	// ErrFormTypeInvalid 表单类型不是 standard/workflow 稳定枚举
	ErrFormTypeInvalid = httpx.NewBiz("FORM_TYPE_INVALID", "表单类型不符合要求", http.StatusBadRequest)

	// ErrFormTypeUnchanged 切换类型时目标类型与当前类型相同（ADR-011）：
	// 无变化不落库，客户端据此感知误操作
	ErrFormTypeUnchanged = httpx.NewBiz("FORM_TYPE_UNCHANGED", "表单类型未发生变化", http.StatusBadRequest)

	// ErrFormIconInvalid 图标/颜色稳定键不符合要求（空串表示清空，其余最长 32 字符）
	ErrFormIconInvalid = httpx.NewBiz("FORM_ICON_INVALID", "表单图标或颜色配置无效", http.StatusBadRequest)

	// ErrFormAppInvalid 应用无效：不存在/跨租户/已归档，或表单不归属该应用
	ErrFormAppInvalid = httpx.NewBiz("FORM_APP_INVALID", "应用不存在或不可用", http.StatusBadRequest)

	// ErrSchemaInvalid 草稿协议校验失败；data 携带 issues:[{path,message}]（JSON Path 级）
	ErrSchemaInvalid = httpx.NewBiz("FORM_SCHEMA_INVALID", "表单内容不符合保存协议", http.StatusBadRequest)

	// ErrRevisionConflict 草稿修订号过期（他人已保存），客户端刷新后重试
	ErrRevisionConflict = httpx.NewBiz("FORM_REVISION_CONFLICT", "表单已被他人更新，请刷新后重试", http.StatusConflict)

	// ErrPublishUnsupportedField 发布命中能力白名单外控件；data 携带 issues:[{path,message}]
	ErrPublishUnsupportedField = httpx.NewBiz("FORM_PUBLISH_UNSUPPORTED_FIELD", "存在暂不能发布的字段，请先移除或等待能力开放", http.StatusBadRequest)

	// ErrNotPublished 表单尚未发布（运行时 bootstrap）
	ErrNotPublished = httpx.NewBiz("FORM_NOT_PUBLISHED", "表单尚未发布", http.StatusNotFound)

	// ErrVersionConflict 提交的发布版本/修订口令与快照不符
	ErrVersionConflict = httpx.NewBiz("FORM_VERSION_CONFLICT", "表单已发布新版本，请刷新后重试", http.StatusConflict)

	// ErrRecordInvalid 提交值校验失败；data 携带 fieldErrors{widgetName:[msg]}
	ErrRecordInvalid = httpx.NewBiz("FORM_RECORD_INVALID", "提交内容未通过校验，请修正后重试", http.StatusBadRequest)

	// ErrRecordValidationFailed 为阻断型提交规则终审失败；data 仅携带安全的
	// validatorErrors:[{index,remind,fields}]，绝不回显公式原始输入值。
	ErrRecordValidationFailed = httpx.NewBiz("FORM_RECORD_VALIDATION_FAILED", "提交内容未通过业务校验，请修正后重试", http.StatusBadRequest)

	// ErrRecordQueryInvalid 记录列表 Query DSL 不符合已发布字段快照或支持的
	// 查询语义；不回显 SQL/JSONB 路径等内部细节。
	ErrRecordQueryInvalid = httpx.NewBiz("FORM_RECORD_QUERY_INVALID", "数据筛选条件不符合要求", http.StatusBadRequest)

	// ErrRecordDeleteInvalid 表示批量删除载荷为空、重复或超出单次上限。
	ErrRecordDeleteInvalid = httpx.NewBiz("FORM_RECORD_DELETE_INVALID", "请选择有效的数据后重试", http.StatusBadRequest)

	// ErrRecordWorkflowActive 流程记录保留审批历史与关联任务，不能绕过流程运行
	// 时直接硬删；须先在流程中心结束实例后再进行数据清理。
	ErrRecordWorkflowActive = httpx.NewBiz("FORM_RECORD_WORKFLOW_ACTIVE", "流程中的数据不能直接删除，请先结束流程", http.StatusConflict)

	// ErrForbidden 表单域操作越权（与鉴权中间件共用 FORBIDDEN 稳定码）
	ErrForbidden = httpx.NewBiz(httpx.CodeForbidden, "没有执行该操作的权限", http.StatusForbidden)

	// ---- 表单资产权限组（表单权限 P1，设计 §7.3 错误码表） ----

	// ErrPermissionDenied 权限组判定：无对应操作/字段/范围权限（执行点判定，
	// 含菜单/运行时入口收口与提交流程特有动作叠加判定）
	ErrPermissionDenied = httpx.NewBiz("FORM_PERMISSION_DENIED", "没有该表单数据的访问权限", http.StatusForbidden)

	// ErrPermissionGroupNotFound 权限组不存在或已删除
	ErrPermissionGroupNotFound = httpx.NewBiz("FORM_PERMISSION_GROUP_NOT_FOUND", "权限组不存在或已删除", http.StatusNotFound)

	// ErrPermissionNameInvalid 权限组名称不符合要求（trim 后 1–64 字符）
	ErrPermissionNameInvalid = httpx.NewBiz("FORM_PERMISSION_NAME_INVALID", "权限组名称不符合要求", http.StatusBadRequest)

	// ErrPermissionOperationInvalid 操作键不在该表单类型合法集（§3 字典分派）
	ErrPermissionOperationInvalid = httpx.NewBiz("FORM_PERMISSION_OPERATION_INVALID", "操作权限配置不符合要求", http.StatusBadRequest)

	// ErrPermissionFieldInvalid 字段键不在清单 / 必填字段违规 / visible-editable
	// 矛盾（§4 配置期校验，含必填协调两规则）
	ErrPermissionFieldInvalid = httpx.NewBiz("FORM_PERMISSION_FIELD_INVALID", "字段权限配置不符合要求", http.StatusBadRequest)

	// ErrPermissionDataScopeInvalid 数据范围配置非法：match 非法 / operator
	// 不适用字段类型 / 比较值形状不符（§5 配置期类型分派）
	ErrPermissionDataScopeInvalid = httpx.NewBiz("FORM_PERMISSION_DATA_SCOPE_INVALID", "数据权限配置不符合要求", http.StatusBadRequest)

	// ErrPermissionSubjectInvalid 主体不存在 / 非同租户 / 超上限（单组 200）
	ErrPermissionSubjectInvalid = httpx.NewBiz("FORM_PERMISSION_SUBJECT_INVALID", "权限组成员配置不符合要求", http.StatusBadRequest)

	// ErrPermissionRevisionConflict 整组乐观锁口令过期（他人已保存），客户端刷新后重试
	ErrPermissionRevisionConflict = httpx.NewBiz("FORM_PERMISSION_REVISION_CONFLICT", "权限组已被他人更新，请刷新后重试", http.StatusConflict)

	// ErrPermissionLimitExceeded 单表单权限组数量超上限（50）
	ErrPermissionLimitExceeded = httpx.NewBiz("FORM_PERMISSION_LIMIT_EXCEEDED", "权限组数量已达上限", http.StatusBadRequest)

	// ErrPermissionBlockedTypeSwitch 类型切换被权限组阻塞（§3.3）：standard
	// 表单存在任一权限组（含禁用组）的 operations 含 workflow_* 键，须先清理
	ErrPermissionBlockedTypeSwitch = httpx.NewBiz("FORM_PERMISSION_BLOCKED_TYPE_SWITCH", "存在包含流程操作的权限组，无法切换为普通表单", http.StatusConflict)

	// ErrPermissionBlockedPublish 发布被启用权限组的 data_scope 字段引用阻塞
	//（§5.2 字段生命周期）：新版本删除或变更（类型/形状）的字段被引用，发布
	// 拒绝并列出冲突字段，由管理员先调整权限组再发布
	ErrPermissionBlockedPublish = httpx.NewBiz("FORM_PERMISSION_BLOCKED_PUBLISH", "发布被权限组的数据条件阻塞，请先调整引用变更字段的权限组", http.StatusConflict)

	// ---- 物理表存储（docs/低代码平台/表单设计器/物理表存储后端实施方案.md §14） ----

	// ErrFieldIdentityFrozen v8 契约冻结：已发布字段的 fieldId/widgetName 不可
	// 修改（字段「改名」只允许改 label）；data 携带 fields:[widgetName]
	ErrFieldIdentityFrozen = httpx.NewBiz("FORM_FIELD_IDENTITY_FROZEN", "已发布字段的标识不可修改，请仅调整字段名称（label）或新建字段", http.StatusConflict)

	// ErrStorageBusy 同一表单存在待执行/执行中的物理模型 Job，新发布请求被拒
	ErrStorageBusy = httpx.NewBiz("FORM_STORAGE_BUSY", "表单结构变更正在执行，请稍后重试", http.StatusConflict)

	// ErrStorageUnsupportedField 发布快照命中尚不具备物理模型的控件（多选/
	// 附件等）；data 携带 issues:[{path,message}]
	ErrStorageUnsupportedField = httpx.NewBiz("FORM_STORAGE_UNSUPPORTED_FIELD", "存在暂不支持物理存储的字段，请先移除或等待能力开放", http.StatusBadRequest)

	// ErrStorageModelInvalid 物理模型构建/校验失败（快照与冻结映射不一致等
	// 服务端内部一致性问题的安全出网口径）
	ErrStorageModelInvalid = httpx.NewBiz("FORM_STORAGE_MODEL_INVALID", "表单物理存储模型无效，请重新发布", http.StatusBadRequest)

	// ErrStorageDDLFailed DDL Job 终态失败；data 携带受控失败信息（jobId/
	// errorCode），内部细节只入日志
	ErrStorageDDLFailed = httpx.NewBiz("FORM_STORAGE_DDL_FAILED", "表单结构变更执行失败，请重试或联系管理员", http.StatusInternalServerError)

	// ErrStorageNotReady 存储未就绪（DDL 未完成/终态失败），运行时提交被拒
	ErrStorageNotReady = httpx.NewBiz("FORM_STORAGE_NOT_READY", "表单存储尚未就绪，请稍后重试", http.StatusConflict)

	// ErrStorageTypeChangeUnsupported 字段类型变更默认拒绝：新建字段并弃用
	// 旧字段；data 携带 fields:[widgetName]
	ErrStorageTypeChangeUnsupported = httpx.NewBiz("FORM_STORAGE_TYPE_CHANGE_UNSUPPORTED", "已发布字段的类型不可修改，请新建字段并弃用原字段", http.StatusConflict)

	// ErrWorkflowProjectionInvalid 流程投影更新入参非法（状态枚举外值等）
	// ErrPublishedMoneyDefinitionLocked 已发布金额字段的币种与有效精度属于数据
	// 语义的一部分，不能静默改写历史记录的解释方式；data 携带 fields:[widgetName]。
	ErrPublishedMoneyDefinitionLocked = httpx.NewBiz("FORM_PUBLISHED_MONEY_DEFINITION_LOCKED", "已发布金额字段的币种或精度不可修改，请新建字段并弃用原字段", http.StatusConflict)

	ErrWorkflowProjectionInvalid = httpx.NewBiz("FORM_WORKFLOW_PROJECTION_INVALID", "流程状态投影无效", http.StatusInternalServerError)

	// ---- 前端事件（协议 v10） ----
	ErrFrontendEventNotFound          = httpx.NewBiz("FORM_EVENT_NOT_FOUND", "前端事件不存在", http.StatusNotFound)
	ErrFrontendEventInvalid           = httpx.NewBiz("FORM_EVENT_INVALID", "前端事件配置无效", http.StatusBadRequest)
	ErrFrontendEventDisabled          = httpx.NewBiz("FORM_EVENT_DISABLED", "前端事件已停用", http.StatusConflict)
	ErrFrontendEventTriggerMismatch   = httpx.NewBiz("FORM_EVENT_TRIGGER_MISMATCH", "触发字段与事件配置不一致", http.StatusBadRequest)
	ErrFrontendEventTemplateInvalid   = httpx.NewBiz("FORM_EVENT_TEMPLATE_INVALID", "前端事件模板无效", http.StatusBadRequest)
	ErrFrontendEventRequestBlocked    = httpx.NewBiz("FORM_EVENT_REQUEST_BLOCKED", "请求地址被安全策略拦截", http.StatusForbidden)
	ErrFrontendEventRequestFailed     = httpx.NewBiz("FORM_EVENT_REQUEST_FAILED", "前端事件请求失败", http.StatusBadGateway)
	ErrFrontendEventResponseInvalid   = httpx.NewBiz("FORM_EVENT_RESPONSE_INVALID", "前端事件响应格式无效", http.StatusBadGateway)
	ErrFrontendEventMappingFailed     = httpx.NewBiz("FORM_EVENT_MAPPING_FAILED", "返回值映射失败", http.StatusBadRequest)
	ErrFrontendEventTargetNotWritable = httpx.NewBiz("FORM_EVENT_TARGET_NOT_WRITABLE", "目标字段不可写", http.StatusForbidden)
	ErrFrontendEventCycleDetected     = httpx.NewBiz("FORM_EVENT_CYCLE_DETECTED", "前端事件调用链存在循环", http.StatusConflict)

	// ErrStorageJobNotFound DDL 发布 Job 不存在
	ErrStorageJobNotFound = httpx.NewBiz("FORM_STORAGE_JOB_NOT_FOUND", "结构变更任务不存在", http.StatusNotFound)
)
