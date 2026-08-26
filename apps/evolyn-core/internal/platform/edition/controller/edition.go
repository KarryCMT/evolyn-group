// Package controller 版本信息域 HTTP 接口：租户侧只读概览挂租户域链
// （Authentication → Tenant → TenantStatus → Authorization），平台运营
// 接口挂平台链（PlatformController 标记）。权限由中间件链执行，Service
// 内不再重复复核资源权限（本域租户侧仅 editions:get 一个读权限）
package controller

import (
	"errors"
	"fmt"
	"net/http"

	"evolyn/internal/contextx"
	platformcontroller "evolyn/internal/platform/controller"
	editionmodel "evolyn/internal/platform/edition/model"
	"evolyn/internal/platform/edition/service"
	"evolyn/internal/platform/ginctx"
	"evolyn/internal/platform/httpx"

	"github.com/gin-gonic/gin"
)

// EditionController 租户侧版本信息（/editions）
type EditionController struct {
	editionService service.EditionService
}

func NewEditionController(editionService service.EditionService) platformcontroller.Controller {
	return &EditionController{editionService: editionService}
}

func (e *EditionController) Name() string {
	return "版本信息"
}

// CurrentEdition / TenantEditionDetail swagger 注解引用的出网类型别名
// （与 service 返回结构一致；本包代码不直接构造，仅为文档解析提供定位）
type (
	CurrentEdition      = editionmodel.CurrentEdition
	TenantEditionDetail = editionmodel.TenantEditionDetail
	GrantableVersion    = editionmodel.GrantableVersion
	GrantRequest        = editionmodel.GrantRequest
)

func (e *EditionController) RegisterRoute(api *gin.RouterGroup) {
	// /editions/current 经 RequestInfo 解析为 resource=editions + verb=get，
	// 与租户管理员基线权限 editions:get 对应（设计 4.2，无需改动授权器）
	api.GET("/editions/current", e.Current)
}

// Current 当前租户版本信息概览。
//
// @Summary 当前版本信息
// @Description 订阅（含到期即时投影）、资源容量（已接入资源含真实用量，未接入返回待计量）与功能权益；普通成员无 editions:get 权限时由鉴权中间件拒绝
// @Produce json
// @Tags 版本信息
// @Security JWT
// @Success 200 {object} httpx.Response{data=controller.CurrentEdition}
// @Failure 401 {object} httpx.Response "errCode=UNAUTHORIZED"
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN"
// @Router /api/v1/editions/current [get]
func (e *EditionController) Current(c *gin.Context) {
	tenantID, ok := contextx.TenantIDFromContext(c.Request.Context())
	if !ok {
		httpx.ResponseFailed(c, http.StatusUnauthorized, fmt.Errorf("tenant context required"))
		return
	}
	// Service 侧再校验成员确属当前租户（防御无成员会话的裸租户上下文）
	if ginctx.GetUser(c) == nil {
		httpx.ResponseFailed(c, http.StatusUnauthorized, fmt.Errorf("member not loaded"))
		return
	}

	result, err := e.editionService.GetCurrent(c.Request.Context(), tenantID)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, result)
}

// responseError 错误统一出口（ADR-008 脱敏）：BizError 按自带状态码出网；
// 非 BizError 一律 500，原文只入日志
func responseError(c *gin.Context, err error) {
	var biz *httpx.BizError
	if errors.As(err, &biz) && biz.HTTP != 0 {
		httpx.ResponseFailed(c, biz.HTTP, err)
		return
	}
	httpx.ResponseFailed(c, http.StatusInternalServerError, err)
}
