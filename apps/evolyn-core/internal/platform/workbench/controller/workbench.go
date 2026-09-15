// Package controller 自定义工作台域 HTTP 接口：/workbench 挂租户域链
// （Authentication → Tenant → TenantStatus → Authorization）。资源级权限由
// 中间件链执行：GET 解析为 list（对应 workbench:view，授全体成员——首页
// 渲染读取全员必需）、PUT 解析为 update（对应 workbench:update，仅授企业
// 管理员，000078 管理员规则签名补授）；数据范围恒为「当前租户」，由
// Service/Repository 显式 tenantID 条件兜底，不接受客户端提交的替代租户。
package controller

import (
	"fmt"
	"net/http"

	"evolyn/internal/contextx"
	platformcontroller "evolyn/internal/platform/controller"
	"evolyn/internal/platform/httpx"
	"evolyn/internal/platform/workbench/model"
	"evolyn/internal/platform/workbench/service"

	"github.com/gin-gonic/gin"
)

// WorkbenchController 企业自定义工作台（/workbench）
type WorkbenchController struct {
	workbenchService service.WorkbenchService
}

// NewWorkbenchController 自定义工作台控制器工厂
func NewWorkbenchController(workbenchService service.WorkbenchService) platformcontroller.Controller {
	return &WorkbenchController{workbenchService: workbenchService}
}

func (w *WorkbenchController) Name() string {
	return "自定义工作台"
}

// swagger 出网类型别名（本包不直接构造，仅为文档解析提供定位）
type (
	WorkbenchView        = model.WorkbenchView
	SaveWorkbenchRequest = model.SaveWorkbenchRequest
)

func (w *WorkbenchController) RegisterRoute(api *gin.RouterGroup) {
	api.GET("/workbench", w.Get)
	api.PUT("/workbench", w.Save)
}

// resolveTenant 解析租户上下文：工作台是租户级资产，只信任 JWT/Gin 上下文
// 的租户标识，不接受客户端传入的替代 tenantId
func resolveTenant(c *gin.Context) (tenantID uint, ok bool) {
	tenantID, valid := contextx.TenantIDFromContext(c.Request.Context())
	if !valid {
		httpx.ResponseFailed(c, http.StatusUnauthorized, fmt.Errorf("tenant context required"))
		return 0, false
	}
	return tenantID, true
}

// Get 查询企业工作台。
//
// @Summary 查询企业自定义工作台
// @Description 返回当前租户的工作台文档（DashboardSchema 单文档 JSON，企业管理员配置、全员共用）与乐观锁 revision；租户开通时已种子默认布局，行缺失时 data 为 null（前端回退默认布局）
// @Produce json
// @Tags 自定义工作台
// @Security JWT
// @Success 200 {object} httpx.Response{data=controller.WorkbenchView}
// @Failure 401 {object} httpx.Response "errCode=UNAUTHORIZED"
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN"
// @Router /api/v1/workbench [get]
func (w *WorkbenchController) Get(c *gin.Context) {
	tenantID, ok := resolveTenant(c)
	if !ok {
		return
	}
	view, err := w.workbenchService.Get(c.Request.Context(), tenantID)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, view)
}

// Save 保存企业工作台。
//
// @Summary 保存企业自定义工作台（仅企业管理员）
// @Description 全量保存当前租户的工作台文档；document 结构经服务端校验器终审（与前端 @evolyn.do/dashboard 校验镜像：version、widgets、卡片类型白名单、坐标约束）。首次保存 revision 传 0（服务端落 1），此后携带上次返回值，不匹配返回 409
// @Accept json
// @Produce json
// @Tags 自定义工作台
// @Security JWT
// @Param body body controller.SaveWorkbenchRequest true "乐观锁 revision 与工作台文档（DashboardSchema JSON）"
// @Success 200 {object} httpx.Response{data=controller.WorkbenchView}
// @Failure 400 {object} httpx.Response "errCode=WORKBENCH_DOCUMENT_INVALID"
// @Failure 401 {object} httpx.Response "errCode=UNAUTHORIZED"
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN（无 workbench:update 权限，仅企业管理员可保存）"
// @Failure 409 {object} httpx.Response "errCode=WORKBENCH_REVISION_CONFLICT"
// @Router /api/v1/workbench [put]
func (w *WorkbenchController) Save(c *gin.Context) {
	tenantID, ok := resolveTenant(c)
	if !ok {
		return
	}
	req := new(model.SaveWorkbenchRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	if len(req.Document) == 0 {
		httpx.ResponseFailed(c, http.StatusBadRequest,
			httpx.NewBiz("WORKBENCH_DOCUMENT_INVALID", "工作台文档不能为空", http.StatusBadRequest))
		return
	}
	view, err := w.workbenchService.Save(c.Request.Context(), tenantID, req)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, view)
}
