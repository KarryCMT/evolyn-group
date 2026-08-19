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

// @Summary List members
// @Description List tenant members (current tenant)
// @Produce json
// @Tags user
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

// @Summary Get member
// @Description Get tenant member by id
// @Produce json
// @Tags user
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

// @Summary Update member
// @Description Update member profile (nickname, in-tenant)
// @Accept json
// @Produce json
// @Tags user
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

// @Summary Delete member
// @Description Delete tenant member
// @Produce json
// @Tags user
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

// @Summary Get groups
// @Description Get member groups
// @Produce json
// @Tags group
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

// @Summary Add role
// @Description Add role to member
// @Produce json
// @Tags user
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

// @Summary Delete role
// @Description delete role from member
// @Produce json
// @Tags user
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

// @Summary Add member
// @Description Add an existing account to current tenant (with quota check and optional department/role binding)
// @Accept json
// @Produce json
// @Tags user
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

// @Summary Set member departments
// @Description Replace member department memberships (multi-department)
// @Accept json
// @Produce json
// @Tags user
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
