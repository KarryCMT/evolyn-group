// Package controller 产品中心域 HTTP 接口（/tenant-products）：全部挂租户
// 域链（Authentication → Tenant → TenantStatus → Authorization）。资源级
// 权限由中间件链执行：GET /tenant-products 解析为 list、两条 PUT 子资源
// 路径解析为 update，与租户管理员基线规则 tenant-products:view/update
// 对应（RequestInfo 按路径推导，无需改动授权器）；Service 内不再重复
// 复核资源权限，仅校验数据归属
package controller

import (
	"fmt"
	"net/http"

	"evolyn/internal/contextx"
	platformcontroller "evolyn/internal/platform/controller"
	"evolyn/internal/platform/ginctx"
	"evolyn/internal/platform/httpx"
	"evolyn/internal/platform/tenantproduct/model"
	"evolyn/internal/platform/tenantproduct/service"

	"github.com/gin-gonic/gin"
)

// TenantProductController 租户产品中心（/tenant-products）
type TenantProductController struct {
	productService service.TenantProductService
}

func NewTenantProductController(productService service.TenantProductService) platformcontroller.Controller {
	return &TenantProductController{productService: productService}
}

func (t *TenantProductController) Name() string {
	return "产品中心"
}

// TenantProduct 相关 swagger 出网/入网类型别名（与 service/model 定义一致，
// 本包不直接构造，仅为文档解析提供定位）
type (
	ProductCenterView        = model.ProductCenterView
	ProductCard              = model.ProductCard
	UpdateEnabledRequest     = model.UpdateEnabledRequest
	UpdateAccessScopeRequest = model.UpdateAccessScopeRequest
)

func (t *TenantProductController) RegisterRoute(api *gin.RouterGroup) {
	api.GET("/tenant-products", t.List)
	api.PUT("/tenant-products/:code/enabled", t.SetEnabled)
	api.PUT("/tenant-products/:code/access-scope", t.UpdateAccessScope)
}

// resolveContext 解析租户与成员上下文：租户上下文缺失或会话未加载成员
// 均按 401 拒绝（与版本信息域控制器同口径，防御无成员会话的裸租户上下文）
func resolveContext(c *gin.Context) (uint, bool) {
	tenantID, ok := contextx.TenantIDFromContext(c.Request.Context())
	if !ok {
		httpx.ResponseFailed(c, http.StatusUnauthorized, fmt.Errorf("tenant context required"))
		return 0, false
	}
	if ginctx.GetUser(c) == nil {
		httpx.ResponseFailed(c, http.StatusUnauthorized, fmt.Errorf("member not loaded"))
		return 0, false
	}
	return tenantID, true
}

// List 产品中心卡片列表。
//
// @Summary 查询产品中心
// @Description 返回当前租户可管理的产品卡片：启用状态、当前版本（来自版本信息域）、可用范围与真实有效成员数；平台目录按展示顺序返回全部产品
// @Produce json
// @Tags 产品中心
// @Security JWT
// @Success 200 {object} httpx.Response{data=controller.ProductCenterView}
// @Failure 401 {object} httpx.Response "errCode=UNAUTHORIZED"
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN"
// @Router /api/v1/tenant-products [get]
func (t *TenantProductController) List(c *gin.Context) {
	tenantID, ok := resolveContext(c)
	if !ok {
		return
	}
	result, err := t.productService.List(c.Request.Context(), tenantID)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusInternalServerError, err)
		return
	}
	httpx.ResponseSuccess(c, result)
}

// SetEnabled 启用或停用产品。
//
// @Summary 启用或停用产品
// @Description 锁定并校验配置版本号（revision）后更新启停状态并递增版本；响应返回最新产品卡片。运行时产品访问以服务端判定为准，前端隐藏入口仅为体验优化
// @Accept json
// @Produce json
// @Tags 产品中心
// @Security JWT
// @Param code path string true "产品稳定机器码（如 lingyanyun）"
// @Param body body controller.UpdateEnabledRequest true "启停状态与读取到的版本号"
// @Success 200 {object} httpx.Response{data=controller.ProductCard}
// @Failure 400 {object} httpx.Response "errCode=VALIDATION"
// @Failure 401 {object} httpx.Response "errCode=UNAUTHORIZED"
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN"
// @Failure 404 {object} httpx.Response "errCode=TENANT_PRODUCT_NOT_FOUND"
// @Failure 409 {object} httpx.Response "errCode=TENANT_PRODUCT_REVISION_CONFLICT"
// @Router /api/v1/tenant-products/{code}/enabled [put]
func (t *TenantProductController) SetEnabled(c *gin.Context) {
	tenantID, ok := resolveContext(c)
	if !ok {
		return
	}
	req := new(model.UpdateEnabledRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	ginctx.TraceStep(c, "set tenant product enabled")
	card, err := t.productService.SetEnabled(c.Request.Context(), tenantID, c.Param("code"), req)
	if err != nil {
		// 状态码由 BizError 自动映射：404 未初始化 / 409 版本冲突
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, card)
}

// UpdateAccessScope 全量替换产品可用范围。
//
// @Summary 替换产品可用范围
// @Description mode=all 时部门与成员清单必须为空；mode=partial 时两者合计至少一项，且全部 ID 在同一事务内校验当前租户归属与有效性；响应返回最新产品卡片
// @Accept json
// @Produce json
// @Tags 产品中心
// @Security JWT
// @Param code path string true "产品稳定机器码（如 lingyanyun）"
// @Param body body controller.UpdateAccessScopeRequest true "范围模式、部门/成员稳定 ID 清单与版本号"
// @Success 200 {object} httpx.Response{data=controller.ProductCard}
// @Failure 400 {object} httpx.Response "errCode=TENANT_PRODUCT_SCOPE_INVALID / TENANT_PRODUCT_SCOPE_EMPTY / TENANT_PRODUCT_MEMBER_INVALID / TENANT_PRODUCT_DEPARTMENT_INVALID"
// @Failure 401 {object} httpx.Response "errCode=UNAUTHORIZED"
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN"
// @Failure 404 {object} httpx.Response "errCode=TENANT_PRODUCT_NOT_FOUND"
// @Failure 409 {object} httpx.Response "errCode=TENANT_PRODUCT_REVISION_CONFLICT"
// @Router /api/v1/tenant-products/{code}/access-scope [put]
func (t *TenantProductController) UpdateAccessScope(c *gin.Context) {
	tenantID, ok := resolveContext(c)
	if !ok {
		return
	}
	req := new(model.UpdateAccessScopeRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	ginctx.TraceStep(c, "update tenant product access scope")
	card, err := t.productService.UpdateAccessScope(c.Request.Context(), tenantID, c.Param("code"), req)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, card)
}
