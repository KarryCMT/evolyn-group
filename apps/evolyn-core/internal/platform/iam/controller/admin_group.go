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

// 响应读模型仅出现在 swag 注释（data=model.AdminGroupDetailView），以类型
// 锚定保持 import 合法
var _ = model.AdminGroupDetailView{}

// AdminGroupController 管理组（权限中心-管理员模块）：系统管理员页
// （scope=system）与灵衍云管理员页（scope=application）共用同一套接口，
// 勾选/选择确认即分区块即时保存
type AdminGroupController struct {
	adminGroupService service.AdminGroupService
}

func NewAdminGroupController(adminGroupService service.AdminGroupService) platformcontroller.Controller {
	return &AdminGroupController{adminGroupService: adminGroupService}
}

// @Summary 管理组列表
// @Description 按 scope 查询当前租户的管理组概要（内置系统管理员组恒在最前）；scope=system 为系统管理员页（通讯录管理组），scope=application 为灵衍云管理员页（普通管理组）
// @Produce json
// @Tags 管理组
// @Security JWT
// @Param scope query string false "管理组类型：system|application，缺省返回全部" Enums(system, application)
// @Success 200 {object} httpx.Response{data=[]model.AdminGroupSummary}
// @Failure 400 {object} httpx.Response "errCode=ADMIN_GROUP_CONFIG_INVALID"
// @Router /api/v1/admin-groups [get]
func (a *AdminGroupController) List(c *gin.Context) {
	groups, err := a.adminGroupService.List(c.Request.Context(), c.Query("scope"))
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, groups)
}

// @Summary 管理组详情
// @Description 返回管理组的成员展示视图与范围配置（部门/角色/互联组织/应用范围），字段名与管理员页权限面板一一对应；各类 ID 清单的展示名由前端结合部门树/角色树/应用列表映射
// @Produce json
// @Tags 管理组
// @Security JWT
// @Param id path int true "管理组 ID"
// @Success 200 {object} httpx.Response{data=model.AdminGroupDetailView}
// @Failure 404 {object} httpx.Response "errCode=ADMIN_GROUP_NOT_FOUND"
// @Router /api/v1/admin-groups/{id} [get]
func (a *AdminGroupController) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	detail, err := a.adminGroupService.Get(c.Request.Context(), uint(id))
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, detail)
}

// @Summary 创建管理组
// @Description 创建自定义管理组（仅名称与类型，范围配置随后即时保存）；内置系统管理员组只经系统预置产生，不接受请求侧创建
// @Accept json
// @Produce json
// @Tags 管理组
// @Security JWT
// @Param body body service.AdminGroupCreateRequest true "管理组名称与类型"
// @Success 200 {object} httpx.Response{data=model.AdminGroupDetailView}
// @Failure 400 {object} httpx.Response "errCode=ADMIN_GROUP_NAME_INVALID|ADMIN_GROUP_CONFIG_INVALID"
// @Failure 409 {object} httpx.Response "errCode=ADMIN_GROUP_DUPLICATE_NAME"
// @Router /api/v1/admin-groups [post]
func (a *AdminGroupController) Create(c *gin.Context) {
	req := new(service.AdminGroupCreateRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	ginctx.TraceStep(c, "create admin group")
	detail, err := a.adminGroupService.Create(c.Request.Context(), req)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, detail)
}

// @Summary 即时更新管理组的一个配置区块
// @Description 每次请求至多携带一个区块（name/members/departmentScope/roleScope/externalOrg/applicationScope/addressBook），区块整体替换；内置系统管理员组仅允许 members（代理 tenant-admin 角色绑定），且必须包含租户创建人；创建人不能加入自定义管理组，且至少保留一名管理员
// @Accept json
// @Produce json
// @Tags 管理组
// @Security JWT
// @Param id path int true "管理组 ID"
// @Param body body service.AdminGroupPatchRequest true "本次变更的唯一区块"
// @Success 200 {object} httpx.Response{data=model.AdminGroupDetailView}
// @Failure 400 {object} httpx.Response "errCode=ADMIN_GROUP_CONFIG_INVALID|ADMIN_GROUP_SCOPE_MISMATCH|ADMIN_GROUP_MEMBER_INVALID|ADMIN_GROUP_TENANT_CREATOR_NOT_ALLOWED|ADMIN_GROUP_TENANT_CREATOR_REQUIRED|ADMIN_GROUP_NAME_INVALID"
// @Failure 403 {object} httpx.Response "errCode=ADMIN_GROUP_BUILTIN_IMMUTABLE|ADMIN_GROUP_SELF_REMOVAL_NOT_ALLOWED"
// @Failure 404 {object} httpx.Response "errCode=ADMIN_GROUP_NOT_FOUND"
// @Failure 409 {object} httpx.Response "errCode=ADMIN_GROUP_DUPLICATE_NAME|ADMIN_GROUP_LAST_ADMIN"
// @Router /api/v1/admin-groups/{id} [patch]
func (a *AdminGroupController) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	req := new(service.AdminGroupPatchRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	ginctx.TraceStep(c, "update admin group")
	detail, err := a.adminGroupService.Update(c.Request.Context(), uint(id), req)
	if err != nil {
		// 状态码由 BizError 自动映射（404/403/409/400）
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, detail)
}

// @Summary 删除管理组
// @Description 删除自定义管理组（成员绑定同事务清理）；内置系统管理员组不可删除
// @Produce json
// @Tags 管理组
// @Security JWT
// @Param id path int true "管理组 ID"
// @Success 200 {object} httpx.Response
// @Failure 403 {object} httpx.Response "errCode=ADMIN_GROUP_BUILTIN_IMMUTABLE"
// @Failure 404 {object} httpx.Response "errCode=ADMIN_GROUP_NOT_FOUND"
// @Router /api/v1/admin-groups/{id} [delete]
func (a *AdminGroupController) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	if err := a.adminGroupService.Delete(c.Request.Context(), uint(id)); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, nil)
}

func (a *AdminGroupController) RegisterRoute(api *gin.RouterGroup) {
	api.GET("/admin-groups", a.List)
	api.POST("/admin-groups", a.Create)
	api.GET("/admin-groups/:id", a.Get)
	api.PATCH("/admin-groups/:id", a.Update)
	api.DELETE("/admin-groups/:id", a.Delete)
}

func (a *AdminGroupController) Name() string {
	return "AdminGroup"
}
