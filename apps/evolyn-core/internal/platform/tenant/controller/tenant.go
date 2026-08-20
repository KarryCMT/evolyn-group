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

// @Summary 租户列表（平台）
// @Description 平台侧查询租户列表
// @Produce json
// @Tags 平台管理
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

// @Summary 开通租户（平台）
// @Description 开通租户并初始化所有者成员身份与基线角色/分组
// @Accept json
// @Produce json
// @Tags 平台管理
// @Security JWT
// @Param tenant body tenantservice.OpenTenantRequest true "open request"
// @Success 200 {object} httpx.Response{data=tenantmodel.Tenant}
// @Failure 409 {object} httpx.Response "AUTH_TENANT_CODE_DUPLICATED·租户编码已存在"
// @Failure 403 {object} httpx.Response "QUOTA_EXCEEDED·配额已用尽"
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

// @Summary 租户详情（平台）
// @Produce json
// @Tags 平台管理
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

// @Summary 更新租户（平台）
// @Description 更新租户名称/套餐/配置/配额
// @Accept json
// @Produce json
// @Tags 平台管理
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

// @Summary 变更租户状态（平台）
// @Description 租户生命周期状态流转：启用 / 冻结 / 注销
// @Accept json
// @Produce json
// @Tags 平台管理
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

// RegisterRoute 注册到平台运营组（FIX-008：装配层传入的即 /api/v1/platform，
// 本控制器同时以 Platform() 标记自身归属平台域，不进入租户中间件链）
func (tc *TenantController) RegisterRoute(api *gin.RouterGroup) {
	api.GET("/tenants", tc.List)
	api.POST("/tenants", tc.Create)
	api.GET("/tenants/:id", tc.Get)
	api.PUT("/tenants/:id", tc.Update)
	api.PUT("/tenants/:id/status", tc.SetStatus)
}

func (tc *TenantController) Name() string {
	return "PlatformTenant"
}

// Platform 平台运营域标记（FIX-008）
func (tc *TenantController) Platform() bool {
	return true
}
