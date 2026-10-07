// Package controller 暴露仪表盘资产管理态 API；路由挂租户认证、状态与 RBAC 链。
package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	platformcontroller "evolyn/internal/platform/controller"
	dashboardmodel "evolyn/internal/platform/dashboard/model"
	"evolyn/internal/platform/dashboard/service"
	"evolyn/internal/platform/ginctx"
	"evolyn/internal/platform/httpx"

	"github.com/gin-gonic/gin"
)

type DashboardController struct{ service service.DashboardService }

func NewDashboardController(svc service.DashboardService) platformcontroller.Controller {
	return &DashboardController{service: svc}
}

func responseError(c *gin.Context, err error) {
	var biz *httpx.BizError
	if errors.As(err, &biz) && biz.HTTP != 0 {
		httpx.ResponseFailed(c, biz.HTTP, err)
		return
	}
	httpx.ResponseFailed(c, http.StatusInternalServerError, err)
}

func dashboardCode(c *gin.Context) (string, bool) {
	code := strings.TrimSpace(c.Param("code"))
	if !strings.HasPrefix(code, "dashboard_") || len(code) <= len("dashboard_") {
		httpx.ResponseFailed(c, http.StatusBadRequest, fmt.Errorf("无效的仪表盘编码"))
		return "", false
	}
	return code, true
}

// Precreate godoc
// @Summary 预创建仪表盘
// @Description 在一个事务内完成幂等占位、dashboards 配额判定、空草稿和应用菜单节点创建；相同 requestId 与载荷安全返回原结果
// @Accept json
// @Produce json
// @Tags 仪表盘管理
// @Security JWT
// @Param code path string true "应用公开编码"
// @Param dashboard body dashboardmodel.PrecreateRequest true "创建幂等标识、默认名称与可选父分组"
// @Success 201 {object} httpx.Response{data=dashboardmodel.Detail}
// @Failure 400 {object} httpx.Response "errCode=DASHBOARD_NAME_INVALID/DASHBOARD_APP_INVALID/APP_MENU_PARENT_INVALID"
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN/QUOTA_EXCEEDED"
// @Failure 409 {object} httpx.Response "errCode=DASHBOARD_IDEMPOTENCY_CONFLICT"
// @Router /api/v1/apps/code/{code}/dashboards [post]
func (d *DashboardController) Precreate(c *gin.Context) {
	req := new(dashboardmodel.PrecreateRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	detail, err := d.service.Precreate(c.Request.Context(), ginctx.GetUser(c), c.Param("code"), req)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.NewResponse(c, http.StatusCreated, detail, "创建成功")
}

// Get godoc
// @Summary 仪表盘详情
// @Description 返回展示信息、完整草稿、草稿 revision 与发布摘要
// @Produce json
// @Tags 仪表盘管理
// @Security JWT
// @Param code path string true "dashboard_ 公开编码"
// @Success 200 {object} httpx.Response{data=dashboardmodel.Detail}
// @Failure 404 {object} httpx.Response "errCode=DASHBOARD_NOT_FOUND"
// @Router /api/v1/dashboards/{code} [get]
func (d *DashboardController) Get(c *gin.Context) {
	code, ok := dashboardCode(c)
	if !ok {
		return
	}
	detail, err := d.service.Get(c.Request.Context(), ginctx.GetUser(c), code)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, detail)
}

// Update godoc
// @Summary 更新仪表盘展示信息或移动菜单节点
// @Accept json
// @Produce json
// @Tags 仪表盘管理
// @Security JWT
// @Param code path string true "dashboard_ 公开编码"
// @Param dashboard body dashboardmodel.UpdateRequest true "名称、图标、颜色或父分组"
// @Success 200 {object} httpx.Response{data=dashboardmodel.Detail}
// @Router /api/v1/dashboards/{code} [patch]
func (d *DashboardController) Update(c *gin.Context) {
	code, ok := dashboardCode(c)
	if !ok {
		return
	}
	req := new(dashboardmodel.UpdateRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	detail, err := d.service.Update(c.Request.Context(), ginctx.GetUser(c), code, req)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, detail)
}

// SaveDraft godoc
// @Summary 保存仪表盘草稿
// @Description 服务端规范化完整文档，并按 expectedRevision 条件更新；冲突返回 HTTP 409
// @Accept json
// @Produce json
// @Tags 仪表盘管理
// @Security JWT
// @Param code path string true "dashboard_ 公开编码"
// @Param draft body dashboardmodel.SaveDraftRequest true "完整草稿与期望 revision"
// @Success 200 {object} httpx.Response{data=dashboardmodel.SaveDraftResult}
// @Failure 400 {object} httpx.Response "errCode=DASHBOARD_SCHEMA_INVALID"
// @Failure 409 {object} httpx.Response "errCode=DASHBOARD_DRAFT_CONFLICT"
// @Router /api/v1/dashboards/{code}/draft [put]
func (d *DashboardController) SaveDraft(c *gin.Context) {
	code, ok := dashboardCode(c)
	if !ok {
		return
	}
	req := new(dashboardmodel.SaveDraftRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	result, err := d.service.SaveDraft(c.Request.Context(), ginctx.GetUser(c), code, req)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, result)
}

// Delete godoc
// @Summary 删除仪表盘
// @Description 软删除资产并在同一事务内摘除菜单节点与个人收藏
// @Produce json
// @Tags 仪表盘管理
// @Security JWT
// @Param code path string true "dashboard_ 公开编码"
// @Success 200 {object} httpx.Response
// @Router /api/v1/dashboards/{code} [delete]
func (d *DashboardController) Delete(c *gin.Context) {
	code, ok := dashboardCode(c)
	if !ok {
		return
	}
	if err := d.service.Delete(c.Request.Context(), ginctx.GetUser(c), code); err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, nil)
}

func (d *DashboardController) RegisterRoute(api *gin.RouterGroup) {
	api.POST("/apps/code/:code/dashboards", d.Precreate)
	api.GET("/dashboards/:code", d.Get)
	api.PATCH("/dashboards/:code", d.Update)
	api.PUT("/dashboards/:code/draft", d.SaveDraft)
	api.DELETE("/dashboards/:code", d.Delete)
}

func (d *DashboardController) Name() string { return "Dashboard" }
