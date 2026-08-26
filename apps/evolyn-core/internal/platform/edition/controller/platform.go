package controller

import (
	"fmt"
	"net/http"
	"strconv"

	platformcontroller "evolyn/internal/platform/controller"
	editionmodel "evolyn/internal/platform/edition/model"
	"evolyn/internal/platform/edition/service"
	"evolyn/internal/platform/ginctx"
	"evolyn/internal/platform/httpx"

	"github.com/gin-gonic/gin"
)

// PlatformEditionController 平台运营域版本信息（/platform/tenants/:id/edition
// 与 /platform/edition-plan-versions）：实现 PlatformController 标记，
// 挂载到 /api/v1/platform 组（Authentication + PlatformAuthorization，
// cluster-admin 角色判定，无租户上下文，FIX-008 双域隔离）
type PlatformEditionController struct {
	editionService service.EditionService
}

func NewPlatformEditionController(editionService service.EditionService) platformcontroller.Controller {
	return &PlatformEditionController{editionService: editionService}
}

func (p *PlatformEditionController) Name() string {
	return "平台版本管理"
}

// Platform 平台运营域标记（FIX-008）
func (p *PlatformEditionController) Platform() bool { return true }

func (p *PlatformEditionController) RegisterRoute(api *gin.RouterGroup) {
	api.GET("/tenants/:id/edition", p.GetTenantEdition)
	api.PUT("/tenants/:id/edition", p.Grant)
	api.GET("/edition-plan-versions", p.ListGrantableVersions)
}

// tenantIDFromParam 解析路径参数中的租户 ID
func tenantIDFromParam(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		httpx.ResponseFailed(c, http.StatusBadRequest, fmt.Errorf("无效的租户 ID：%s", c.Param("id")))
		return 0, false
	}
	return uint(id), true
}

// operatorAccountID 平台操作者账号（会话 claims；无会话场景为 0，仅记录用）
func operatorAccountID(c *gin.Context) uint {
	if claims := ginctx.GetSession(c); claims != nil {
		return claims.AccountID
	}
	return 0
}

// GetTenantEdition 租户版本详情（平台运营面）。
//
// @Summary 查询租户版本详情
// @Description 当前订阅概览、历史订阅（含运营备注）与特批覆盖记录；仅平台运营管理员可访问
// @Produce json
// @Tags 平台版本管理
// @Security JWT
// @Param id path int true "租户 ID"
// @Success 200 {object} httpx.Response{data=controller.TenantEditionDetail}
// @Failure 400 {object} httpx.Response "路径参数非法"
// @Failure 404 {object} httpx.Response "errCode=EDITION_TENANT_NOT_FOUND"
// @Router /api/v1/platform/tenants/{id}/edition [get]
func (p *PlatformEditionController) GetTenantEdition(c *gin.Context) {
	tenantID, ok := tenantIDFromParam(c)
	if !ok {
		return
	}
	detail, err := p.editionService.GetTenantEdition(c.Request.Context(), tenantID)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, detail)
}

// Grant 人工授予/替换/取消租户订阅。
//
// @Summary 人工授予或取消租户订阅
// @Description action=grant：校验套餐版本与授予规则（试用必须带 endsAt、存储整 GiB）后单事务完成关旧订阅、建新订阅、覆盖替换与 tenants.plan/quotas 兼容投影同步；action=cancel：取消当前订阅并降级免费版。提交后写审计日志
// @Accept json
// @Produce json
// @Tags 平台版本管理
// @Security JWT
// @Param id path int true "租户 ID"
// @Param body body controller.GrantRequest true "授予/取消请求"
// @Success 200 {object} httpx.Response "操作成功"
// @Failure 400 {object} httpx.Response "errCode=EDITION_GRANT_INVALID/EDITION_OVERRIDE_INVALID/EDITION_STORAGE_LIMIT_INVALID"
// @Failure 404 {object} httpx.Response "errCode=EDITION_TENANT_NOT_FOUND/EDITION_PLAN_VERSION_NOT_FOUND/EDITION_SUBSCRIPTION_NOT_FOUND"
// @Failure 409 {object} httpx.Response "errCode=EDITION_PLAN_VERSION_NOT_GRANTABLE"
// @Router /api/v1/platform/tenants/{id}/edition [put]
func (p *PlatformEditionController) Grant(c *gin.Context) {
	tenantID, ok := tenantIDFromParam(c)
	if !ok {
		return
	}
	req := new(editionmodel.GrantRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	if err := p.editionService.Grant(c.Request.Context(), tenantID, operatorAccountID(c), req); err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, nil)
}

// ListGrantableVersions 可授予套餐版本列表。
//
// @Summary 查询可授予的套餐版本
// @Description 已发布、未下架的基础套餐版本（含权益摘要与适用授予方式）；运营界面先读本列表再提交 planVersionId
// @Produce json
// @Tags 平台版本管理
// @Security JWT
// @Param kind query string false "套餐类型，默认 base"
// @Param status query string false "状态过滤，默认 published"
// @Success 200 {object} httpx.Response{data=[]controller.GrantableVersion}
// @Router /api/v1/platform/edition-plan-versions [get]
func (p *PlatformEditionController) ListGrantableVersions(c *gin.Context) {
	versions, err := p.editionService.ListGrantableVersions(c.Request.Context())
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, versions)
}
