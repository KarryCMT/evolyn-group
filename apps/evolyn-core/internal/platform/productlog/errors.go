// Package productlog 产品日志域（000064，docs/低代码平台/产品日志/）：
// 管理后台「产品日志」的只读查询与导出编排——读取 tn_audit_logs 中应用内
// 操作（产品分类）的受控投影。与企业日志共用审计事实表和写入链路（audit
// 域），但查询范围按 category_code 互斥。稳定业务错误码集中定义于本包
// （ADR-008），前端按 errCode 分支（packages/utils/src/request/errorCodes.ts
// 对齐维护），内部细节经 httpx.Wrap 只入日志
package productlog

import (
	"net/http"

	"evolyn/internal/platform/httpx"
)

var (
	// ErrDateInvalid 日期入参格式非法（应为 yyyy-MM-dd 东八区自然日）
	ErrDateInvalid = httpx.NewBiz("PRODUCT_LOG_DATE_INVALID", "日期格式非法，应为 yyyy-MM-dd", http.StatusBadRequest)

	// ErrTimeRangeInvalid 开始时间晚于结束时间
	ErrTimeRangeInvalid = httpx.NewBiz("PRODUCT_LOG_TIME_RANGE_INVALID", "开始时间不能晚于结束时间", http.StatusBadRequest)

	// ErrMemberInvalid 筛选成员不存在、已删除或不属于当前租户
	ErrMemberInvalid = httpx.NewBiz("PRODUCT_LOG_MEMBER_INVALID", "所选成员无效", http.StatusBadRequest)

	// ErrApplicationInvalid 筛选应用不存在、已删除或不属于当前租户
	ErrApplicationInvalid = httpx.NewBiz("PRODUCT_LOG_APPLICATION_INVALID", "所选应用无效", http.StatusBadRequest)

	// ErrCategoryUnknown 未登记的产品日志范围码
	ErrCategoryUnknown = httpx.NewBiz("PRODUCT_LOG_CATEGORY_UNKNOWN", "未知的日志范围", http.StatusBadRequest)

	// ErrEventUnknown 未登记的操作类型事件码
	ErrEventUnknown = httpx.NewBiz("PRODUCT_LOG_EVENT_UNKNOWN", "未知的操作类型", http.StatusBadRequest)

	// ErrExportNotFound 导出任务不存在或不属于当前租户
	ErrExportNotFound = httpx.NewBiz("PRODUCT_LOG_EXPORT_NOT_FOUND", "导出任务不存在", http.StatusNotFound)

	// ErrExportTooLarge 单次导出超过平台上限（留存策略接入前的固定上限，
	// 提示缩小筛选范围）
	ErrExportTooLarge = httpx.NewBiz("PRODUCT_LOG_EXPORT_TOO_LARGE", "导出数据量超过上限，请缩小筛选范围后重试", http.StatusBadRequest)

	// ErrExportNotReady 任务尚未生成完成（同步生成下一般为瞬时态）
	ErrExportNotReady = httpx.NewBiz("PRODUCT_LOG_EXPORT_NOT_READY", "导出文件尚未生成完成", http.StatusConflict)

	// ErrExportExpired 导出文件已过有效期，需重新发起导出
	ErrExportExpired = httpx.NewBiz("PRODUCT_LOG_EXPORT_EXPIRED", "导出文件已过期，请重新导出", http.StatusGone)

	// ErrExportForbidden 具备查看权限但缺少导出权限（下载路径复核用）
	ErrExportForbidden = httpx.NewBiz("PRODUCT_LOG_EXPORT_FORBIDDEN", "无日志导出权限", http.StatusForbidden)
)
