package controller

import (
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

// UserController 成员管理（/users 路径保留，语义为租户成员；ADR-006）
type UserController struct {
	userService service.UserService
}

func NewUserController(userService service.UserService) platformcontroller.Controller {
	return &UserController{
		userService: userService,
	}
}

// @Summary List members
// @Description List tenant members (current tenant)
// @Produce json
// @Tags user
// @Security JWT
// @Success 200 {object} httpx.Response{data=model.Users}
// @Router /api/v1/users [get]
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
// @Router /api/v1/users/{id} [get]
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
// @Router /api/v1/users/{id} [put]
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
// @Router /api/v1/users/{id} [delete]
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
// @Router /api/v1/users/{id}/groups [get]
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
// @Router /api/v1/users/{id}/roles/{rid} [post]
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
// @Router /api/v1/users/{id}/roles/{rid} [delete]
func (u *UserController) DelRole(c *gin.Context) {
	if err := u.userService.DelRole(c.Request.Context(), c.Param("id"), c.Param("rid")); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	httpx.ResponseSuccess(c, nil)
}

func (u *UserController) RegisterRoute(api *gin.RouterGroup) {
	api.GET("/users", u.List)
	api.GET("/users/:id", u.Get)
	api.PUT("/users/:id", u.Update)
	api.DELETE("/users/:id", u.Delete)
	api.GET("/users/:id/groups", u.GetGroups)
	api.POST("/users/:id/roles/:rid", u.AddRole)
	api.DELETE("/users/:id/roles/:rid", u.DelRole)
}

func (u *UserController) Name() string {
	return "User"
}
