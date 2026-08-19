package controller

import (
	"evolyn/internal/platform/ginctx"
	"evolyn/internal/platform/httpx"
	"fmt"
	"net/http"
	"strconv"

	platformcontroller "evolyn/internal/platform/controller"
	"evolyn/internal/platform/iam/model"
	"evolyn/internal/platform/iam/service"
	"evolyn/internal/utils/trace"

	"github.com/gin-gonic/gin"
)

type GroupController struct {
	groupService service.GroupService
}

func NewGroupController(groupService service.GroupService) platformcontroller.Controller {
	return &GroupController{
		groupService: groupService,
	}
}

// @Summary 分组列表
// @Description 查询分组列表
// @Produce json
// @Tags 分组
// @Security JWT
// @Success 200 {object} httpx.Response{data=[]model.Group}
// @Router /api/v1/groups [get]
func (g *GroupController) List(c *gin.Context) {
	ginctx.TraceStep(c, "start list group")
	groups, err := g.groupService.List(c.Request.Context())
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	ginctx.TraceStep(c, "list group done")
	httpx.ResponseSuccess(c, groups)
}

// @Summary 分组详情
// @Description 查询单个分组详情
// @Produce json
// @Tags 分组
// @Security JWT
// @Param id path int true "group id"
// @Success 200 {object} httpx.Response{data=model.Group}
// @Router /api/v1/groups/{id} [get]
func (g *GroupController) Get(c *gin.Context) {
	group, err := g.groupService.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, group)
}

// @Summary 创建分组
// @Description 创建分组及存储配置
// @Accept json
// @Produce json
// @Tags 分组
// @Security JWT
// @Param group body model.CreatedGroup true "group info"
// @Success 200 {object} httpx.Response{data=model.Group}
// @Router /api/v1/groups [post]
func (g *GroupController) Create(c *gin.Context) {
	user := ginctx.GetUser(c)
	if user == nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, fmt.Errorf("failed to get user"))
		return
	}

	createdGroup := new(model.CreatedGroup)
	if err := c.BindJSON(createdGroup); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	group := createdGroup.GetGroup(user.ID)
	ginctx.TraceStep(c, "start create group", trace.Field{"group", group.Name})
	defer ginctx.TraceStep(c, "create group done", trace.Field{"group", group.Name})

	group, err := g.groupService.Create(c.Request.Context(), user, group)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusInternalServerError, err)
		return
	}

	httpx.ResponseSuccess(c, group)
}

// @Summary 更新分组
// @Description 更新分组及存储配置
// @Accept json
// @Produce json
// @Tags 分组
// @Security JWT
// @Param group body model.UpdatedGroup true "group info"
// @Param id   path      int  true  "group id"
// @Success 200 {object} httpx.Response{data=model.Group}
// @Router /api/v1/groups/{id} [put]
func (g *GroupController) Update(c *gin.Context) {
	user := ginctx.GetUser(c)
	if user == nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, fmt.Errorf("failed to get user"))
		return
	}

	id := c.Param("id")

	new := new(model.UpdatedGroup)
	if err := c.BindJSON(new); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	ginctx.TraceStep(c, "start update group", trace.Field{"group", new.Name})
	defer ginctx.TraceStep(c, "update group done", trace.Field{"group", new.Name})

	group, err := g.groupService.Update(c.Request.Context(), id, new.GetGroup(user.ID))
	if err != nil {
		httpx.ResponseFailed(c, http.StatusInternalServerError, err)
		return
	}

	httpx.ResponseSuccess(c, group)
}

// @Summary 删除分组
// @Description 删除分组
// @Produce json
// @Tags 分组
// @Security JWT
// @Param id path int true "group id"
// @Success 200 {object} httpx.Response
// @Router /api/v1/groups/{id} [delete]
func (g *GroupController) Delete(c *gin.Context) {
	user := ginctx.GetUser(c)
	if user == nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, fmt.Errorf("failed to get user"))
		return
	}

	if err := g.groupService.Delete(c.Request.Context(), c.Param("id")); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	httpx.ResponseSuccess(c, nil)
}

// @Summary 分组成员列表
// @Description 查询分组下的成员列表
// @Produce json
// @Tags 分组
// @Security JWT
// @Param id path int true "group id"
// @Success 200 {object} httpx.Response
// @Router /api/v1/groups/{id}/users [get]
func (g *GroupController) GetUsers(c *gin.Context) {
	users, err := g.groupService.GetUsers(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	httpx.ResponseSuccess(c, users)
}

// @Summary 添加分组成员
// @Description 将成员加入分组
// @Produce json
// @Tags 分组
// @Security JWT
// @Param id path int true "group id"
// @Param user body model.User true "user info"
// @Success 200 {object} httpx.Response
// @Router /api/v1/groups/{id}/users [post]
func (g *GroupController) AddUser(c *gin.Context) {
	user := new(model.User)
	if err := c.BindJSON(user); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	if err := g.groupService.AddUser(c.Request.Context(), user, c.Param("id")); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	httpx.ResponseSuccess(c, nil)
}

// @Summary 移除分组成员
// @Description 将成员移出分组
// @Produce json
// @Tags 分组
// @Security JWT
// @Param id path int true "group id"
// @Param id    query     int  true  "member id"
// @Success 200 {object} httpx.Response
// @Router /api/v1/groups/{id}/users [delete]
func (g *GroupController) DelUser(c *gin.Context) {
	// 成员模型无登录名（ADR-006），按成员 ID 移除
	uid, err := strconv.Atoi(c.Query("id"))
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	user := &model.User{ID: uint(uid)}

	if err := g.groupService.DelUser(c.Request.Context(), user, c.Param("id")); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	httpx.ResponseSuccess(c, nil)
}

// @Summary 分组绑定角色
// @Description 为分组绑定角色
// @Produce json
// @Tags 分组
// @Security JWT
// @Param id path int true "group id"
// @Param rid path int true "role id"
// @Success 200 {object} httpx.Response
// @Router /api/v1/groups/{id}/roles/{rid} [post]
func (g *GroupController) AddRole(c *gin.Context) {
	if err := g.groupService.AddRole(c.Request.Context(), c.Param("id"), c.Param("rid")); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	httpx.ResponseSuccess(c, nil)
}

// @Summary 分组解绑角色
// @Description 解除分组绑定的角色
// @Produce json
// @Tags 分组
// @Security JWT
// @Param id path int true "group id"
// @Param rid path int true "role id"
// @Success 200 {object} httpx.Response
// @Router /api/v1/groups/{id}/roles/{rid} [delete]
func (g *GroupController) DelRole(c *gin.Context) {
	if err := g.groupService.DelRole(c.Request.Context(), c.Param("id"), c.Param("rid")); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	httpx.ResponseSuccess(c, nil)
}

func (g *GroupController) RegisterRoute(api *gin.RouterGroup) {
	api.GET("/groups", g.List)
	api.POST("/groups", g.Create)
	api.GET("/groups/:id", g.Get)
	api.PUT("/groups/:id", g.Update)
	api.DELETE("/groups/:id", g.Delete)
	api.GET("/groups/:id/users", g.GetUsers)
	api.POST("/groups/:id/users", g.AddUser)
	api.DELETE("/groups/:id/users", g.DelUser)
	api.POST("/groups/:id/roles/:rid", g.AddRole)
	api.DELETE("/groups/:id/roles/:rid", g.DelRole)
}

func (g *GroupController) Name() string {
	return "Group"
}
