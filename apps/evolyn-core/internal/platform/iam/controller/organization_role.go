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

// OrganizationRoleController 为内部组织“角色”页签提供专用契约。路由仍位于
// /roles 资源下，因此复用既有 roles RBAC 权限，而不引入新的授权资源。
type OrganizationRoleController struct {
	service service.OrganizationRoleService
}

func NewOrganizationRoleController(s service.OrganizationRoleService) platformcontroller.Controller {
	return &OrganizationRoleController{service: s}
}

// @Summary 角色组织树
// @Description 查询内部组织页使用的角色组及角色树；未归类的存量角色展示在默认组
// @Produce json
// @Tags 角色权限
// @Security JWT
// @Success 200 {object} httpx.Response{data=service.OrganizationRoleTree}
// @Router /api/v1/roles/tree [get]
func (o *OrganizationRoleController) Tree(c *gin.Context) {
	tree, err := o.service.Tree(c.Request.Context())
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, tree)
}

// @Summary 创建角色组
// @Description 创建仅用于角色展示归类的角色组
// @Accept json
// @Produce json
// @Tags 角色权限
// @Security JWT
// @Param body body service.CreateOrganizationRoleGroupRequest true "角色组信息"
// @Success 200 {object} httpx.Response{data=model.RoleGroup}
// @Failure 400 {object} httpx.Response "errCode=ROLE_GROUP_NAME_INVALID"
// @Failure 409 {object} httpx.Response "errCode=DUPLICATE_NAME"
// @Router /api/v1/roles/groups [post]
func (o *OrganizationRoleController) CreateGroup(c *gin.Context) {
	member := ginctx.GetUser(c)
	if member == nil {
		httpx.ResponseFailed(c, http.StatusUnauthorized, nil)
		return
	}
	req := new(service.CreateOrganizationRoleGroupRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	group, err := o.service.CreateGroup(c.Request.Context(), member.ID, *req)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, group)
}

// @Summary 修改角色组名称
// @Description 修改内部组织页角色展示分组名称
// @Accept json
// @Produce json
// @Tags 角色权限
// @Security JWT
// @Param id path int true "角色组 ID"
// @Param body body string true "角色组名称"
// @Success 200 {object} httpx.Response{data=model.RoleGroup}
// @Failure 400 {object} httpx.Response "errCode=ROLE_GROUP_NAME_INVALID"
// @Failure 409 {object} httpx.Response "errCode=DUPLICATE_NAME"
// @Router /api/v1/roles/groups/{id} [put]
func (o *OrganizationRoleController) RenameGroup(c *gin.Context) {
	var req struct {
		Name string `json:"name"`
	}
	if err := c.BindJSON(&req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	group, err := o.service.RenameGroup(c.Request.Context(), c.Param("id"), req.Name)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, group)
}

// @Summary 删除角色组
// @Description 删除角色展示分组；组内角色将回到默认角色组，角色权限和成员绑定保持不变
// @Produce json
// @Tags 角色权限
// @Security JWT
// @Param id path int true "角色组 ID"
// @Success 200 {object} httpx.Response
// @Router /api/v1/roles/groups/{id} [delete]
func (o *OrganizationRoleController) DeleteGroup(c *gin.Context) {
	if err := o.service.DeleteGroup(c.Request.Context(), c.Param("id")); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, nil)
}

// @Summary 拖拽排序角色组
// @Description 按左侧角色树的完整角色组顺序保存排序
// @Accept json
// @Produce json
// @Tags 角色权限
// @Security JWT
// @Param body body service.ReorderOrganizationRoleGroupRequest true "角色组排序"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response "errCode=ORGANIZATION_ROLE_REQUEST_INVALID"
// @Router /api/v1/roles/groups/order [put]
func (o *OrganizationRoleController) ReorderGroups(c *gin.Context) {
	req := new(service.ReorderOrganizationRoleGroupRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	if err := o.service.ReorderGroups(c.Request.Context(), req.GroupIDs); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, nil)
}

// @Summary 在角色组内创建角色
// @Description 创建角色并指定其唯一的内部组织展示分组
// @Accept json
// @Produce json
// @Tags 角色权限
// @Security JWT
// @Param body body service.CreateOrganizationRoleRequest true "角色信息"
// @Success 200 {object} httpx.Response{data=model.Role}
// @Failure 400 {object} httpx.Response "errCode=ORGANIZATION_ROLE_REQUEST_INVALID"
// @Failure 409 {object} httpx.Response "errCode=DUPLICATE_NAME"
// @Router /api/v1/roles/organization [post]
func (o *OrganizationRoleController) CreateRole(c *gin.Context) {
	req := new(service.CreateOrganizationRoleRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	role, err := o.service.CreateRole(c.Request.Context(), *req)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, role)
}

// @Summary 在指定角色组添加角色
// @Description 用于角色组操作菜单，角色自动归属路径指定的角色组
// @Accept json
// @Produce json
// @Tags 角色权限
// @Security JWT
// @Param id path int true "角色组 ID"
// @Param body body string true "角色名称"
// @Success 200 {object} httpx.Response{data=model.Role}
// @Failure 400 {object} httpx.Response "errCode=ORGANIZATION_ROLE_REQUEST_INVALID"
// @Failure 409 {object} httpx.Response "errCode=DUPLICATE_NAME"
// @Router /api/v1/roles/groups/{id}/roles [post]
func (o *OrganizationRoleController) CreateRoleInGroup(c *gin.Context) {
	var req struct {
		Name string `json:"name"`
	}
	if err := c.BindJSON(&req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	groupID, err := parseUintParam(c.Param("id"))
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, service.ErrOrganizationRoleRequestInvalid)
		return
	}
	role, err := o.service.CreateRole(c.Request.Context(), service.CreateOrganizationRoleRequest{Name: req.Name, GroupID: groupID})
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, role)
}

// @Summary 修改角色名称
// @Description 仅修改角色名称，不会覆盖角色已配置的权限规则
// @Accept json
// @Produce json
// @Tags 角色权限
// @Security JWT
// @Param id path int true "角色 ID"
// @Param body body string true "角色名称"
// @Success 200 {object} httpx.Response{data=model.Role}
// @Failure 409 {object} httpx.Response "errCode=DUPLICATE_NAME"
// @Router /api/v1/roles/{id}/name [put]
func (o *OrganizationRoleController) RenameRole(c *gin.Context) {
	var req struct {
		Name string `json:"name"`
	}
	if err := c.BindJSON(&req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	role, err := o.service.RenameRole(c.Request.Context(), c.Param("id"), req.Name)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, role)
}

// @Summary 调整角色分组
// @Description 将角色移动到指定角色组，一个角色仅能归属一个展示分组
// @Accept json
// @Produce json
// @Tags 角色权限
// @Security JWT
// @Param id path int true "角色 ID"
// @Param groupId body string true "目标角色组 ID"
// @Success 200 {object} httpx.Response{data=model.Role}
// @Router /api/v1/roles/{id}/group [put]
func (o *OrganizationRoleController) MoveRole(c *gin.Context) {
	var req struct {
		GroupID string `json:"groupId"`
	}
	if err := c.BindJSON(&req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	role, err := o.service.MoveRole(c.Request.Context(), c.Param("id"), req.GroupID)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, role)
}

// @Summary 拖拽排序角色
// @Description 按角色组内完整角色顺序保存排序；跨组移动请使用调整分组
// @Accept json
// @Produce json
// @Tags 角色权限
// @Security JWT
// @Param id path int true "角色组 ID"
// @Param body body service.ReorderOrganizationRoleRequest true "角色排序"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response "errCode=ORGANIZATION_ROLE_REQUEST_INVALID"
// @Router /api/v1/roles/groups/{id}/roles/order [put]
func (o *OrganizationRoleController) ReorderRoles(c *gin.Context) {
	req := new(service.ReorderOrganizationRoleRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	if err := o.service.ReorderRoles(c.Request.Context(), c.Param("id"), req.RoleIDs); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, nil)
}

// @Summary 角色成员列表
// @Description 分页查询直接绑定指定角色的成员，可按关键词和成员状态筛选
// @Produce json
// @Tags 角色权限
// @Security JWT
// @Param id path int true "角色 ID"
// @Param status query string false "成员状态"
// @Param keyword query string false "姓名、手机号或邮箱关键词"
// @Param page query int false "页码"
// @Param pageSize query int false "每页条数"
// @Success 200 {object} httpx.Response{data=model.MemberPage}
// @Router /api/v1/roles/{id}/members [get]
func (o *OrganizationRoleController) ListMembers(c *gin.Context) {
	page, err := o.service.ListMembers(c.Request.Context(), c.Param("id"), model.MemberListQuery{
		Status: c.Query("status"), Keyword: c.Query("keyword"), Page: queryInt(c, "page"), PageSize: queryInt(c, "pageSize"),
	})
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, page)
}

// @Summary 批量添加角色成员
// @Description 将成员直接绑定至指定角色，全部成员均成功后才提交
// @Accept json
// @Produce json
// @Tags 角色权限
// @Security JWT
// @Param id path int true "角色 ID"
// @Param body body service.OrganizationRoleMemberRequest true "成员 ID 列表"
// @Success 200 {object} httpx.Response
// @Router /api/v1/roles/{id}/members [post]
func (o *OrganizationRoleController) AddMembers(c *gin.Context) {
	req := new(service.OrganizationRoleMemberRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	if err := o.service.AddMembers(c.Request.Context(), c.Param("id"), req.MemberIDs); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, nil)
}

// @Summary 移出角色成员
// @Description 解除成员与指定角色的直接绑定
// @Produce json
// @Tags 角色权限
// @Security JWT
// @Param id path int true "角色 ID"
// @Param memberId path int true "成员 ID"
// @Success 200 {object} httpx.Response
// @Router /api/v1/roles/{id}/members/{memberId} [delete]
func (o *OrganizationRoleController) RemoveMember(c *gin.Context) {
	if err := o.service.RemoveMember(c.Request.Context(), c.Param("id"), c.Param("memberId")); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, nil)
}

// @Summary 设置成员角色
// @Description 原子替换成员的全部直接角色，支持传入空列表以解除全部角色
// @Accept json
// @Produce json
// @Tags 角色权限
// @Security JWT
// @Param id path int true "成员 ID"
// @Param body body service.ReplaceMemberRolesRequest true "角色 ID 列表"
// @Success 200 {object} httpx.Response
// @Router /api/v1/members/{id}/roles [put]
func (o *OrganizationRoleController) ReplaceMemberRoles(c *gin.Context) {
	req := new(service.ReplaceMemberRolesRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	if err := o.service.ReplaceMemberRoles(c.Request.Context(), c.Param("id"), req.RoleIDs); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, nil)
}

func (o *OrganizationRoleController) RegisterRoute(api *gin.RouterGroup) {
	api.GET("/roles/tree", o.Tree)
	api.POST("/roles/groups", o.CreateGroup)
	api.PUT("/roles/groups/order", o.ReorderGroups)
	api.PUT("/roles/groups/:id", o.RenameGroup)
	api.DELETE("/roles/groups/:id", o.DeleteGroup)
	api.PUT("/roles/groups/:id/roles/order", o.ReorderRoles)
	api.POST("/roles/groups/:id/roles", o.CreateRoleInGroup)
	api.POST("/roles/organization", o.CreateRole)
	api.PUT("/roles/:id/name", o.RenameRole)
	api.PUT("/roles/:id/group", o.MoveRole)
	api.GET("/roles/:id/members", o.ListMembers)
	api.POST("/roles/:id/members", o.AddMembers)
	api.DELETE("/roles/:id/members/:memberId", o.RemoveMember)
	api.PUT("/members/:id/roles", o.ReplaceMemberRoles)
}

func parseUintParam(value string) (uint, error) {
	id, err := strconv.ParseUint(value, 10, 0)
	if err != nil || id == 0 {
		return 0, service.ErrOrganizationRoleRequestInvalid
	}
	return uint(id), nil
}

func (*OrganizationRoleController) Name() string { return "OrganizationRole" }
