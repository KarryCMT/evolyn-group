package controller

import (
	"net/http"

	platformcontroller "evolyn/internal/platform/controller"
	"evolyn/internal/platform/ginctx"
	"evolyn/internal/platform/httpx"
	"evolyn/internal/platform/iam/model"
	"evolyn/internal/platform/iam/service"

	"github.com/gin-gonic/gin"
)

// 响应读模型仅出现在 swag 注释（data=model.MemberAdminScopes），以类型
// 锚定保持 import 合法
var _ = model.MemberAdminScopes{}

// AdminScopesController 当前成员的管理组身份（/auth/admin-scopes）：
// 管理员页入口/菜单与按钮级控制的数据源。独立于 AuthController——
// 只读自查端点，不掺入登录链路的巨型构造
type AdminScopesController struct {
	adminGroupService service.AdminGroupService
}

// errAuthRequired 未登录拒绝（口径同 AuthorizationMiddleware 的稳定码）
var errAuthRequired = httpx.NewBiz(httpx.CodeUnauthorized, "请先登录", http.StatusUnauthorized)

func NewAdminScopesController(adminGroupService service.AdminGroupService) platformcontroller.Controller {
	return &AdminScopesController{adminGroupService: adminGroupService}
}

// @Summary 我的管理组身份
// @Description 返回当前成员的管理组身份聚合：systemAdmin（内置系统管理员组，即租户管理员）与所属自定义管理组清单，供管理后台菜单/页面入口与按钮控制
// @Produce json
// @Tags 管理组
// @Security JWT
// @Success 200 {object} httpx.Response{data=model.MemberAdminScopes}
// @Router /api/v1/auth/admin-scopes [get]
func (a *AdminScopesController) Get(c *gin.Context) {
	member := ginctx.GetUser(c)
	if member == nil || member.ID == 0 {
		httpx.ResponseFailed(c, http.StatusUnauthorized, errAuthRequired)
		return
	}
	scopes, err := a.adminGroupService.ScopesOfMember(c.Request.Context(), member.ID)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusInternalServerError, err)
		return
	}
	httpx.ResponseSuccess(c, scopes)
}

func (a *AdminScopesController) RegisterRoute(api *gin.RouterGroup) {
	api.GET("/auth/admin-scopes", a.Get)
}

func (a *AdminScopesController) Name() string {
	return "AdminScopes"
}
