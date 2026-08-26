package controller

import (
	"net/http"

	platformcontroller "evolyn/internal/platform/controller"
	"evolyn/internal/platform/ginctx"
	"evolyn/internal/platform/httpx"
	tenantservice "evolyn/internal/platform/tenant/service"

	"github.com/gin-gonic/gin"
)

// TenantProfileController 租户自助资料：与平台运营面 TenantController 隔离，
// 仅允许当前会话所属租户修改其组织根节点名称。
type TenantProfileController struct {
	tenantService tenantservice.TenantService
}

func NewTenantProfileController(tenantService tenantservice.TenantService) platformcontroller.Controller {
	return &TenantProfileController{tenantService: tenantService}
}

// updateTenantProfileRequest 当前仅暴露名称，避免客户端触碰套餐、配额等运营字段。
type updateTenantProfileRequest struct {
	Name string `json:"name" binding:"required"`
}

// @Summary 修改当前租户名称
// @Description 更新当前登录租户的组织根节点名称，仅租户管理员可操作
// @Accept json
// @Produce json
// @Tags 租户管理
// @Security JWT
// @Param body body controller.updateTenantProfileRequest true "租户名称"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response "TENANT_NAME_INVALID·租户名称格式不正确"
// @Router /api/v1/tenant/profile [put]
func (tc *TenantProfileController) UpdateProfile(c *gin.Context) {
	req := new(updateTenantProfileRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	tenant, err := tc.tenantService.UpdateMyName(c.Request.Context(), ginctx.GetTenant(c), req.Name)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, tenant)
}

func (tc *TenantProfileController) RegisterRoute(api *gin.RouterGroup) {
	api.PUT("/tenant/profile", tc.UpdateProfile)
}

func (tc *TenantProfileController) Name() string {
	return "TenantProfile"
}
