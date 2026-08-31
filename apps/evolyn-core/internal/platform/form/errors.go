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
)
