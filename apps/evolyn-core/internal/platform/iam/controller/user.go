package controller

import (
	"errors"
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
}

func NewUserController(userService service.UserService, departmentService service.DepartmentService) platformcontroller.Controller {
	return &UserController{
		userService:       userService,
		departmentService: departmentService,
	}
}

// @Summary 成员列表
// @Description 查询当前租户的成员列表
// @Produce json
// @Tags 成员
// @Security JWT
// @Success 200 {object} httpx.Response{data=model.Users}
// @Router /api/v1/members [get]
func (u *UserController) List(c *gin.Context) {
	ginctx.TraceStep(c, "start list members")
	users, err := u.userService.List(c.Request.Context())
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	ginctx.TraceStep(c, "list members done")
	httpx.ResponseSuccess(c, users)
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
		// 参数/重复/配额类错误 400，跨租户绑定拒绝 403
		status := http.StatusBadRequest
		if errors.Is(err, service.ErrCrossTenantBinding) {
			status = http.StatusForbidden
		}
		httpx.ResponseFailed(c, status, err)
		return
	}

	httpx.ResponseSuccess(c, member)
}

func (u *UserController) RegisterRoute(api *gin.RouterGroup) {
	api.GET("/members", u.List)
	api.POST("/members", u.AddMember)
	api.GET("/members/:id", u.Get)
	api.PUT("/members/:id", u.Update)
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
