package httpx

import (
	"fmt"
	"net/http"
	"reflect"

	"evolyn/internal/platform/ginctx"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Cookie 名称（登录态）：token 为 JWT，loginUser 为前端展示用的用户信息
const (
	CookieTokenName = `token`
	CookieLoginUser = `loginUser`
)

// Response 统一响应结构
type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
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

// ResponseFailed 401 时顺带清理登录 Cookie；err 非空时记录请求上下文日志
func ResponseFailed(c *gin.Context, code int, err error) {
	if code == 0 {
		code = http.StatusInternalServerError
	}
	if code == http.StatusUnauthorized && c.Request != nil {
		if val, err := c.Cookie(CookieTokenName); err == nil && val != "" {
			c.SetCookie(CookieTokenName, "", -1, "/", "", true, true)
			c.SetCookie(CookieLoginUser, "", -1, "/", "", true, false)
		}
	}

	var msg string
	if err != nil {
		msg = err.Error()
		user := ginctx.GetUser(c)
		var name string
		if user != nil {
			name = user.Name
		}
		var url string
		if c.Request != nil {
			url = c.Request.URL.String()
		}
		logrus.Warnf("url: %s, user: %s, error: %v", url, name, msg)
	}
	NewResponse(c, code, nil, msg)
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
