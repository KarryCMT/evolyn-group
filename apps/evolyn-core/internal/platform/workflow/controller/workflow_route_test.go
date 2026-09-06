package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestWorkflowSaveDraftRouteUsesDedicatedDraftPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gin.Recovery())
	(&WorkflowController{}).RegisterRoute(router.Group(""))

	request := httptest.NewRequest(http.MethodPut, "/workflows/wf_3454e9914ed99498/draft", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	// 路由命中后会先因空请求体被 BindJSON 拒绝；404 说明草稿路径未注册。
	assert.Equal(t, http.StatusBadRequest, response.Code)
}
