package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestWorkflowTaskSummaryRouteIsOutsideTaskIDNamespace(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gin.Recovery())
	(&WorkflowTaskController{}).RegisterRoute(router.Group(""))

	request := httptest.NewRequest(http.MethodGet, "/workflow-task-summaries/current", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	// PendingSummary 的未注入 service 会触发 Recovery 并返回 500。若路径被
	// 错误地纳入任务详情路由，parseUintParam 会返回 400；这能锁定资源隔离。
	assert.Equal(t, http.StatusInternalServerError, response.Code)
}
