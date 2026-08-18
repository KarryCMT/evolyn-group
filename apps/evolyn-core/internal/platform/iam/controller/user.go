package controller

import (
	"evolyn/internal/platform/ginctx"
	"evolyn/internal/platform/httpx"
	"net/http"
	"strconv"

	platformcontroller "evolyn/internal/platform/controller"
	"evolyn/internal/platform/iam/authorization"
	"evolyn/internal/platform/iam/model"
	"evolyn/internal/platform/iam/service"
	"evolyn/internal/utils/trace"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type UserController struct {
	userService service.UserService
}

func NewUserController(userService service.UserService) platformcontroller.Controller {
	return &UserController{
		userService: userService,
	}
}

// @Summary List user
// @Description List user and storage
// @Produce json
// @Tags user
// @Security JWT
// @Success 200 {object} httpx.Response{data=model.Users}
// @Router /api/v1/users [get]
func (u *UserController) List(c *gin.Context) {
	ginctx.TraceStep(c, "start list user")
	users, err := u.userService.List(c.Request.Context())
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	ginctx.TraceStep(c, "list user done")
	httpx.ResponseSuccess(c, users)
}

// @Summary Get user
// @Description Get user and storage
// @Produce json
// @Tags user
// @Security JWT
// @Param id path int true "user id"
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

// @Summary Create user
// @Description Create user and storage
// @Accept json
// @Produce json
// @Tags user
// @Security JWT
// @Param user body model.CreatedUser true "user info"
// @Success 200 {object} httpx.Response{data=model.User}
// @Router /api/v1/users [post]
func (u *UserController) Create(c *gin.Context) {
	createdUser := new(model.CreatedUser)
	if err := c.BindJSON(createdUser); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	user := createdUser.GetUser()
	if err := u.userService.Validate(user); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	u.userService.Default(user)
	ginctx.TraceStep(c, "start create user", trace.Field{"user", user.Name})
	defer ginctx.TraceStep(c, "create user done", trace.Field{"user", user.Name})
	user, err := u.userService.Create(c.Request.Context(), user)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusInternalServerError, err)
	}

	httpx.ResponseSuccess(c, user)
}

// @Summary Update user
// @Description Update user and storage
// @Accept json
// @Produce json
// @Tags user
// @Security JWT
// @Param user body model.UpdatedUser true "user info"
// @Param id   path      int  true  "user id"
// @Success 200 {object} httpx.Response{data=model.User}
// @Router /api/v1/users/{id} [put]
func (u *UserController) Update(c *gin.Context) {
	user := ginctx.GetUser(c)
	if user == nil || (strconv.Itoa(int(user.ID)) != c.Param("id") && !authorization.IsClusterAdmin(user)) {
		httpx.ResponseFailed(c, http.StatusForbidden, nil)
		return
	}

	new := new(model.UpdatedUser)
	if err := c.BindJSON(new); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	logrus.Infof("get update user: %#v", new.Name)

	ginctx.TraceStep(c, "start update user", trace.Field{"user", new.Name})
	defer ginctx.TraceStep(c, "update user done", trace.Field{"user", new.Name})

	user, err := u.userService.Update(c.Request.Context(), c.Param("id"), new.GetUser())
	if err != nil {
		httpx.ResponseFailed(c, http.StatusInternalServerError, err)
		return
	}

	httpx.ResponseSuccess(c, user)
}

// @Summary Delete user
// @Description Delete user and storage
// @Produce json
// @Tags user
// @Security JWT
// @Param id path int true "user id"
// @Success 200 {object} httpx.Response
// @Router /api/v1/users/{id} [delete]
func (u *UserController) Delete(c *gin.Context) {
	user := ginctx.GetUser(c)
	if user == nil || (strconv.Itoa(int(user.ID)) != c.Param("id") && !authorization.IsClusterAdmin(user)) {
		httpx.ResponseFailed(c, http.StatusForbidden, nil)
		return
	}

	if err := u.userService.Delete(c.Request.Context(), c.Param("id")); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	httpx.ResponseSuccess(c, nil)
}

// @Summary Get groups
// @Description Get groups
// @Produce json
// @Tags group
// @Security JWT
// @Param id path int true "user id"
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
// @Description Add role to user
// @Produce json
// @Tags user
// @Security JWT
// @Param id path int true "user id"
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
// @Description delete role from user
// @Produce json
// @Tags user
// @Security JWT
// @Param id path int true "user id"
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
	api.POST("/users", u.Create)
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
