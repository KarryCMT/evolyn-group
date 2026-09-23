// Package label 是标签模板、发布版本与正式渲染的平台适配层。
package label

import (
	"net/http"

	"evolyn/internal/platform/httpx"
)

var (
	ErrTemplateNotFound     = httpx.NewBiz("LABEL_TEMPLATE_NOT_FOUND", "标签模板不存在或无权访问", http.StatusNotFound)
	ErrTemplateNotPublished = httpx.NewBiz("LABEL_TEMPLATE_NOT_PUBLISHED", "标签模板尚未发布", http.StatusBadRequest)
	ErrSchemaInvalid        = httpx.NewBiz("LABEL_SCHEMA_INVALID", "标签模板不符合 LabelSchema 1.0 协议", http.StatusBadRequest)
	ErrFieldNotFound        = httpx.NewBiz("LABEL_FIELD_NOT_FOUND", "标签引用了不存在的表单字段", http.StatusBadRequest)
	ErrFieldNoPermission    = httpx.NewBiz("LABEL_FIELD_NO_PERMISSION", "没有标签字段的查看权限", http.StatusForbidden)
	ErrRecordNotFound       = httpx.NewBiz("LABEL_RECORD_NOT_FOUND", "表单记录不存在或无权访问", http.StatusNotFound)
	ErrRecordNoPermission   = httpx.NewBiz("LABEL_RECORD_NO_PERMISSION", "没有表单记录的查看权限", http.StatusForbidden)
	ErrExpression           = httpx.NewBiz("LABEL_EXPRESSION_ERROR", "标签表达式解析失败", http.StatusBadRequest)
	ErrQRGenerate           = httpx.NewBiz("LABEL_QR_GENERATE_ERROR", "二维码生成失败", http.StatusInternalServerError)
	ErrRender               = httpx.NewBiz("LABEL_RENDER_ERROR", "标签渲染失败", http.StatusInternalServerError)
	ErrRevisionConflict     = httpx.NewBiz("LABEL_REVISION_CONFLICT", "标签模板已被他人更新，请刷新后重试", http.StatusConflict)
	ErrRealPreviewRequired  = httpx.NewBiz("LABEL_REAL_PREVIEW_REQUIRED", "发布前必须使用真实表单记录完成预览", http.StatusBadRequest)
	ErrFormAlreadyBound     = httpx.NewBiz("LABEL_FORM_ALREADY_BOUND", "该表单已绑定二维码标签模板", http.StatusConflict)
	ErrCodeInvalid          = httpx.NewBiz("LABEL_TEMPLATE_CODE_INVALID", "无效的标签模板编码", http.StatusBadRequest)
	ErrNameInvalid          = httpx.NewBiz("LABEL_TEMPLATE_NAME_INVALID", "标签模板名称不符合要求", http.StatusBadRequest)
	ErrFormInvalid          = httpx.NewBiz("LABEL_FORM_INVALID", "绑定表单不存在或无权访问", http.StatusBadRequest)
	ErrFormatUnsupported    = httpx.NewBiz("LABEL_FORMAT_UNSUPPORTED", "暂不支持该标签输出格式", http.StatusBadRequest)
	ErrBatchInvalid         = httpx.NewBiz("LABEL_BATCH_INVALID", "批量标签请求不符合要求", http.StatusBadRequest)
	ErrTaskNotFound         = httpx.NewBiz("LABEL_TASK_NOT_FOUND", "标签渲染任务不存在或无权访问", http.StatusNotFound)
	ErrTaskQueueUnavailable = httpx.NewBiz("LABEL_TASK_QUEUE_UNAVAILABLE", "标签渲染队列暂不可用，请稍后重试", http.StatusServiceUnavailable)
	ErrTaskNotReady         = httpx.NewBiz("LABEL_TASK_NOT_READY", "标签渲染任务尚未生成可下载文件", http.StatusConflict)
	ErrStorageUnavailable   = httpx.NewBiz("LABEL_STORAGE_UNAVAILABLE", "标签文件存储暂不可用，请稍后重试", http.StatusServiceUnavailable)
	ErrQRTokenInvalid       = httpx.NewBiz("LABEL_QR_TOKEN_INVALID", "二维码已失效或无权访问", http.StatusNotFound)
	ErrForbidden            = httpx.NewBiz(httpx.CodeForbidden, "没有执行该操作的权限", http.StatusForbidden)
)
