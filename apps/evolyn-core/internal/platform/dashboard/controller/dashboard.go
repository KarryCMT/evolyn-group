// Package controller 暴露仪表盘资产管理态 API；路由挂租户认证、状态与 RBAC 链。
package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"evolyn/internal/metrics"
	platformcontroller "evolyn/internal/platform/controller"
	dashboarderrors "evolyn/internal/platform/dashboard"
	dashboardmodel "evolyn/internal/platform/dashboard/model"
	"evolyn/internal/platform/dashboard/service"
	"evolyn/internal/platform/ginctx"
	"evolyn/internal/platform/httpx"
	"evolyn/internal/utils/trace"

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

// ListFormDataSources godoc
// @Summary 列出仪表盘可用表单数据源
// @Description 仅返回同应用内已发布且当前成员具有记录查看权限的表单
// @Produce json
// @Tags 仪表盘管理
// @Security JWT
// @Param code path string true "dashboard_ 公开编码"
// @Success 200 {object} httpx.Response{data=[]dashboardmodel.FormDataSource}
// @Failure 404 {object} httpx.Response "errCode=DASHBOARD_DATA_SOURCE_UNAVAILABLE"
// @Router /api/v1/dashboards/{code}/data-sources/forms [get]
func (d *DashboardController) ListFormDataSources(c *gin.Context) {
	code, ok := dashboardCode(c)
	if !ok {
		return
	}
	items, err := d.service.ListFormDataSources(c.Request.Context(), ginctx.GetUser(c), code)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, items)
}

// GetFormFieldCatalog godoc
// @Summary 获取仪表盘表单字段目录
// @Description 从不可变发布快照返回稳定 fieldId 与权限裁剪后的 Query DSL 能力
// @Produce json
// @Tags 仪表盘管理
// @Security JWT
// @Param code path string true "dashboard_ 公开编码"
// @Param formCode path string true "form_ 公开编码"
// @Success 200 {object} httpx.Response{data=dashboardmodel.FormFieldCatalog}
// @Failure 404 {object} httpx.Response "errCode=DASHBOARD_DATA_SOURCE_UNAVAILABLE"
// @Router /api/v1/dashboards/{code}/data-sources/forms/{formCode}/fields [get]
func (d *DashboardController) GetFormFieldCatalog(c *gin.Context) {
	code, ok := dashboardCode(c)
	if !ok {
		return
	}
	formCode := strings.TrimSpace(c.Param("formCode"))
	if !strings.HasPrefix(formCode, "form_") {
		httpx.ResponseFailed(c, http.StatusBadRequest, fmt.Errorf("无效的表单编码"))
		return
	}
	catalog, err := d.service.GetFormFieldCatalog(c.Request.Context(), ginctx.GetUser(c), code, formCode)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, catalog)
}

// PreviewWidgetQuery godoc
// @Summary 查询单个草稿组件预览数据
// @Description 查询语义仅从指定已保存草稿 revision 恢复，请求不能提交 Dataset 或 Query AST
// @Accept json
// @Produce json
// @Tags 仪表盘管理
// @Security JWT
// @Param code path string true "dashboard_ 公开编码"
// @Param widgetId path string true "组件稳定 ID"
// @Param query body dashboardmodel.PreviewQueryRequest true "草稿 revision 与运行时分页"
// @Success 200 {object} httpx.Response{data=dashboardmodel.PreviewQueryResult}
// @Failure 400 {object} httpx.Response "errCode=DASHBOARD_QUERY_INVALID"
// @Failure 409 {object} httpx.Response "errCode=DASHBOARD_DRAFT_CONFLICT"
// @Failure 422 {object} httpx.Response "errCode=DASHBOARD_QUERY_LIMIT_EXCEEDED"
// @Router /api/v1/dashboards/{code}/widgets/{widgetId}/preview-query [post]
func (d *DashboardController) PreviewWidgetQuery(c *gin.Context) {
	startedAt := time.Now()
	code, ok := dashboardCode(c)
	if !ok {
		return
	}
	req := new(dashboardmodel.PreviewQueryRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	result, err := d.service.PreviewWidgetQuery(c.Request.Context(), ginctx.GetUser(c), code, c.Param("widgetId"), req)
	resultClass := dashboardQueryResultClass(err)
	rows := 0
	if result != nil {
		rows = len(result.Rows)
	}
	duration := time.Since(startedAt)
	metrics.DashboardQueryTotal.WithLabelValues(resultClass).Inc()
	metrics.DashboardQueryDuration.WithLabelValues(resultClass).Observe(duration.Seconds())
	if err == nil {
		metrics.DashboardQueryRows.Observe(float64(rows))
	}
	member := ginctx.GetUser(c)
	tenantID := uint(0)
	if member != nil {
		tenantID = member.TenantID
	}
	ginctx.TraceStep(c, "dashboard widget query",
		trace.Field{Key: "tenantId", Value: tenantID},
		trace.Field{Key: "dashboardCode", Value: code},
		trace.Field{Key: "widgetId", Value: c.Param("widgetId")},
		trace.Field{Key: "draftRevision", Value: req.DraftRevision},
		trace.Field{Key: "result", Value: resultClass},
		trace.Field{Key: "rowCount", Value: rows},
		trace.Field{Key: "durationMs", Value: duration.Milliseconds()},
	)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, result)
}

// dashboardQueryResultClass 将内部错误收敛为低基数稳定分类；超时通过
// BizError 的 unwrap 链识别，仍对客户端复用 QUERY_LIMIT_EXCEEDED。
func dashboardQueryResultClass(err error) string {
	if err == nil {
		return "success"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	if errors.Is(err, dashboarderrors.ErrQueryLimitExceeded) {
		return "limit_exceeded"
	}
	if errors.Is(err, dashboarderrors.ErrQueryInvalid) || errors.Is(err, dashboarderrors.ErrSchemaInvalid) {
		return "invalid"
	}
	if errors.Is(err, dashboarderrors.ErrForbidden) {
		return "forbidden"
	}
	var biz *httpx.BizError
	if errors.As(err, &biz) {
		return "biz_" + strconv.Itoa(biz.HTTP)
	}
	return "internal"
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
	api.GET("/dashboards/:code/data-sources/forms", d.ListFormDataSources)
	api.GET("/dashboards/:code/data-sources/forms/:formCode/fields", d.GetFormFieldCatalog)
	api.POST("/dashboards/:code/widgets/:widgetId/preview-query", d.PreviewWidgetQuery)
	api.DELETE("/dashboards/:code", d.Delete)
}

func (d *DashboardController) Name() string { return "Dashboard" }
