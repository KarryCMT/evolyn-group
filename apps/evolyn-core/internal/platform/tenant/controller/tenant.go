package controller

import (
	"net/http"

	platformcontroller "evolyn/internal/platform/controller"
	"evolyn/internal/platform/httpx"
	tenantmodel "evolyn/internal/platform/tenant/model"
	tenantservice "evolyn/internal/platform/tenant/service"

	"github.com/gin-gonic/gin"
)

// TenantController 平台运营域（/platform/tenants，P3-1）：
// 管租户的开通/配置/冻结/注销（26.6 后台分离），仅平台管理员可用
type TenantController struct {
	tenantService tenantservice.TenantService
}

func NewTenantController(tenantService tenantservice.TenantService) platformcontroller.Controller {
	return &TenantController{
		tenantService: tenantService,
	}
}

// @Summary List tenants (platform)
// @Description Platform-side tenant list
// @Produce json
// @Tags platform
// @Security JWT
// @Success 200 {object} httpx.Response{data=[]tenantmodel.Tenant}
// @Router /api/v1/platform/tenants [get]
func (tc *TenantController) List(c *gin.Context) {
	tenants, err := tc.tenantService.List(c.Request.Context())
	if err != nil {
		httpx.ResponseFailed(c, http.StatusInternalServerError, err)
		return
	}
	httpx.ResponseSuccess(c, tenants)
}

// @Summary Open tenant (platform)
// @Description Open a tenant with owner membership and baseline roles/groups
// @Accept json
// @Produce json
// @Tags platform
// @Security JWT
// @Param tenant body tenantservice.OpenTenantRequest true "open request"
// @Success 200 {object} httpx.Response{data=tenantmodel.Tenant}
// @Router /api/v1/platform/tenants [post]
func (tc *TenantController) Create(c *gin.Context) {
	req := new(tenantservice.OpenTenantRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	tenant, err := tc.tenantService.Open(c.Request.Context(), req)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, tenant)
}

// @Summary Get tenant (platform)
// @Produce json
// @Tags platform
// @Security JWT
// @Param id path int true "tenant id"
// @Success 200 {object} httpx.Response{data=tenantmodel.Tenant}
// @Router /api/v1/platform/tenants/{id} [get]
func (tc *TenantController) Get(c *gin.Context) {
	tenant, err := tc.tenantService.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, tenant)
}

// @Summary Update tenant (platform)
// @Description Update name/plan/config/quotas
// @Accept json
// @Produce json
// @Tags platform
// @Security JWT
// @Param id path int true "tenant id"
// @Param tenant body tenantmodel.Tenant true "tenant fields"
// @Success 200 {object} httpx.Response{data=tenantmodel.Tenant}
// @Router /api/v1/platform/tenants/{id} [put]
func (tc *TenantController) Update(c *gin.Context) {
	tenant := new(tenantmodel.Tenant)
	if err := c.BindJSON(tenant); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	tenant, err := tc.tenantService.Update(c.Request.Context(), c.Param("id"), tenant)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, tenant)
}

// statusRequest 生命周期流转请求
type statusRequest struct {
	Status string `json:"status" binding:"required"`
}

// @Summary Set tenant status (platform)
// @Description Lifecycle transition: active / frozen / deleted
// @Accept json
// @Produce json
// @Tags platform
// @Security JWT
// @Param id path int true "tenant id"
// @Param body body controller.statusRequest true "status"
// @Success 200 {object} httpx.Response
// @Router /api/v1/platform/tenants/{id}/status [put]
func (tc *TenantController) SetStatus(c *gin.Context) {
	req := new(statusRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	if err := tc.tenantService.SetStatus(c.Request.Context(), c.Param("id"), req.Status); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, nil)
}

func (tc *TenantController) RegisterRoute(api *gin.RouterGroup) {
	platform := api.Group("/platform")
	platform.GET("/tenants", tc.List)
	platform.POST("/tenants", tc.Create)
	platform.GET("/tenants/:id", tc.Get)
	platform.PUT("/tenants/:id", tc.Update)
	platform.PUT("/tenants/:id/status", tc.SetStatus)
}

func (tc *TenantController) Name() string {
	return "PlatformTenant"
}
