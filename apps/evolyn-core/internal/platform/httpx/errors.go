package httpx

import (
	"errors"
	"fmt"
	"net/http"
)

// 通用业务错误码（ADR-008 首批稳定码表·通用段）：
// 域内专属码在各域包定义（iam/tenant/auth），此处只放跨域通用码
const (
	CodeNotFound       = "NOT_FOUND"       // 404 记录不存在
	CodeValidation     = "VALIDATION"      // 400 参数/校验错误
	CodeUnauthorized   = "UNAUTHORIZED"    // 401 未认证/凭证失效
	CodeForbidden      = "FORBIDDEN"       // 403 无权限
	CodeRateLimited    = "RATE_LIMITED"    // 429 频控
	CodeInternalServer = "INTERNAL_SERVER" // 500 服务端内部错误
)

// BizError 业务错误：稳定错误码 + 对外安全文案 + 建议 HTTP 状态码（ADR-008）。
// 域服务用 NewBiz 构造常量、Wrap 附加原始错误；ResponseFailed 统一映射出网——
// 原始错误只进日志，客户端只见 Code/Msg。
// errors.Is 兼容：按 Code 判定（含 fmt.Errorf %w 包装链）。
type BizError struct {
	Code string // 稳定业务码，如 DUPLICATE_PHONE，前端按此分支
	Msg  string // 对外文案，可直接展示（禁止携带内部数据）
	HTTP int    // 建议 HTTP 状态码（信封 code 同步）
	err  error  // 被包装的原始错误，仅用于日志与 Unwrap
}

// NewBiz 构造业务错误常量（各域包顶部定义）
func NewBiz(code, msg string, httpStatus int) *BizError {
	return &BizError{Code: code, Msg: msg, HTTP: httpStatus}
}

// Wrap 为业务错误附加原始错误（保留码/文案/状态码），细节不外泄只入日志
func Wrap(biz *BizError, err error) *BizError {
	if biz == nil {
		return nil
	}
	return &BizError{Code: biz.Code, Msg: biz.Msg, HTTP: biz.HTTP, err: err}
}

func (e *BizError) Error() string {
	if e.err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Msg, e.err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Msg)
}

// Unwrap 暴露被包装的原始错误（errors.As/Is 链）
func (e *BizError) Unwrap() error {
	return e.err
}

// Is 按 Code 判定同一性：errors.Is(err, ErrXxx) 在 %w 包装链上依然成立
func (e *BizError) Is(target error) bool {
	var other *BizError
	if errors.As(target, &other) {
		return e.Code == other.Code
	}
	return false
}

// 便捷构造（按通用语义）
func ErrNotFound(msg string) *BizError {
	if msg == "" {
		msg = "记录不存在"
	}
	return NewBiz(CodeNotFound, msg, http.StatusNotFound)
}
