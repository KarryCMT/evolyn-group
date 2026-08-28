// Package notification 消息中心域（docs/低代码平台/消息中心/消息中心现状分析
// 与后端开发设计.md）：租户×成员的站内信收件箱、租户通知偏好与自定义提醒对象。
// 业务域经事务 Outbox 发布结构化事件，Dispatcher Worker 渲染纯文本快照并扇出。
// 稳定业务错误码集中定义于本包（ADR-008），调用方按 errCode 分支；内部细节经
// httpx.Wrap 只入日志。
package notification

import (
	"net/http"

	"evolyn/internal/platform/httpx"
)

var (
	// ErrCategoryUnknown 分类码未在事件目录登记（八个稳定分类之外）
	ErrCategoryUnknown = httpx.NewBiz("NOTIFICATION_CATEGORY_UNKNOWN", "消息分类不存在", http.StatusBadRequest)

	// ErrEventUnknown 事件码未登记或不属于当前分类
	ErrEventUnknown = httpx.NewBiz("NOTIFICATION_EVENT_UNKNOWN", "事件类型不存在或不属于当前分类", http.StatusBadRequest)

	// ErrCursorInvalid 游标无法解析或不透明载荷不合法
	ErrCursorInvalid = httpx.NewBiz("NOTIFICATION_CURSOR_INVALID", "分页游标无效，请刷新后重试", http.StatusBadRequest)

	// ErrNotFound 当前成员收件箱中不存在该消息（不泄露他人消息的存在性）
	ErrNotFound = httpx.NewBiz("NOTIFICATION_NOT_FOUND", "消息不存在", http.StatusNotFound)

	// ErrSettingsConflict 租户通知设置 revision 冲突（已被其他管理员修改）
	ErrSettingsConflict = httpx.NewBiz("NOTIFICATION_SETTINGS_CONFLICT", "通知设置已被他人修改，请刷新后重试", http.StatusConflict)

	// ErrChannelNotSupported 事件不支持请求渠道
	ErrChannelNotSupported = httpx.NewBiz("NOTIFICATION_CHANNEL_NOT_SUPPORTED", "该事件不支持此提醒渠道", http.StatusBadRequest)

	// ErrChannelRequired 尝试关闭必选渠道（站内信）
	ErrChannelRequired = httpx.NewBiz("NOTIFICATION_CHANNEL_REQUIRED", "站内信为必选渠道，不能关闭", http.StatusBadRequest)

	// ErrChannelUnavailable Provider/计费能力尚不可用，不允许新开启（P3 前邮件/短信恒不可用）
	ErrChannelUnavailable = httpx.NewBiz("NOTIFICATION_CHANNEL_UNAVAILABLE", "该提醒渠道暂未开放，无法开启", http.StatusConflict)

	// ErrRecipientInvalid 姓名/手机/邮箱或接收规则不合法
	ErrRecipientInvalid = httpx.NewBiz("NOTIFICATION_RECIPIENT_INVALID", "提醒对象信息不完整或不合法", http.StatusBadRequest)

	// ErrRecipientNotFound 联系人不存在、已删除或不属于当前租户
	ErrRecipientNotFound = httpx.NewBiz("NOTIFICATION_RECIPIENT_NOT_FOUND", "提醒对象不存在", http.StatusNotFound)

	// ErrRecipientInUse 联系人仍被事件偏好引用；data 携带脱敏 usedByEventCodes
	ErrRecipientInUse = httpx.NewBiz("NOTIFICATION_RECIPIENT_IN_USE", "提醒对象仍被通知偏好引用，请先移除引用", http.StatusConflict)

	// ErrRecipientLimitExceeded 自定义提醒对象达到租户上限（服务端配置控制）
	ErrRecipientLimitExceeded = httpx.NewBiz("NOTIFICATION_RECIPIENT_LIMIT_EXCEEDED", "提醒对象数量已达上限", http.StatusForbidden)
)
