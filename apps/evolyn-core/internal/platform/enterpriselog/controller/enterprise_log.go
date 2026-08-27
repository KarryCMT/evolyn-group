// Package controller 企业日志域 HTTP 接口（/enterprise-logs）：全部挂租户
// 域链（Authentication → Tenant → TenantStatus → Authorization）。资源级
// 权限由中间件链执行：三条 GET 集合路径解析为 list、GET /exports/:id 及
// 下载路径解析为 get（对应 enterprise-logs:view），POST /exports 解析为
// create（对应 enterprise-logs:export）；下载在 get 之上由控制器复核
// create 动词——view 权限可查任务状态但不可下载文件
package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"evolyn/internal/contextx"
	platformcontroller "evolyn/internal/platform/controller"
	"evolyn/internal/platform/enterpriselog"
	"evolyn/internal/platform/enterpriselog/model"
	"evolyn/internal/platform/enterpriselog/service"
	"evolyn/internal/platform/ginctx"
	"evolyn/internal/platform/httpx"
	iammodel "evolyn/internal/platform/iam/model"
	"evolyn/internal/utils/request"

	"github.com/gin-gonic/gin"
)

// EnterpriseLogController 企业日志（/enterprise-logs）
type EnterpriseLogController struct {
	logService service.EnterpriseLogService
	authorizer ExportAuthorizer
}

// ExportAuthorizer 导出权限复核窄端口（iam authorization.Authorizer 结构性
// 满足）：下载路径在路由级 get（view）之上复核 create（export）能力
type ExportAuthorizer interface {
	Authorize(ctx context.Context, user *iammodel.User, ri *request.RequestInfo) (bool, error)
}

// NewEnterpriseLogController 企业日志控制器工厂
func NewEnterpriseLogController(logService service.EnterpriseLogService, authorizer ExportAuthorizer) platformcontroller.Controller {
	return &EnterpriseLogController{logService: logService, authorizer: authorizer}
}

func (e *EnterpriseLogController) Name() string {
	return "企业日志"
}

// swagger 出网类型别名（本包不直接构造，仅为文档解析提供定位）
type (
	LoginLogPage        = model.LoginLogPage
	LoginLogItem        = model.LoginLogItem
	OperationLogPage    = model.OperationLogPage
	OperationLogItem    = model.OperationLogItem
	CategoryOption      = model.CategoryOption
	ExportTaskView      = model.ExportTaskView
	CreateExportRequest = model.CreateExportRequest
)

func (e *EnterpriseLogController) RegisterRoute(api *gin.RouterGroup) {
	api.GET("/enterprise-logs/login", e.ListLoginLogs)
	api.GET("/enterprise-logs/operations", e.ListOperationLogs)
	api.GET("/enterprise-logs/operation-categories", e.ListOperationCategories)
	api.POST("/enterprise-logs/exports", e.CreateExport)
	api.GET("/enterprise-logs/exports/:id", e.GetExport)
	api.GET("/enterprise-logs/exports/:id/download", e.DownloadExport)
}

// resolveTenant 解析租户上下文：缺失按 401 拒绝（与产品中心域同口径，
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

// canExport 复核 enterprise-logs:export 能力（路由级 get 之上的第二道门）
func (e *EnterpriseLogController) canExport(c *gin.Context) bool {
	user := ginctx.GetUser(c)
	if user == nil || e.authorizer == nil {
		return false
	}
	ok, err := e.authorizer.Authorize(c.Request.Context(), user, &request.RequestInfo{
		IsResourceRequest: true,
		Resource:          iammodel.EnterpriseLogResource,
		Verb:              request.CreateOperation,
	})
	return err == nil && ok
}

// ListLoginLogs 登录日志分页查询。
//
// @Summary 查询登录日志
// @Description 当前租户成员的会话建立流水（按登录时间倒序）；登录人筛选按成员 ID 并校验租户归属，时间为东八区自然日闭区间
// @Produce json
// @Tags 企业日志
// @Security JWT
// @Param memberId query int false "当前租户成员 ID"
// @Param startAt query string false "开始日期（yyyy-MM-dd 东八区）"
// @Param endAt query string false "结束日期（yyyy-MM-dd 东八区）"
// @Param page query int false "页码，默认 1"
// @Param pageSize query int false "每页条数，默认 20，上限 100"
// @Success 200 {object} httpx.Response{data=controller.LoginLogPage}
// @Failure 400 {object} httpx.Response "errCode=ENTERPRISE_LOG_DATE_INVALID / ENTERPRISE_LOG_TIME_RANGE_INVALID / ENTERPRISE_LOG_MEMBER_INVALID"
// @Failure 401 {object} httpx.Response "errCode=UNAUTHORIZED"
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN"
// @Router /api/v1/enterprise-logs/login [get]
func (e *EnterpriseLogController) ListLoginLogs(c *gin.Context) {
	tenantID, ok := resolveTenant(c)
	if !ok {
		return
	}
	memberID, err := parseUintQueryParam(c, "memberId")
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("pageSize"))

	result, err := e.logService.ListLoginLogs(c.Request.Context(), tenantID, model.LoginLogQuery{
		MemberID:  memberID,
		StartDate: c.Query("startAt"),
		EndDate:   c.Query("endAt"),
		Page:      page,
		PageSize:  pageSize,
	})
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, result)
}

// ListOperationLogs 操作日志分页查询。
//
// @Summary 查询操作日志
// @Description 当前租户的管理与业务变更审计（按发生时间倒序）；出参仅含操作人/时间/范围/类型/脱敏详情与 IP，历史记录降级展示为「历史操作记录」
// @Produce json
// @Tags 企业日志
// @Security JWT
// @Param memberId query int false "当前租户成员 ID"
// @Param categoryCode query string false "日志范围码（见筛选项接口）"
// @Param eventCode query string false "操作类型事件码"
// @Param startAt query string false "开始日期（yyyy-MM-dd 东八区）"
// @Param endAt query string false "结束日期（yyyy-MM-dd 东八区）"
// @Param page query int false "页码，默认 1"
// @Param pageSize query int false "每页条数，默认 20，上限 100"
// @Success 200 {object} httpx.Response{data=controller.OperationLogPage}
// @Failure 400 {object} httpx.Response "errCode=ENTERPRISE_LOG_DATE_INVALID / ENTERPRISE_LOG_TIME_RANGE_INVALID / ENTERPRISE_LOG_MEMBER_INVALID / ENTERPRISE_LOG_CATEGORY_UNKNOWN / ENTERPRISE_LOG_EVENT_UNKNOWN"
// @Failure 401 {object} httpx.Response "errCode=UNAUTHORIZED"
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN"
// @Router /api/v1/enterprise-logs/operations [get]
func (e *EnterpriseLogController) ListOperationLogs(c *gin.Context) {
	tenantID, ok := resolveTenant(c)
	if !ok {
		return
	}
	memberID, err := parseUintQueryParam(c, "memberId")
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("pageSize"))

	result, err := e.logService.ListOperationLogs(c.Request.Context(), tenantID, model.OperationLogQuery{
		MemberID:     memberID,
		CategoryCode: c.Query("categoryCode"),
		EventCode:    c.Query("eventCode"),
		StartDate:    c.Query("startAt"),
		EndDate:      c.Query("endAt"),
		Page:         page,
		PageSize:     pageSize,
	})
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, result)
}

// ListOperationCategories 操作日志筛选项。
//
// @Summary 查询日志范围与操作类型筛选项
// @Description 返回日志范围（分类）与各自可选的操作类型事件码清单，供操作日志页筛选下拉使用
// @Produce json
// @Tags 企业日志
// @Security JWT
// @Success 200 {object} httpx.Response{data=[]controller.CategoryOption}
// @Failure 401 {object} httpx.Response "errCode=UNAUTHORIZED"
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN"
// @Router /api/v1/enterprise-logs/operation-categories [get]
func (e *EnterpriseLogController) ListOperationCategories(c *gin.Context) {
	httpx.ResponseSuccess(c, e.logService.ListCategories())
}

// CreateExport 创建日志导出任务。
//
// @Summary 创建日志导出任务
// @Description 携带日志类型（login/operation）与列表同构的筛选条件；任务固化筛选/申请人/租户/数据量/状态与过期时间。一期同步生成（上限 50000 行，超限提示缩小范围），导出文件 24 小时有效；导出行为落操作审计但不记录文件内容
// @Accept json
// @Produce json
// @Tags 企业日志
// @Security JWT
// @Param body body controller.CreateExportRequest true "日志类型与筛选条件"
// @Success 200 {object} httpx.Response{data=controller.ExportTaskView}
// @Failure 400 {object} httpx.Response "errCode=ENTERPRISE_LOG_EXPORT_KIND_INVALID / ENTERPRISE_LOG_EXPORT_TOO_LARGE / ENTERPRISE_LOG_DATE_INVALID / ENTERPRISE_LOG_TIME_RANGE_INVALID / ENTERPRISE_LOG_MEMBER_INVALID / ENTERPRISE_LOG_CATEGORY_UNKNOWN / ENTERPRISE_LOG_EVENT_UNKNOWN"
// @Failure 401 {object} httpx.Response "errCode=UNAUTHORIZED"
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN"
// @Router /api/v1/enterprise-logs/exports [post]
func (e *EnterpriseLogController) CreateExport(c *gin.Context) {
	tenantID, ok := resolveTenant(c)
	if !ok {
		return
	}
	req := new(model.CreateExportRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	ginctx.TraceStep(c, "create enterprise log export")
	result, err := e.logService.CreateExport(c.Request.Context(), tenantID, *req)
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
// @Tags 企业日志
// @Security JWT
// @Param id path int true "导出任务 ID"
// @Success 200 {object} httpx.Response{data=controller.ExportTaskView}
// @Failure 400 {object} httpx.Response "errCode=VALIDATION"
// @Failure 401 {object} httpx.Response "errCode=UNAUTHORIZED"
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN"
// @Failure 404 {object} httpx.Response "errCode=ENTERPRISE_LOG_EXPORT_NOT_FOUND"
// @Router /api/v1/enterprise-logs/exports/{id} [get]
func (e *EnterpriseLogController) GetExport(c *gin.Context) {
	tenantID, ok := resolveTenant(c)
	if !ok {
		return
	}
	taskID, err := parseIntPathParam(c, "id")
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	result, err := e.logService.GetExport(c.Request.Context(), tenantID, taskID)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, result)
}

// DownloadExport 下载导出文件。
//
// @Summary 下载导出文件
// @Description 在路由级 get（查看权限）之上复核导出权限（enterprise-logs:export）与任务租户归属、有效期后返回 CSV 文件流
// @Produce text/csv
// @Tags 企业日志
// @Security JWT
// @Param id path int true "导出任务 ID"
// @Success 200 {file} file "CSV 文件流"
// @Failure 400 {object} httpx.Response "errCode=VALIDATION"
// @Failure 401 {object} httpx.Response "errCode=UNAUTHORIZED"
// @Failure 403 {object} httpx.Response "errCode=ENTERPRISE_LOG_EXPORT_FORBIDDEN / FORBIDDEN"
// @Failure 404 {object} httpx.Response "errCode=ENTERPRISE_LOG_EXPORT_NOT_FOUND"
// @Failure 409 {object} httpx.Response "errCode=ENTERPRISE_LOG_EXPORT_NOT_READY"
// @Failure 410 {object} httpx.Response "errCode=ENTERPRISE_LOG_EXPORT_EXPIRED"
// @Router /api/v1/enterprise-logs/exports/{id}/download [get]
func (e *EnterpriseLogController) DownloadExport(c *gin.Context) {
	// 下载路径复核导出权限：查看权限可看任务状态但不可取文件
	if !e.canExport(c) {
		httpx.ResponseFailed(c, http.StatusForbidden, enterpriselog.ErrExportForbidden)
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
	ginctx.TraceStep(c, "download enterprise log export")
	file, err := e.logService.ExportFile(c.Request.Context(), tenantID, taskID)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	// RFC 5987：中文文件名经 filename* 编码下发，兼容主流浏览器
	c.Header("Content-Disposition",
		fmt.Sprintf(`attachment; filename*=UTF-8''%s`, url.PathEscape(file.FileName)))
	c.Data(http.StatusOK, file.ContentType, file.Data)
}
