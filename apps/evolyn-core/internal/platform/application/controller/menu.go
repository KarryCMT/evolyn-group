// Package controller 应用菜单 HTTP 接口（M2-菜单-1）：解析请求、取当前
// 成员，返回 httpx 统信封。权限由租户域中间件链执行（verb=get），Service
// 内部再经 ApplicationAccessEvaluator 复核（§6.1）
package controller

import (
	"net/http"

	appmodel "evolyn/internal/platform/application/model"
	"evolyn/internal/platform/application/service"
	platformcontroller "evolyn/internal/platform/controller"
	"evolyn/internal/platform/ginctx"
	"evolyn/internal/platform/httpx"

	"github.com/gin-gonic/gin"
)

// MenuController 应用菜单（/applications/code/:code/menu）
type MenuController struct {
	menuService service.ApplicationMenuService
}

// CreateGroup 创建菜单分组
// @Summary 创建应用菜单分组
// @Description 在应用根级或一级分组下创建分组；请求携带最近读取到的 menuRevision，发生并发更新时返回冲突并要求客户端刷新菜单
// @Accept json
// @Produce json
// @Tags 应用管理
// @Security JWT
// @Param code path string true "应用编码（app_ 前缀）"
// @Param group body appmodel.CreateMenuGroupRequest true "分组名称、父节点与菜单修订号"
// @Success 201 {object} httpx.Response{data=appmodel.MenuGroupMutation}
// @Failure 400 {object} httpx.Response "errCode=APP_MENU_NAME_INVALID/APP_MENU_PARENT_INVALID/APP_MENU_DEPTH_EXCEEDED"
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN"
// @Failure 404 {object} httpx.Response "errCode=APP_NOT_FOUND"
// @Failure 409 {object} httpx.Response "errCode=APP_MENU_VERSION_CONFLICT/APP_STATUS_INVALID/APP_PROVISIONING"
// @Router /api/v1/applications/code/{code}/menu/groups [post]
func (a *MenuController) CreateGroup(c *gin.Context) {
	code, ok := codeFromParam(c)
	if !ok {
		return
	}
	var req appmodel.CreateMenuGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	created, err := a.menuService.CreateGroup(c.Request.Context(), ginctx.GetUser(c), code, &req)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.NewResponse(c, http.StatusCreated, created, "创建成功")
}

func NewMenuController(menuService service.ApplicationMenuService) platformcontroller.Controller {
	return &MenuController{menuService: menuService}
}

// GetMenu 获取应用菜单
// @Summary 获取应用菜单
// @Description 按应用编码读取当前成员可见的菜单树：分组/表单/仪表盘/页面统一为菜单节点，entryMap 仅含可见节点、无可见后代的分组被裁剪；返回 menuRevision 供后续管理接口做乐观并发。表单/仪表盘资产域落地前菜单为空（空树是合法的 200，触发空应用引导）
// @Produce json
// @Tags 应用管理
// @Security JWT
// @Param code path string true "应用编码（app_ 前缀）"
// @Success 200 {object} httpx.Response{data=appmodel.MenuSnapshot}
// @Failure 404 {object} httpx.Response "errCode=APP_NOT_FOUND"
// @Failure 500 {object} httpx.Response "errCode=APP_MENU_INVALID"
// @Router /api/v1/applications/code/{code}/menu [get]
func (a *MenuController) GetMenu(c *gin.Context) {
	code, ok := codeFromParam(c)
	if !ok {
		return
	}

	// 显式声明出网类型：swag 以本文件 import 别名解析注释中的 MenuSnapshot
	var menu *appmodel.MenuSnapshot
	menu, err := a.menuService.GetMenu(c.Request.Context(), ginctx.GetUser(c), code)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, menu)
}

func (a *MenuController) RegisterRoute(api *gin.RouterGroup) {
	// 与既有 /applications/code/:code 同前缀（gin radix tree 静态段优先，
	// 不会被 /applications/:id 捕获）；URL 鉴权解析为 resource=applications
	// verb=get，即 applications:get
	api.GET("/applications/code/:code/menu", a.GetMenu)
	// POST 映射 applications:create；Service 内再次复核相同权限。
	api.POST("/applications/code/:code/menu/groups", a.CreateGroup)
}

func (a *MenuController) Name() string {
	return "ApplicationMenu"
}
