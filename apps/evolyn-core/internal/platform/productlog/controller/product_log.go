// Package controller 产品日志域 HTTP 接口（/product-logs）：全部挂租户域链
// （Authentication → Tenant → TenantStatus → Authorization）。资源级权限由
// 中间件链执行：GET /product-logs 与 /product-logs/options 解析为 list、
// GET /exports/:id 及下载路径解析为 get（对应 product-logs:view），POST
// /exports 解析为 create（对应 product-logs:export）；下载在 get 之上由
// 控制器复核 create 动词——view 权限可查任务状态但不可下载文件
package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"evolyn/internal/contextx"
	platformcontroller "evolyn/internal/platform/controller"
	"evolyn/internal/platform/ginctx"
	"evolyn/internal/platform/httpx"
	iammodel "evolyn/internal/platform/iam/model"
	"evolyn/internal/platform/productlog"
	"evolyn/internal/platform/productlog/model"
	"evolyn/internal/platform/productlog/service"
	"evolyn/internal/utils/request"

	"github.com/gin-gonic/gin"
)

// ProductLogController 产品日志（/product-logs）
type ProductLogController struct {
	logService service.ProductLogService
	authorizer ExportAuthorizer
}

// ExportAuthorizer 导出权限复核窄端口（iam authorization.Authorizer 结构性
// 满足）：下载路径在路由级 get（view）之上复核 create（export）能力
type ExportAuthorizer interface {
	Authorize(ctx context.Context, user *iammodel.User, ri *request.RequestInfo) (bool, error)
}

// NewProductLogController 产品日志控制器工厂
func NewProductLogController(logService service.ProductLogService, authorizer ExportAuthorizer) platformcontroller.Controller {
	return &ProductLogController{logService: logService, authorizer: authorizer}
}

func (p *ProductLogController) Name() string {
	return "产品日志"
}

// swagger 出网类型别名（本包不直接构造，仅为文档解析提供定位）
type (
	ProductLogPage      = model.ProductLogPage
	ProductLogItem      = model.ProductLogItem
	ProductLogOptions   = model.ProductLogOptions
	ExportTaskView      = model.ExportTaskView
	CreateExportRequest = model.CreateExportRequest
)

func (p *ProductLogController) RegisterRoute(api *gin.RouterGroup) {
	api.GET("/product-logs", p.List)
	api.GET("/product-logs/options", p.Options)
	api.POST("/product-logs/exports", p.CreateExport)
	api.GET("/product-logs/exports/:id", p.GetExport)
	api.GET("/product-logs/exports/:id/download", p.DownloadExport)
}

// resolveTenant 解析租户上下文：缺失按 401 拒绝（与企业日志域同口径，
// 防御无租户上下文的裸请求）
func resolveTenant(c *gin.Context) (uint, bool) {
	tenantID, ok := contextx.TenantIDFromContext(c.Request.Context())
	if !ok {
		httpx.ResponseFailed(c, http.StatusUnauthorized, fmt.Errorf("tenant context required"))
		return 0, false
	}
	return tenantID, true
}

// parseUintQueryParam 查询参数 → uint；缺省返回 0，非法值返回错误
func parseUintQueryParam(c *gin.Context, key string) (uint, error) {
	raw := c.Query(key)
	if raw == "" {
		return 0, nil
	}
	v, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, httpx.NewBiz(httpx.CodeValidation, "参数 "+key+" 必须为数字", http.StatusBadRequest)
	}
	return uint(v), nil
}

// parseIntPathParam 路径参数 → uint；非法值返回错误
func parseIntPathParam(c *gin.Context, key string) (uint, error) {
	v, err := strconv.ParseUint(c.Param(key), 10, 64)
	if err != nil {
		return 0, httpx.NewBiz(httpx.CodeValidation, "路径参数 "+key+" 必须为数字", http.StatusBadRequest)
	}
	return uint(v), nil
}

// canExport 复核 product-logs:export 能力（路由级 get 之上的第二道门）
func (p *ProductLogController) canExport(c *gin.Context) bool {
	user := ginctx.GetUser(c)
	if user == nil || p.authorizer == nil {
		return false
	}
	ok, err := p.authorizer.Authorize(c.Request.Context(), user, &request.RequestInfo{
		IsResourceRequest: true,
		Resource:          iammodel.ProductLogResource,
		Verb:              request.CreateOperation,
	})
	return err == nil && ok
}

// List 产品日志分页查询。
//
// @Summary 查询产品日志
// @Description 当前租户内各应用及应用内资源（应用/菜单/表单/流程/数据/应用权限）的操作流水，按操作时间倒序；出参仅含操作人/时间/范围/类型/所属应用/对象/脱敏详情与 IP，历史记录降级展示为「历史操作记录」
// @Produce json
// @Tags 产品日志
// @Security JWT
// @Param categoryCode query string false "日志范围码（见筛选项接口）"
// @Param eventCode query string false "操作类型事件码"
// @Param memberId query int false "当前租户成员 ID"
// @Param applicationId query int false "当前租户应用 ID"
// @Param keyword query string false "关键词（匹配所属应用/操作对象/操作详情）"
// @Param startAt query string false "开始日期（yyyy-MM-dd 东八区）"
// @Param endAt query string false "结束日期（yyyy-MM-dd 东八区）"
// @Param page query int false "页码，默认 1"
// @Param pageSize query int false "每页条数，默认 20，上限 100"
// @Success 200 {object} httpx.Response{data=controller.ProductLogPage}
// @Failure 400 {object} httpx.Response "errCode=PRODUCT_LOG_DATE_INVALID / PRODUCT_LOG_TIME_RANGE_INVALID / PRODUCT_LOG_MEMBER_INVALID / PRODUCT_LOG_APPLICATION_INVALID / PRODUCT_LOG_CATEGORY_UNKNOWN / PRODUCT_LOG_EVENT_UNKNOWN"
// @Failure 401 {object} httpx.Response "errCode=UNAUTHORIZED"
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN"
// @Router /api/v1/product-logs [get]
func (p *ProductLogController) List(c *gin.Context) {
	tenantID, ok := resolveTenant(c)
	if !ok {
		return
	}
	memberID, err := parseUintQueryParam(c, "memberId")
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	applicationID, err := parseUintQueryParam(c, "applicationId")
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("pageSize"))

	result, err := p.logService.List(c.Request.Context(), tenantID, model.ProductLogQuery{
		CategoryCode:  c.Query("categoryCode"),
		EventCode:     c.Query("eventCode"),
		MemberID:      memberID,
		ApplicationID: applicationID,
		Keyword:       c.Query("keyword"),
		StartDate:     c.Query("startAt"),
		EndDate:       c.Query("endAt"),
		Page:          page,
		PageSize:      pageSize,
	})
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, result)
}

// Options 产品日志筛选项。
//
// @Summary 查询产品日志筛选项
// @Description 返回产品日志分类及各自可选的操作类型事件码、当前租户可选操作人与有效应用清单，供筛选下拉使用（前端不硬编码）
// @Produce json
// @Tags 产品日志
// @Security JWT
// @Success 200 {object} httpx.Response{data=controller.ProductLogOptions}
// @Failure 401 {object} httpx.Response "errCode=UNAUTHORIZED"
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN"
// @Router /api/v1/product-logs/options [get]
func (p *ProductLogController) Options(c *gin.Context) {
	tenantID, ok := resolveTenant(c)
	if !ok {
		return
	}
	result, err := p.logService.Options(c.Request.Context(), tenantID)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, result)
}

// CreateExport 创建产品日志导出任务。
//
// @Summary 创建产品日志导出任务
// @Description 携带与列表完全一致的筛选条件；任务固化筛选/申请人/租户/数据量/状态与过期时间。一期同步生成（上限 50000 行，超限提示缩小范围），导出文件 24 小时有效；导出行为落企业治理类操作审计但不记录文件内容
// @Accept json
// @Produce json
// @Tags 产品日志
// @Security JWT
// @Param body body controller.CreateExportRequest true "筛选条件（与列表同构）"
// @Success 200 {object} httpx.Response{data=controller.ExportTaskView}
// @Failure 400 {object} httpx.Response "errCode=PRODUCT_LOG_EXPORT_TOO_LARGE / PRODUCT_LOG_DATE_INVALID / PRODUCT_LOG_TIME_RANGE_INVALID / PRODUCT_LOG_MEMBER_INVALID / PRODUCT_LOG_APPLICATION_INVALID / PRODUCT_LOG_CATEGORY_UNKNOWN / PRODUCT_LOG_EVENT_UNKNOWN"
// @Failure 401 {object} httpx.Response "errCode=UNAUTHORIZED"
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN"
// @Router /api/v1/product-logs/exports [post]
func (p *ProductLogController) CreateExport(c *gin.Context) {
	tenantID, ok := resolveTenant(c)
	if !ok {
		return
	}
	req := new(model.CreateExportRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	ginctx.TraceStep(c, "create product log export")
	result, err := p.logService.CreateExport(c.Request.Context(), tenantID, *req)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, result)
}

// GetExport 导出任务状态。
//
// @Summary 查询导出任务状态
// @Description 复核任务所属租户；ready 且已过期的任务投影为 expired 状态
// @Produce json
// @Tags 产品日志
// @Security JWT
// @Param id path int true "导出任务 ID"
// @Success 200 {object} httpx.Response{data=controller.ExportTaskView}
// @Failure 400 {object} httpx.Response "errCode=VALIDATION"
// @Failure 401 {object} httpx.Response "errCode=UNAUTHORIZED"
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN"
// @Failure 404 {object} httpx.Response "errCode=PRODUCT_LOG_EXPORT_NOT_FOUND"
// @Router /api/v1/product-logs/exports/{id} [get]
func (p *ProductLogController) GetExport(c *gin.Context) {
	tenantID, ok := resolveTenant(c)
	if !ok {
		return
	}
	taskID, err := parseIntPathParam(c, "id")
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	result, err := p.logService.GetExport(c.Request.Context(), tenantID, taskID)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, result)
}

// DownloadExport 下载导出文件。
//
// @Summary 下载导出文件
// @Description 在路由级 get（查看权限）之上复核导出权限（product-logs:export）与任务租户归属、有效期后返回 CSV 文件流
// @Produce text/csv
// @Tags 产品日志
// @Security JWT
// @Param id path int true "导出任务 ID"
// @Success 200 {file} file "CSV 文件流"
// @Failure 400 {object} httpx.Response "errCode=VALIDATION"
// @Failure 401 {object} httpx.Response "errCode=UNAUTHORIZED"
// @Failure 403 {object} httpx.Response "errCode=PRODUCT_LOG_EXPORT_FORBIDDEN / FORBIDDEN"
// @Failure 404 {object} httpx.Response "errCode=PRODUCT_LOG_EXPORT_NOT_FOUND"
// @Failure 409 {object} httpx.Response "errCode=PRODUCT_LOG_EXPORT_NOT_READY"
// @Failure 410 {object} httpx.Response "errCode=PRODUCT_LOG_EXPORT_EXPIRED"
// @Router /api/v1/product-logs/exports/{id}/download [get]
func (p *ProductLogController) DownloadExport(c *gin.Context) {
	// 下载路径复核导出权限：查看权限可看任务状态但不可取文件
	if !p.canExport(c) {
		httpx.ResponseFailed(c, http.StatusForbidden, productlog.ErrExportForbidden)
		return
	}
	tenantID, ok := resolveTenant(c)
	if !ok {
		return
	}
	taskID, err := parseIntPathParam(c, "id")
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	ginctx.TraceStep(c, "download product log export")
	file, err := p.logService.ExportFile(c.Request.Context(), tenantID, taskID)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	// RFC 5987：中文文件名经 filename* 编码下发，兼容主流浏览器
	c.Header("Content-Disposition",
		fmt.Sprintf(`attachment; filename*=UTF-8''%s`, url.PathEscape(file.FileName)))
	c.Data(http.StatusOK, file.ContentType, file.Data)
}
