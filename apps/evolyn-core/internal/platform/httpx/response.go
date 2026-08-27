package httpx

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"

	"evolyn/internal/platform/ginctx"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Cookie 名称（登录态）：token 为 JWT，loginUser 为前端展示用的用户信息
const (
	CookieTokenName = `token`
	CookieLoginUser = `loginUser`
)

// 未知服务端错误的对外兜底文案（ADR-008 脱敏原则：原始错误只进日志）
const internalServerMsg = "服务器开小差了，请稍后重试"

// Response 统一响应结构。errCode 为稳定业务码（ADR-008），
// 成功与未分类错误之外必有值；code 保持 HTTP 状态码（兼容保留）
type Response struct {
	Code    int         `json:"code"`
	ErrCode string      `json:"errCode,omitempty"`
	Msg     string      `json:"msg"`
	Data    interface{} `json:"data"`
}

func NewResponse(c *gin.Context, code int, data interface{}, msg string) {
	c.JSON(code, Response{
		Code: code,
		Msg:  msg,
		Data: data,
	})
}

func ResponseSuccess(c *gin.Context, data interface{}) {
	NewResponse(c, http.StatusOK, data, "success")
}

// ResponseFailed 统一错误出口（ADR-008 自动映射）：
//   - BizError：用其 HTTP/Code/Msg（调用方传入的 code 仅在 BizError.HTTP 为 0 时兜底）；
//   - gorm.ErrRecordNotFound / redis.Nil：404 + NOT_FOUND；
//   - 其他错误且状态码 >= 500：500 + INTERNAL_SERVER + 脱敏文案（原文只进日志）；
//   - 其他错误且状态码为 4xx：保留调用方判定与原文（多为参数/会话类校验文案），
//     errCode 按状态给通用码。
//
// 401 时顺带清理登录 Cookie；err 非空时记录请求上下文日志
func ResponseFailed(c *gin.Context, code int, err error) {
	if code == 0 {
		code = http.StatusInternalServerError
	}

	var errCode, msg string
	var biz *BizError
	var data any
	switch {
	case errors.As(err, &biz):
		if biz.HTTP != 0 {
			code = biz.HTTP
		}
		errCode, msg, data = biz.Code, biz.Msg, biz.Data
	case errors.Is(err, gorm.ErrRecordNotFound), errors.Is(err, redis.Nil):
		code, errCode, msg = http.StatusNotFound, CodeNotFound, "记录不存在"
	default:
		if code >= http.StatusInternalServerError {
			errCode, msg = CodeInternalServer, internalServerMsg
		} else {
			errCode = statusErrCode(code)
			if err != nil {
				msg = err.Error()
			}
		}
	}

	if code == http.StatusUnauthorized && c.Request != nil {
		if val, err := c.Cookie(CookieTokenName); err == nil && val != "" {
			c.SetCookie(CookieTokenName, "", -1, "/", "", true, true)
			c.SetCookie(CookieLoginUser, "", -1, "/", "", true, false)
		}
	}

	if err != nil {
		user := ginctx.GetUser(c)
		var name string
		if user != nil {
			name = user.Nickname
		}
		var url string
		if c.Request != nil {
			url = c.Request.URL.String()
		}
		logrus.Warnf("url: %s, user: %s, code: %s, error: %v", url, name, errCode, err)
	}
	c.JSON(code, Response{Code: code, ErrCode: errCode, Msg: msg, Data: data})
}

// statusErrCode 4xx 未分类错误按状态给通用码
func statusErrCode(code int) string {
	switch code {
	case http.StatusBadRequest:
		return CodeValidation
	case http.StatusUnauthorized:
		return CodeUnauthorized
	case http.StatusForbidden:
		return CodeForbidden
	case http.StatusTooManyRequests:
		return CodeRateLimited
	default:
		return ""
	}
}

// WrapFunc will wrap func(args ...interface{}) (interface{}, <error>) as a Gin HandlerFunc
func WrapFunc(f interface{}, args ...interface{}) gin.HandlerFunc {
	fn := reflect.ValueOf(f)
	if fn.Type().NumIn() != len(args) {
		panic(fmt.Sprintf("invaild input parameters of function %v", fn.Type()))
	}

	outNum := fn.Type().NumOut()
	if outNum == 0 {
		panic(fmt.Sprintf("invaild output parameters of function %v, at least one, but got %d", fn.Type(), outNum))
	}

	inputs := make([]reflect.Value, len(args))
	for k, in := range args {
		inputs[k] = reflect.ValueOf(in)
	}

	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logrus.Warnf("panic: %v", err)
				ResponseFailed(c, http.StatusInternalServerError, fmt.Errorf("%v", err))
			}
		}()

		outputs := fn.Call(inputs)
		if len(outputs) > 1 {
			err, ok := outputs[len(outputs)-1].Interface().(error)
			if ok && err != nil {
				ResponseFailed(c, http.StatusInternalServerError, err)
				return
			}
		}
		c.JSON(http.StatusOK, outputs[0].Interface())
	}
}
