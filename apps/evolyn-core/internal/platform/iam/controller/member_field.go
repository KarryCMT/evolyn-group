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

// 响应读模型仅出现在 swag 注释（data=model.MemberFieldConfigSnapshot），
// 以类型锚定保持 import 合法
var _ = model.MemberFieldConfigSnapshot{}

// MemberFieldController 成员信息管理（/member-field-settings）：字段设置与
// 卡片展示页签共用的租户级字段显示策略，勾选即时保存（无整页保存按钮）
type MemberFieldController struct {
	fieldService service.MemberFieldService
}

func NewMemberFieldController(fieldService service.MemberFieldService) platformcontroller.Controller {
	return &MemberFieldController{fieldService: fieldService}
}

// @Summary 查询成员字段配置
// @Description 返回当前租户的完整字段配置快照（预置字段恒完整）；首次读取自动补齐默认配置，字段设置页与卡片展示页共用
// @Produce json
// @Tags 成员信息管理
// @Security JWT
// @Success 200 {object} httpx.Response{data=model.MemberFieldConfigSnapshot}
// @Router /api/v1/member-field-settings [get]
func (m *MemberFieldController) Get(c *gin.Context) {
	snapshot, err := m.fieldService.GetSnapshot(c.Request.Context())
	if err != nil {
		httpx.ResponseFailed(c, http.StatusInternalServerError, err)
		return
	}
	httpx.ResponseSuccess(c, snapshot)
}

// @Summary 即时更新一个成员字段配置
// @Description 只提交本次变更的开关与页面读取到的版本号；关闭可见时自动联动关闭可编辑；响应返回最新整页快照供前端覆盖本地状态
// @Accept json
// @Produce json
// @Tags 成员信息管理
// @Security JWT
// @Param fieldKey path string true "预置字段 key（如 mobile）"
// @Param body body service.MemberFieldSettingUpdateRequest true "本次变更的配置值与版本号"
// @Success 200 {object} httpx.Response{data=model.MemberFieldConfigSnapshot}
// @Failure 400 {object} httpx.Response "errCode=MEMBER_FIELD_CONFIG_INVALID"
// @Failure 403 {object} httpx.Response "errCode=MEMBER_FIELD_LOCKED"
// @Failure 404 {object} httpx.Response "errCode=MEMBER_FIELD_NOT_FOUND"
// @Failure 409 {object} httpx.Response "errCode=MEMBER_FIELD_CONFIG_CONFLICT"
// @Router /api/v1/member-field-settings/{fieldKey} [patch]
func (m *MemberFieldController) Update(c *gin.Context) {
	req := new(service.MemberFieldSettingUpdateRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	ginctx.TraceStep(c, "update member field setting")
	snapshot, err := m.fieldService.UpdateField(c.Request.Context(), c.Param("fieldKey"), req)
	if err != nil {
		// 状态码由 BizError 自动映射：404 不存在/403 锁定/400 联动冲突/409 版本冲突
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, snapshot)
}

func (m *MemberFieldController) RegisterRoute(api *gin.RouterGroup) {
	api.GET("/member-field-settings", m.Get)
	api.PATCH("/member-field-settings/:fieldKey", m.Update)
}

func (m *MemberFieldController) Name() string {
	return "MemberFieldSetting"
}
