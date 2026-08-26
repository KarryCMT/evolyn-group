package controller

import (
	"net/http"
	"strconv"

	platformcontroller "evolyn/internal/platform/controller"
	"evolyn/internal/platform/ginctx"
	"evolyn/internal/platform/httpx"
	"evolyn/internal/platform/iam/model"
	"evolyn/internal/platform/iam/service"

	"github.com/gin-gonic/gin"
)

// 响应读模型仅出现在 swag 注释（data=model.MemberProfileView 等），以类型
// 锚定保持 import 合法
var _ = model.MemberProfileView{}

// MemberProfileController 正式成员扩展档案：本人视图（/accounts/me/member-profile，
// 按字段配置裁剪）与管理员视图（/members/:id/profile，全量 + 卡片裁剪）。
// 手机号、邮箱、部门、角色不在此入口变更，分别走账号安全与成员关系接口
type MemberProfileController struct {
	profileService service.MemberProfileService
}

func NewMemberProfileController(profileService service.MemberProfileService) platformcontroller.Controller {
	return &MemberProfileController{profileService: profileService}
}

// updateMyProfileRequest 本人资料更新：只允许 personalEditable 的扩展字段
type updateMyProfileRequest struct {
	Values map[string]string `json:"values"`
}

// @Summary 我的成员资料
// @Description 读取本人在当前租户的成员资料：values 按字段配置的个人可见性裁剪，editableKeys 为可提交的扩展字段集合
// @Produce json
// @Tags 账号
// @Security JWT
// @Success 200 {object} httpx.Response{data=model.MemberProfileView}
// @Router /api/v1/accounts/me/member-profile [get]
func (m *MemberProfileController) GetMy(c *gin.Context) {
	member := ginctx.GetUser(c)
	if member == nil || member.ID == 0 {
		httpx.ResponseFailed(c, http.StatusUnauthorized, nil)
		return
	}
	view, err := m.profileService.GetMyProfile(c.Request.Context(), member.ID)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, view)
}

// @Summary 更新我的成员资料
// @Description 本人更新扩展资料（别名、工号等）；仅接受字段配置中个人可编辑的扩展字段，手机号/邮箱/部门/角色等一律拒绝
// @Accept json
// @Produce json
// @Tags 账号
// @Security JWT
// @Param body body controller.updateMyProfileRequest true "扩展字段值（key → 值）"
// @Success 200 {object} httpx.Response{data=model.MemberProfileView}
// @Failure 400 {object} httpx.Response "errCode=MEMBER_PROFILE_INVALID"
// @Router /api/v1/accounts/me/member-profile [put]
func (m *MemberProfileController) UpdateMy(c *gin.Context) {
	member := ginctx.GetUser(c)
	if member == nil || member.ID == 0 {
		httpx.ResponseFailed(c, http.StatusUnauthorized, nil)
		return
	}
	req := new(updateMyProfileRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	view, err := m.profileService.UpdateMyProfile(c.Request.Context(), member.ID, req.Values)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, view)
}

// @Summary 成员资料（管理员）
// @Description 读取指定成员的完整资料：values 为全量字段值，cardValues 按卡片展示配置服务端裁剪（成员卡片必须消费该视图），fieldConfig 为字段元数据
// @Produce json
// @Tags 成员
// @Security JWT
// @Param id path int true "member id"
// @Success 200 {object} httpx.Response{data=model.MemberProfileAdminView}
// @Router /api/v1/members/{id}/profile [get]
func (m *MemberProfileController) GetMember(c *gin.Context) {
	memberID, ok := m.parseMemberID(c)
	if !ok {
		return
	}
	view, err := m.profileService.GetMemberProfile(c.Request.Context(), memberID)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, view)
}

// @Summary 维护成员资料（管理员）
// @Description 管理员维护成员扩展资料与企业内编号；不受理手机号、邮箱、部门、角色的变更（走成员管理专用接口）
// @Accept json
// @Produce json
// @Tags 成员
// @Security JWT
// @Param id path int true "member id"
// @Param body body service.MemberProfileUpdateRequest true "编号与扩展字段值"
// @Success 200 {object} httpx.Response{data=model.MemberProfileAdminView}
// @Failure 400 {object} httpx.Response "errCode=MEMBER_PROFILE_INVALID"
// @Router /api/v1/members/{id}/profile [put]
func (m *MemberProfileController) UpdateMember(c *gin.Context) {
	memberID, ok := m.parseMemberID(c)
	if !ok {
		return
	}
	req := new(service.MemberProfileUpdateRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	view, err := m.profileService.UpdateMemberProfile(c.Request.Context(), memberID, req)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, view)
}

// parseMemberID 解析并校验路径中的成员 ID
func (m *MemberProfileController) parseMemberID(c *gin.Context) (uint, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		httpx.ResponseFailed(c, http.StatusBadRequest, nil)
		return 0, false
	}
	return uint(id), true
}

func (m *MemberProfileController) RegisterRoute(api *gin.RouterGroup) {
	api.GET("/accounts/me/member-profile", m.GetMy)
	api.PUT("/accounts/me/member-profile", m.UpdateMy)
	api.GET("/members/:id/profile", m.GetMember)
	api.PUT("/members/:id/profile", m.UpdateMember)
}

func (m *MemberProfileController) Name() string {
	return "MemberProfile"
}
