package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func failedBody(t *testing.T, code int, err error) Response {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)

	ResponseFailed(c, code, err)

	var resp Response
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, w.Code, resp.Code, "信封 code 应与 HTTP 状态一致")
	return resp
}

func TestBizErrorMapping(t *testing.T) {
	biz := NewBiz("DUPLICATE_PHONE", "手机号已注册", http.StatusConflict)

	// 直接传递
	resp := failedBody(t, http.StatusBadRequest, biz)
	assert.Equal(t, http.StatusConflict, resp.Code)
	assert.Equal(t, "DUPLICATE_PHONE", resp.ErrCode)
	assert.Equal(t, "手机号已注册", resp.Msg)

	// %w 包装链：码与文案保持，原始错误不出网
	wrapped := fmt.Errorf("register step: %w", Wrap(biz, errors.New("pg duplicate key")))
	resp = failedBody(t, 0, wrapped)
	assert.Equal(t, http.StatusConflict, resp.Code)
	assert.Equal(t, "DUPLICATE_PHONE", resp.ErrCode)
	assert.Equal(t, "手机号已注册", resp.Msg)
	assert.NotContains(t, resp.Msg, "pg duplicate key")
}

func TestBizErrorErrorsIsCompat(t *testing.T) {
	biz := NewBiz("QUOTA_EXCEEDED", "配额已用尽", http.StatusForbidden)
	wrapped := fmt.Errorf("add member: %w", biz)

	// Is 按 Code 判定：包装链上哨兵判定仍成立（兼容既有调用方）
	assert.ErrorIs(t, wrapped, biz)
	assert.True(t, errors.Is(wrapped, NewBiz("QUOTA_EXCEEDED", "另一个文案", 403)))
	assert.False(t, errors.Is(wrapped, NewBiz("OTHER", "x", 400)))

	// As 取回完整结构
	var got *BizError
	assert.ErrorAs(t, wrapped, &got)
	assert.Equal(t, "配额已用尽", got.Msg)
}

func TestRecordNotFoundMapping(t *testing.T) {
	// GORM not found：无论调用方给什么码，统一 404 + NOT_FOUND
	resp := failedBody(t, http.StatusBadRequest, gorm.ErrRecordNotFound)
	assert.Equal(t, http.StatusNotFound, resp.Code)
	assert.Equal(t, CodeNotFound, resp.ErrCode)
	assert.Equal(t, "记录不存在", resp.Msg)
}

func TestUnknownErrorSanitized(t *testing.T) {
	// 5xx 未知错误：脱敏为通用文案 + INTERNAL_SERVER，原文不出网
	resp := failedBody(t, http.StatusInternalServerError, errors.New("sql: no rows in result set (table users, tenant 3)"))
	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.Equal(t, CodeInternalServer, resp.ErrCode)
	assert.Equal(t, internalServerMsg, resp.Msg)
	assert.NotContains(t, resp.Msg, "sql:")

	// code=0 兜底 500 同样脱敏
	resp = failedBody(t, 0, errors.New("dial tcp 127.0.0.1:5432: refuse"))
	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.Equal(t, CodeInternalServer, resp.ErrCode)
}

func TestClientErrorKept(t *testing.T) {
	// 4xx 未分类错误：保留调用方判定与文案（参数/会话类），errCode 给通用码
	resp := failedBody(t, http.StatusBadRequest, errors.New("department name is required"))
	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Equal(t, CodeValidation, resp.ErrCode)
	assert.Equal(t, "department name is required", resp.Msg)
}

func TestSuccessEnvelopeNoErrCode(t *testing.T) {
	// 成功响应不携带 errCode（omitempty），兼容既有前端
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	ResponseSuccess(c, map[string]string{"ok": "1"})

	body := w.Body.String()
	assert.Contains(t, body, `"code":200`)
	assert.NotContains(t, body, "errCode")
}
