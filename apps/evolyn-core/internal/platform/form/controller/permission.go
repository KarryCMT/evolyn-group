// 权限组配置面 HTTP 接口（表单权限 P1，设计 §7）：路由挂 /forms 首段
// （中间件 URL 门解析为 forms:get/create/update/delete），配置面权限键
// form-permissions:list/create/update/delete 由 Service 层按权限集独立复核。
package controller

import (
	"errors"
	"net/http"
	"strings"

	platformcontroller "evolyn/internal/platform/controller"
	formmodel "evolyn/internal/platform/form/model"
	"evolyn/internal/platform/form/service"
	"evolyn/internal/platform/ginctx"
	"evolyn/internal/platform/httpx"

	"github.com/gin-gonic/gin"
)

// PermissionGroupController 表单资产权限组（配置面）
type PermissionGroupController struct {
	permissionService service.PermissionGroupService
}

func NewPermissionGroupController(permissionService service.PermissionGroupService) platformcontroller.Controller {
	return &PermissionGroupController{permissionService: permissionService}
}

// permissionGroupCodeFromParam 解析权限组公开编码，非法直接回 400。
func permissionGroupCodeFromParam(c *gin.Context, name string) (string, bool) {
	code := strings.TrimSpace(c.Param(name))
	if !strings.HasPrefix(code, "fpg_") || len(code) <= len("fpg_") {
		httpx.ResponseFailed(c, http.StatusBadRequest, errors.New("无效的权限组编码："+code))
		return "", false
	}
	return code, true
}

// @Summary 表单权限组清单
// @Description 按表单编码读取全部权限组（含禁用组，四要素整体出网：操作集/字段矩阵/数据范围/主体清单，主体携带展示名）
// @Produce json
// @Tags 表单管理
// @Security JWT
// @Param code path string true "表单编码（form_ 前缀）"
// @Success 200 {object} httpx.Response{data=[]formmodel.PermissionGroupView}
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN（须 form-permissions:list）"
// @Failure 404 {object} httpx.Response "errCode=FORM_NOT_FOUND"
// @Router /api/v1/forms/{code}/permission-groups [get]
func (f *PermissionGroupController) ListGroups(c *gin.Context) {
	code, ok := formCodeFromParam(c, "code")
	if !ok {
		return
	}
	groups, err := f.permissionService.ListGroups(c.Request.Context(), ginctx.GetUser(c), code)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, groups)
}

// @Summary 创建表单权限组
// @Description 创建权限组（四要素整体提交：操作集按表单类型合法集校验、字段矩阵按字段清单校验含必填协调、数据范围按字段类型分派 operator 白名单、主体校验同租户存在性）；单表单上限 50 组
// @Accept json
// @Produce json
// @Tags 表单管理
// @Security JWT
// @Param code path string true "表单编码（form_ 前缀）"
// @Param group body formmodel.CreatePermissionGroupRequest true "名称/描述/启用状态/操作集/字段矩阵/数据范围/主体清单"
// @Success 201 {object} httpx.Response{data=formmodel.PermissionGroupView}
// @Failure 400 {object} httpx.Response "errCode=FORM_PERMISSION_NAME_INVALID/FORM_PERMISSION_OPERATION_INVALID/FORM_PERMISSION_FIELD_INVALID/FORM_PERMISSION_DATA_SCOPE_INVALID/FORM_PERMISSION_SUBJECT_INVALID/FORM_PERMISSION_LIMIT_EXCEEDED"
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN（须 form-permissions:create）"
// @Failure 404 {object} httpx.Response "errCode=FORM_NOT_FOUND"
// @Router /api/v1/forms/{code}/permission-groups [post]
func (f *PermissionGroupController) CreateGroup(c *gin.Context) {
	code, ok := formCodeFromParam(c, "code")
	if !ok {
		return
	}
	req := new(formmodel.CreatePermissionGroupRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	group, err := f.permissionService.CreateGroup(c.Request.Context(), ginctx.GetUser(c), code, req)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.NewResponse(c, http.StatusCreated, group, "创建成功")
}

// @Summary 更新表单权限组
// @Description 整组全量更新（PUT，四要素整体替换；baseRevision 为整组乐观锁口令，冲突返回 409 FORM_PERMISSION_REVISION_CONFLICT）
// @Accept json
// @Produce json
// @Tags 表单管理
// @Security JWT
// @Param code path string true "表单编码（form_ 前缀）"
// @Param groupCode path string true "权限组编码（fpg_ 前缀）"
// @Param group body formmodel.UpdatePermissionGroupRequest true "四要素全量 + baseRevision"
// @Success 200 {object} httpx.Response{data=formmodel.PermissionGroupView}
// @Failure 400 {object} httpx.Response "errCode=FORM_PERMISSION_NAME_INVALID/FORM_PERMISSION_OPERATION_INVALID/FORM_PERMISSION_FIELD_INVALID/FORM_PERMISSION_DATA_SCOPE_INVALID/FORM_PERMISSION_SUBJECT_INVALID"
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN（须 form-permissions:update）"
// @Failure 404 {object} httpx.Response "errCode=FORM_NOT_FOUND/FORM_PERMISSION_GROUP_NOT_FOUND"
// @Failure 409 {object} httpx.Response "errCode=FORM_PERMISSION_REVISION_CONFLICT"
// @Router /api/v1/forms/{code}/permission-groups/{groupCode} [put]
func (f *PermissionGroupController) UpdateGroup(c *gin.Context) {
	code, ok := formCodeFromParam(c, "code")
	if !ok {
		return
	}
	groupCode, ok := permissionGroupCodeFromParam(c, "groupCode")
	if !ok {
		return
	}
	req := new(formmodel.UpdatePermissionGroupRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	group, err := f.permissionService.UpdateGroup(c.Request.Context(), ginctx.GetUser(c), code, groupCode, req)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, group)
}

// @Summary 删除表单权限组
// @Description 软删权限组并同事务硬删主体关联行；删除后若表单不再存在任何权限组行则回落基线（S4）
// @Produce json
// @Tags 表单管理
// @Security JWT
// @Param code path string true "表单编码（form_ 前缀）"
// @Param groupCode path string true "权限组编码（fpg_ 前缀）"
// @Success 200 {object} httpx.Response
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN（须 form-permissions:delete）"
// @Failure 404 {object} httpx.Response "errCode=FORM_NOT_FOUND/FORM_PERMISSION_GROUP_NOT_FOUND"
// @Router /api/v1/forms/{code}/permission-groups/{groupCode} [delete]
func (f *PermissionGroupController) DeleteGroup(c *gin.Context) {
	code, ok := formCodeFromParam(c, "code")
	if !ok {
		return
	}
	groupCode, ok := permissionGroupCodeFromParam(c, "groupCode")
	if !ok {
		return
	}
	if err := f.permissionService.DeleteGroup(c.Request.Context(), ginctx.GetUser(c), code, groupCode); err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, nil)
}

// @Summary 表单权限配置字段清单
// @Description 返回权限配置可用的字段清单（最新发布版本 schema 提取，未发布回落草稿；仅值字段，含 label/type/required），字段矩阵与数据范围配置的事实源
// @Produce json
// @Tags 表单管理
// @Security JWT
// @Param code path string true "表单编码（form_ 前缀）"
// @Success 200 {object} httpx.Response{data=[]formmodel.PermissionFieldView}
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN（须 form-permissions:list）"
// @Failure 404 {object} httpx.Response "errCode=FORM_NOT_FOUND"
// @Router /api/v1/forms/{code}/permission-fields [get]
func (f *PermissionGroupController) ListFields(c *gin.Context) {
	code, ok := formCodeFromParam(c, "code")
	if !ok {
		return
	}
	fields, err := f.permissionService.ListPermissionFields(c.Request.Context(), ginctx.GetUser(c), code)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, fields)
}

func (f *PermissionGroupController) RegisterRoute(api *gin.RouterGroup) {
	// 配置面路由挂 /forms 首段：URL 门与表单管理一致（forms:get/create/
	// update/delete），form-permissions:* 由 Service 层独立复核
	api.GET("/forms/:code/permission-groups", f.ListGroups)
	api.POST("/forms/:code/permission-groups", f.CreateGroup)
	api.PUT("/forms/:code/permission-groups/:groupCode", f.UpdateGroup)
	api.DELETE("/forms/:code/permission-groups/:groupCode", f.DeleteGroup)
	api.GET("/forms/:code/permission-fields", f.ListFields)
}

func (f *PermissionGroupController) Name() string {
	return "FormPermissionGroup"
}
