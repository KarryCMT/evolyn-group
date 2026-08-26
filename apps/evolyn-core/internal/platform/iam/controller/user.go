package controller

import (
	"io"
	"net/http"
	"strconv"

	platformcontroller "evolyn/internal/platform/controller"
	"evolyn/internal/platform/ginctx"
	"evolyn/internal/platform/httpx"
	"evolyn/internal/platform/iam/authorization"
	"evolyn/internal/platform/iam/model"
	"evolyn/internal/platform/iam/service"
	"evolyn/internal/utils/trace"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// UserController 成员管理（/members，语义为租户成员；ADR-006）
type UserController struct {
	userService       service.UserService
	departmentService service.DepartmentService
	invitationService service.MemberInvitationService
}

func NewUserController(userService service.UserService, departmentService service.DepartmentService, invitationService service.MemberInvitationService) platformcontroller.Controller {
	return &UserController{
		userService:       userService,
		departmentService: departmentService,
		invitationService: invitationService,
	}
}

// @Summary 成员列表
// @Description 分页查询当前租户的成员；可按部门、成员状态及关键词筛选。全部成员默认不含离职成员，离职成员传 status=resigned 查询
// @Produce json
// @Tags 成员
// @Security JWT
// @Param departmentId query int false "部门 ID"
// @Param status query string false "成员状态：active/disabled/resigned，默认 active 与 disabled"
// @Param keyword query string false "姓名、手机号或邮箱关键词"
// @Param page query int false "页码，默认 1"
// @Param pageSize query int false "每页条数，默认 20，上限 100"
// @Success 200 {object} httpx.Response{data=model.MemberPage}
// @Failure 400 {object} httpx.Response "errCode=MEMBER_STATUS_INVALID"
// @Failure 403 {object} httpx.Response "errCode=TENANT_CREATOR_STATUS_IMMUTABLE"
// @Router /api/v1/members [get]
func (u *UserController) List(c *gin.Context) {
	ginctx.TraceStep(c, "start list members")
	departmentID := queryInt(c, "departmentId")
	query := model.MemberListQuery{
		Status:   c.Query("status"),
		Keyword:  c.Query("keyword"),
		Page:     queryInt(c, "page"),
		PageSize: queryInt(c, "pageSize"),
	}
	if departmentID > 0 {
		query.DepartmentID = uint(departmentID)
	}
	page, err := u.userService.ListPage(c.Request.Context(), query)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	ginctx.TraceStep(c, "list members done")
	httpx.ResponseSuccess(c, page)
}

// @Summary 成员详情
// @Description 按 ID 查询租户成员详情
// @Produce json
// @Tags 成员
// @Security JWT
// @Param id path int true "member id"
// @Success 200 {object} httpx.Response{data=model.User}
// @Router /api/v1/members/{id} [get]
func (u *UserController) Get(c *gin.Context) {
	user, err := u.userService.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, user)
}

// @Summary 更新成员
// @Description 更新成员的租户内资料（如昵称）
// @Accept json
// @Produce json
// @Tags 成员
// @Security JWT
// @Param member body model.UpdatedMember true "member info"
// @Param id   path      int  true  "member id"
// @Success 200 {object} httpx.Response{data=model.User}
// @Router /api/v1/members/{id} [put]
func (u *UserController) Update(c *gin.Context) {
	user := ginctx.GetUser(c)
	if user == nil || (strconv.Itoa(int(user.ID)) != c.Param("id") && !authorization.IsClusterAdmin(user)) {
		httpx.ResponseFailed(c, http.StatusForbidden, nil)
		return
	}

	new := new(model.UpdatedMember)
	if err := c.BindJSON(new); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	ginctx.TraceStep(c, "start update member", trace.Field{"member", new.Nickname})
	defer ginctx.TraceStep(c, "update member done", trace.Field{"member", new.Nickname})

	user, err := u.userService.Update(c.Request.Context(), c.Param("id"), new.GetMember())
	if err != nil {
		httpx.ResponseFailed(c, http.StatusInternalServerError, err)
		return
	}

	httpx.ResponseSuccess(c, user)
}

// updateMemberStatusRequest 成员在当前租户中的生命周期状态。
type updateMemberStatusRequest struct {
	Status string `json:"status"`
}

// @Summary 更新成员状态
// @Description 更新当前租户内成员状态；转为离职会保留成员历史，以便在离职成员中查询
// @Accept json
// @Produce json
// @Tags 成员
// @Security JWT
// @Param id path int true "member id"
// @Param body body controller.updateMemberStatusRequest true "member status"
// @Success 200 {object} httpx.Response{data=model.User}
// @Failure 400 {object} httpx.Response "errCode=MEMBER_STATUS_INVALID"
// @Router /api/v1/members/{id}/status [put]
func (u *UserController) UpdateStatus(c *gin.Context) {
	req := new(updateMemberStatusRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	member, err := u.userService.UpdateStatus(c.Request.Context(), c.Param("id"), req.Status)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, member)
}

// @Summary 移除成员
// @Description 将成员移出租户
// @Produce json
// @Tags 成员
// @Security JWT
// @Param id path int true "member id"
// @Success 200 {object} httpx.Response
// @Router /api/v1/members/{id} [delete]
func (u *UserController) Delete(c *gin.Context) {
	user := ginctx.GetUser(c)
	if user == nil || (strconv.Itoa(int(user.ID)) != c.Param("id") && !authorization.IsClusterAdmin(user)) {
		httpx.ResponseFailed(c, http.StatusForbidden, nil)
		return
	}

	logrus.Infof("delete member: %s", c.Param("id"))
	if err := u.userService.Delete(c.Request.Context(), c.Param("id")); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	httpx.ResponseSuccess(c, nil)
}

// @Summary 成员所属分组
// @Description 查询成员所属的分组列表
// @Produce json
// @Tags 分组
// @Security JWT
// @Param id path int true "member id"
// @Success 200 {object} httpx.Response
// @Router /api/v1/members/{id}/groups [get]
func (u *UserController) GetGroups(c *gin.Context) {
	groups, err := u.userService.GetGroups(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	httpx.ResponseSuccess(c, groups)
}

// @Summary 成员绑定角色
// @Description 为成员绑定角色
// @Produce json
// @Tags 成员
// @Security JWT
// @Param id path int true "member id"
// @Param rid path int true "role id"
// @Success 200 {object} httpx.Response
// @Router /api/v1/members/{id}/roles/{rid} [post]
func (u *UserController) AddRole(c *gin.Context) {
	if err := u.userService.AddRole(c.Request.Context(), c.Param("id"), c.Param("rid")); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	httpx.ResponseSuccess(c, nil)
}

// @Summary 成员解绑角色
// @Description 解除成员绑定的角色
// @Produce json
// @Tags 成员
// @Security JWT
// @Param id path int true "member id"
// @Param rid path int true "role id"
// @Success 200 {object} httpx.Response
// @Router /api/v1/members/{id}/roles/{rid} [delete]
func (u *UserController) DelRole(c *gin.Context) {
	if err := u.userService.DelRole(c.Request.Context(), c.Param("id"), c.Param("rid")); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	httpx.ResponseSuccess(c, nil)
}

// @Summary 添加成员
// @Description 将已有平台账号加入当前租户（含配额校验，可选绑定部门/角色）
// @Accept json
// @Produce json
// @Tags 成员
// @Security JWT
// @Param member body service.AddMemberRequest true "member to add"
// @Success 200 {object} httpx.Response{data=model.User}
// @Router /api/v1/members [post]
func (u *UserController) AddMember(c *gin.Context) {
	req := new(service.AddMemberRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	member, err := u.userService.AddMember(c.Request.Context(), req)
	if err != nil {
		// 状态码/业务码由 BizError 自动映射（ADR-008）：重复 409/配额 403/跨租户 403
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	httpx.ResponseSuccess(c, member)
}

// @Summary 邀请成员
// @Description 保存手动填写的成员邀请及完整档案，手机号和邮箱至少填写一项
// @Accept json
// @Produce json
// @Tags 成员
// @Security JWT
// @Param body body service.MemberInvitationRequest true "成员邀请信息"
// @Success 200 {object} httpx.Response{data=model.MemberInvitation}
// @Failure 400 {object} httpx.Response "errCode=MEMBER_INVITATION_INVALID|MEMBER_INVITATION_CONTACT_REQUIRED"
// @Router /api/v1/members/invitations [post]
func (u *UserController) CreateInvitation(c *gin.Context) {
	if u.invitationService == nil {
		httpx.ResponseFailed(c, http.StatusNotImplemented, nil)
		return
	}
	req := new(service.MemberInvitationRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	member := ginctx.GetUser(c)
	invitation, err := u.invitationService.Create(c.Request.Context(), member.ID, *req)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, invitation)
}

// @Summary 批量导入邀请成员
// @Description 导入通讯录模板（xlsx，最大 5MB、最多 200 条），已存在的手机号或邮箱不会创建重复邀请
// @Accept multipart/form-data
// @Produce json
// @Tags 成员
// @Security JWT
// @Param file formData file true "通讯录批量导入模板"
// @Success 200 {object} httpx.Response{data=model.MemberInvitationBatchResult}
// @Failure 400 {object} httpx.Response "errCode=MEMBER_INVITATION_IMPORT_FILE_INVALID"
// @Router /api/v1/members/invitations/import [post]
func (u *UserController) ImportInvitations(c *gin.Context) {
	if u.invitationService == nil {
		httpx.ResponseFailed(c, http.StatusNotImplemented, nil)
		return
	}
	file, err := c.FormFile("file")
	if err != nil || file.Size <= 0 || file.Size > 5*1024*1024 {
		httpx.ResponseFailed(c, http.StatusBadRequest, service.ErrMemberInvitationImportFile)
		return
	}
	reader, err := file.Open()
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, service.ErrMemberInvitationImportFile)
		return
	}
	defer reader.Close()
	content, err := io.ReadAll(io.LimitReader(reader, 5*1024*1024+1))
	if err != nil || len(content) > 5*1024*1024 {
		httpx.ResponseFailed(c, http.StatusBadRequest, service.ErrMemberInvitationImportFile)
		return
	}
	member := ginctx.GetUser(c)
	result, err := u.invitationService.Import(c.Request.Context(), member.ID, content)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, result)
}

// @Summary 查询公开邀请链接
// @Description 返回当前租户的公开邀请链接开关与安全令牌；首次查询自动生成未开启链接
// @Produce json
// @Tags 成员
// @Security JWT
// @Success 200 {object} httpx.Response{data=model.TenantPublicInvitationLink}
// @Router /api/v1/members/invitation-link [get]
func (u *UserController) GetPublicInvitationLink(c *gin.Context) {
	if u.invitationService == nil {
		httpx.ResponseFailed(c, http.StatusNotImplemented, nil)
		return
	}
	link, err := u.invitationService.GetPublicLink(c.Request.Context())
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, link)
}

type updatePublicInvitationLinkRequest struct {
	Enabled bool `json:"enabled"`
}

// @Summary 设置公开邀请链接
// @Description 开启或关闭当前租户的公开邀请链接
// @Accept json
// @Produce json
// @Tags 成员
// @Security JWT
// @Param body body controller.updatePublicInvitationLinkRequest true "公开邀请链接设置"
// @Success 200 {object} httpx.Response{data=model.TenantPublicInvitationLink}
// @Router /api/v1/members/invitation-link [put]
func (u *UserController) UpdatePublicInvitationLink(c *gin.Context) {
	if u.invitationService == nil {
		httpx.ResponseFailed(c, http.StatusNotImplemented, nil)
		return
	}
	req := new(updatePublicInvitationLinkRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	member := ginctx.GetUser(c)
	link, err := u.invitationService.UpdatePublicLink(c.Request.Context(), member.ID, req.Enabled)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, link)
}

func (u *UserController) RegisterRoute(api *gin.RouterGroup) {
	api.GET("/members", u.List)
	api.POST("/members", u.AddMember)
	api.POST("/members/invitations", u.CreateInvitation)
	api.POST("/members/invitations/import", u.ImportInvitations)
	api.GET("/members/invitation-link", u.GetPublicInvitationLink)
	api.PUT("/members/invitation-link", u.UpdatePublicInvitationLink)
	api.GET("/members/:id", u.Get)
	api.PUT("/members/:id", u.Update)
	api.PUT("/members/:id/status", u.UpdateStatus)
	api.DELETE("/members/:id", u.Delete)
	api.GET("/members/:id/groups", u.GetGroups)
	api.POST("/members/:id/roles/:rid", u.AddRole)
	api.DELETE("/members/:id/roles/:rid", u.DelRole)
	api.PUT("/members/:id/departments", u.SetDepartments)
}

func (u *UserController) Name() string {
	return "Member"
}

// setDepartmentsRequest 成员部门归属整体替换
type setDepartmentsRequest struct {
	DepartmentIDs []uint `json:"departmentIds"`
}

// @Summary 设置成员部门
// @Description 整体替换成员的部门归属（支持多部门）
// @Accept json
// @Produce json
// @Tags 成员
// @Security JWT
// @Param id path int true "member id"
// @Param body body controller.setDepartmentsRequest true "department ids"
// @Success 200 {object} httpx.Response
// @Router /api/v1/members/{id}/departments [put]
func (u *UserController) SetDepartments(c *gin.Context) {
	req := new(setDepartmentsRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	if err := u.departmentService.SetMemberDepartments(c.Request.Context(), c.Param("id"), req.DepartmentIDs); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, nil)
}
