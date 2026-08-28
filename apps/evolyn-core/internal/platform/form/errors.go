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
)
