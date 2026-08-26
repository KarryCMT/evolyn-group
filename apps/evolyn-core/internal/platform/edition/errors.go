// Package edition 版本信息域（一期）：套餐目录/套餐版本快照/租户订阅/
// 特批权益覆盖。稳定业务错误码集中定义于本包（ADR-008），前端按 errCode
// 分支（packages/utils/src/request/errorCodes.ts 对齐维护），内部细节经
// httpx.Wrap 只入日志。二期交易类错误码（ORDER_* / PAYMENT_*）随商品域补充
package edition

import (
	"net/http"

	"evolyn/internal/platform/httpx"
)

var (
	// ErrTenantNotFound 目标租户不存在（平台授予/读取路径按 ID 定位租户）
	ErrTenantNotFound = httpx.NewBiz("EDITION_TENANT_NOT_FOUND", "租户不存在或已注销", http.StatusNotFound)

	// ErrPlanVersionNotFound 套餐版本不存在（planVersionId 无效）
	ErrPlanVersionNotFound = httpx.NewBiz("EDITION_PLAN_VERSION_NOT_FOUND", "套餐版本不存在", http.StatusNotFound)

	// ErrPlanVersionNotGrantable 套餐版本不可授予（未发布/已下架/非基础套餐）
	ErrPlanVersionNotGrantable = httpx.NewBiz("EDITION_PLAN_VERSION_NOT_GRANTABLE", "该套餐版本不可授予", http.StatusConflict)

	// ErrGrantInvalid 授予参数不合法（授予方式、起止时间、试用缺到期日等）
	ErrGrantInvalid = httpx.NewBiz("EDITION_GRANT_INVALID", "订阅授予参数不合法", http.StatusBadRequest)

	// ErrOverrideInvalid 特批覆盖不合法（未知资源键、非法数值）
	ErrOverrideInvalid = httpx.NewBiz("EDITION_OVERRIDE_INVALID", "特批权益配置无效", http.StatusBadRequest)

	// ErrStorageLimitInvalid 存储上限违反整 GiB 约束（一期保证新旧双键精确换算）
	ErrStorageLimitInvalid = httpx.NewBiz("EDITION_STORAGE_LIMIT_INVALID", "存储上限只支持不限量、禁用或整 GiB 的字节数", http.StatusBadRequest)

	// ErrSubscriptionNotFound 当前无可操作的订阅（取消路径无活动/待审订阅）
	ErrSubscriptionNotFound = httpx.NewBiz("EDITION_SUBSCRIPTION_NOT_FOUND", "当前没有可操作的订阅", http.StatusNotFound)
)
