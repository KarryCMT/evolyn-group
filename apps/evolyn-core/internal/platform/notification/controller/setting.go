package controller

import (
	"fmt"
	"net/http"
	"strconv"

	"evolyn/internal/contextx"
	platformcontroller "evolyn/internal/platform/controller"
	"evolyn/internal/platform/httpx"
	"evolyn/internal/platform/notification/model"
	"evolyn/internal/platform/notification/service"

	"github.com/gin-gonic/gin"
)

// SettingController 租户通知设置（/notification-settings）：全部挂租户域链，
// 资源权限 notification-settings（仅授租户管理员，不经管理组范围回落放行
// ——当前模型是全租户配置且包含外部联系人隐私数据）。GET 解析为 list/get、
// PATCH 为 patch、POST 为 create、DELETE 为 delete。
type SettingController struct {
	settingService service.SettingService
}

// NewSettingController 通知设置控制器工厂
func NewSettingController(settingService service.SettingService) platformcontroller.Controller {
	return &SettingController{settingService: settingService}
}

func (s *SettingController) Name() string {
	return "通知设置"
}

// swagger 出网类型别名（本包不直接构造，仅为文档解析提供定位）
type (
	SettingAggregateView         = model.SettingAggregateView
	SettingCategoryView          = model.SettingCategoryView
	SettingEventView             = model.SettingEventView
	RecipientView                = model.RecipientView
	ChannelCapabilityView        = model.ChannelCapabilityView
	PatchPreferenceRequest       = model.PatchPreferenceRequest
	PatchPreferenceResponse      = model.PatchPreferenceResponse
	CustomRecipientView          = model.CustomRecipientView
	CreateCustomRecipientRequest = model.CreateCustomRecipientRequest
)

func (s *SettingController) RegisterRoute(api *gin.RouterGroup) {
	api.GET("/notification-settings", s.GetAggregate)
	api.PATCH("/notification-settings/preferences/:eventCode", s.PatchPreference)
	api.GET("/notification-settings/recipients", s.ListRecipients)
	api.POST("/notification-settings/recipients", s.CreateRecipient)
	api.DELETE("/notification-settings/recipients/:id", s.DeleteRecipient)
}

// resolveSettingTenant 解析租户上下文：设置是租户级聚合，无成员维度入参
func resolveSettingTenant(c *gin.Context) (uint, bool) {
	tenantID, ok := contextx.TenantIDFromContext(c.Request.Context())
	if !ok {
		httpx.ResponseFailed(c, http.StatusUnauthorized, fmt.Errorf("tenant context required"))
		return 0, false
	}
	return tenantID, true
}

// GetAggregate 通知设置聚合。
//
// @Summary 查询通知设置
// @Description 返回分类/事件目录、租户有效偏好（无覆盖行投影注册表默认）、渠道能力与聚合 revision；云币/短信额度未接入时 smsBudget 为 null，前端隐藏数值摘要
// @Produce json
// @Tags 通知设置
// @Security JWT
// @Success 200 {object} httpx.Response{data=controller.SettingAggregateView}
// @Failure 401 {object} httpx.Response "errCode=UNAUTHORIZED"
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN"
// @Router /api/v1/notification-settings [get]
func (s *SettingController) GetAggregate(c *gin.Context) {
	tenantID, ok := resolveSettingTenant(c)
	if !ok {
		return
	}
	result, err := s.settingService.GetAggregate(c.Request.Context(), tenantID)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, result)
}

// PatchPreference 更新事件偏好。
//
// @Summary 更新事件通知偏好
// @Description channels 部分更新（缺省键保持不变）；recipients 一旦出现即全量替换该事件接收规则，缺省表示不修改；请求携带聚合 revision，过期返回 409
// @Accept json
// @Produce json
// @Tags 通知设置
// @Security JWT
// @Param eventCode path string true "事件码"
// @Param body body controller.PatchPreferenceRequest true "聚合 revision、渠道部分更新与接收规则全量替换"
// @Success 200 {object} httpx.Response{data=controller.PatchPreferenceResponse}
// @Failure 400 {object} httpx.Response "errCode=NOTIFICATION_EVENT_UNKNOWN / NOTIFICATION_CHANNEL_NOT_SUPPORTED / NOTIFICATION_CHANNEL_REQUIRED / NOTIFICATION_RECIPIENT_INVALID / NOTIFICATION_RECIPIENT_NOT_FOUND"
// @Failure 401 {object} httpx.Response "errCode=UNAUTHORIZED"
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN"
// @Failure 409 {object} httpx.Response "errCode=NOTIFICATION_SETTINGS_CONFLICT / NOTIFICATION_CHANNEL_UNAVAILABLE"
// @Router /api/v1/notification-settings/preferences/{eventCode} [patch]
func (s *SettingController) PatchPreference(c *gin.Context) {
	tenantID, ok := resolveSettingTenant(c)
	if !ok {
		return
	}
	req := new(model.PatchPreferenceRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	result, err := s.settingService.PatchPreference(c.Request.Context(), tenantID, c.Param("eventCode"), *req)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, result)
}

// ListRecipients 自定义提醒对象列表。
//
// @Summary 查询自定义提醒对象
// @Description 租户提醒对象池（姓名/手机/邮箱）；完整联系方式仅通知设置管理员可达
// @Produce json
// @Tags 通知设置
// @Security JWT
// @Success 200 {object} httpx.Response{data=[]controller.CustomRecipientView}
// @Failure 401 {object} httpx.Response "errCode=UNAUTHORIZED"
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN"
// @Router /api/v1/notification-settings/recipients [get]
func (s *SettingController) ListRecipients(c *gin.Context) {
	tenantID, ok := resolveSettingTenant(c)
	if !ok {
		return
	}
	result, err := s.settingService.ListRecipients(c.Request.Context(), tenantID)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, result)
}

// CreateRecipient 新增自定义提醒对象。
//
// @Summary 新增自定义提醒对象
// @Description 手机/邮箱至少一项必填（后端校验）；请求携带聚合 revision 做乐观锁；租户上限默认 200，超限返回 403
// @Accept json
// @Produce json
// @Tags 通知设置
// @Security JWT
// @Param body body controller.CreateCustomRecipientRequest true "聚合 revision、姓名、手机与邮箱"
// @Success 200 {object} httpx.Response{data=controller.CustomRecipientView}
// @Failure 400 {object} httpx.Response "errCode=NOTIFICATION_RECIPIENT_INVALID"
// @Failure 401 {object} httpx.Response "errCode=UNAUTHORIZED"
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN / NOTIFICATION_RECIPIENT_LIMIT_EXCEEDED"
// @Failure 409 {object} httpx.Response "errCode=NOTIFICATION_SETTINGS_CONFLICT"
// @Router /api/v1/notification-settings/recipients [post]
func (s *SettingController) CreateRecipient(c *gin.Context) {
	tenantID, ok := resolveSettingTenant(c)
	if !ok {
		return
	}
	req := new(model.CreateCustomRecipientRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	result, err := s.settingService.CreateRecipient(c.Request.Context(), tenantID, *req)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, result)
}

// DeleteRecipient 删除自定义提醒对象。
//
// @Summary 删除自定义提醒对象
// @Description 仅可删除未被事件偏好引用的对象（在用返回 409 与 usedByEventCodes，不级联修改投递范围）；软删除保留关联历史
// @Produce json
// @Tags 通知设置
// @Security JWT
// @Param id path int true "提醒对象 ID"
// @Param revision query int true "聚合 revision（乐观锁口令）"
// @Success 200 {object} httpx.Response "success"
// @Failure 400 {object} httpx.Response "errCode=VALIDATION"
// @Failure 401 {object} httpx.Response "errCode=UNAUTHORIZED"
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN"
// @Failure 404 {object} httpx.Response "errCode=NOTIFICATION_RECIPIENT_NOT_FOUND"
// @Failure 409 {object} httpx.Response "errCode=NOTIFICATION_SETTINGS_CONFLICT / NOTIFICATION_RECIPIENT_IN_USE"
// @Router /api/v1/notification-settings/recipients/{id} [delete]
func (s *SettingController) DeleteRecipient(c *gin.Context) {
	tenantID, ok := resolveSettingTenant(c)
	if !ok {
		return
	}
	recipientID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest,
			httpx.NewBiz(httpx.CodeValidation, "提醒对象 ID 必须为数字", http.StatusBadRequest))
		return
	}
	revision, err := strconv.ParseInt(c.Query("revision"), 10, 64)
	if err != nil || revision <= 0 {
		httpx.ResponseFailed(c, http.StatusBadRequest,
			httpx.NewBiz(httpx.CodeValidation, "revision 必须为正整数", http.StatusBadRequest))
		return
	}
	if err = s.settingService.DeleteRecipient(c.Request.Context(), tenantID, uint(recipientID), revision); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, nil)
}
