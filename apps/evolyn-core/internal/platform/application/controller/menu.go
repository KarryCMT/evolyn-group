// Package controller 应用菜单 HTTP 接口（M2-菜单-1）：解析请求、取当前
// 成员，返回 httpx 统信封。权限由租户域中间件链执行（verb=get），Service
// 内部再经 ApplicationAccessEvaluator 复核（§6.1）
package controller

import (
	"fmt"
	"net/http"
	"strings"

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

// UpdateEntry 菜单节点管理更新（ADR-011）
// @Summary 更新应用菜单节点
// @Description 分组改名 / 资产节点对成员隐藏（须 form-actions:hide 动作授权）/ 移动节点（换父分组或根级，追加到目标末位）；携带 baseMenuRevision 乐观并发口令，冲突返回 409
// @Accept json
// @Produce json
// @Tags 应用管理
// @Security JWT
// @Param code path string true "应用编码（app_ 前缀）"
// @Param entryCode path string true "菜单节点编码（menu_ 前缀）"
// @Param entry body appmodel.UpdateMenuEntryRequest true "更新字段（name 仅分组 / hidden 仅资产节点 / parentEntryCode 空串移动到根级）"
// @Success 200 {object} httpx.Response{data=appmodel.MenuEntryMutation}
// @Failure 400 {object} httpx.Response "errCode=APP_MENU_NAME_INVALID/APP_MENU_ENTRY_RENAME_FORBIDDEN/APP_MENU_HIDDEN_INVALID/APP_MENU_PARENT_INVALID/APP_MENU_MOVE_INVALID/APP_MENU_DEPTH_EXCEEDED"
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN"
// @Failure 404 {object} httpx.Response "errCode=APP_NOT_FOUND/APP_MENU_NOT_FOUND"
// @Failure 409 {object} httpx.Response "errCode=APP_MENU_VERSION_CONFLICT/APP_STATUS_INVALID/APP_PROVISIONING"
// @Router /api/v1/applications/code/{code}/menu/entries/{entryCode} [patch]
func (a *MenuController) UpdateEntry(c *gin.Context) {
	code, ok := codeFromParam(c)
	if !ok {
		return
	}
	entryCode := strings.TrimSpace(c.Param("entryCode"))
	if !strings.HasPrefix(entryCode, "menu_") || len(entryCode) <= len("menu_") {
		httpx.ResponseFailed(c, http.StatusBadRequest, fmt.Errorf("无效的菜单节点编码：%s", entryCode))
		return
	}
	var req appmodel.UpdateMenuEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	updated, err := a.menuService.UpdateEntry(c.Request.Context(), ginctx.GetUser(c), code, entryCode, &req)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, updated)
}

// AddFavorite 收藏菜单节点
// @Summary 收藏应用菜单节点
// @Description 当前成员收藏指定应用的菜单节点（个人状态，凡节点可见即可收藏）；重复收藏幂等成功，不递增菜单修订号
// @Accept json
// @Produce json
// @Tags 应用管理
// @Security JWT
// @Param favorite body appmodel.CreateMenuFavoriteRequest true "应用编码与节点编码"
// @Success 200 {object} httpx.Response{data=appmodel.MenuFavoriteMutation}
// @Failure 400 {object} httpx.Response "errCode=APP_MENU_FAVORITE_INVALID"
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN"
// @Failure 404 {object} httpx.Response "errCode=APP_NOT_FOUND"
// @Router /api/v1/menu-favorites [post]
func (a *MenuController) AddFavorite(c *gin.Context) {
	var req appmodel.CreateMenuFavoriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	updated, err := a.menuService.AddFavorite(c.Request.Context(), ginctx.GetUser(c),
		strings.TrimSpace(req.ApplicationCode), strings.TrimSpace(req.EntryCode))
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, updated)
}

// RemoveFavorite 取消收藏菜单节点
// @Summary 取消收藏应用菜单节点
// @Description 按节点编码取消当前成员的收藏；目标收藏不存在时幂等成功（返回 Favorited=false）
// @Produce json
// @Tags 应用管理
// @Security JWT
// @Param entryCode path string true "菜单节点编码（menu_ 前缀）"
// @Success 200 {object} httpx.Response{data=appmodel.MenuFavoriteMutation}
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN"
// @Router /api/v1/menu-favorites/{entryCode} [delete]
func (a *MenuController) RemoveFavorite(c *gin.Context) {
	entryCode := strings.TrimSpace(c.Param("entryCode"))
	if !strings.HasPrefix(entryCode, "menu_") || len(entryCode) <= len("menu_") {
		httpx.ResponseFailed(c, http.StatusBadRequest, fmt.Errorf("无效的菜单节点编码：%s", entryCode))
		return
	}
	updated, err := a.menuService.RemoveFavorite(c.Request.Context(), ginctx.GetUser(c), entryCode)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, updated)
}

func (a *MenuController) RegisterRoute(api *gin.RouterGroup) {
	// 与既有 /applications/code/:code 同前缀（gin radix tree 静态段优先，
	// 不会被 /applications/:id 捕获）；URL 鉴权解析为 resource=applications
	// verb=get，即 applications:get
	api.GET("/applications/code/:code/menu", a.GetMenu)
	// POST 映射 applications:create；Service 内再次复核相同权限。
	api.POST("/applications/code/:code/menu/groups", a.CreateGroup)
	// PATCH 映射 applications:patch；隐藏开关另经 form-actions:hide 动作复核
	//（ADR-011：动作授权键不随菜单管理权限放大）
	api.PATCH("/applications/code/:code/menu/entries/:entryCode", a.UpdateEntry)
	// 个人收藏（ADR-011）：独立资源 menu-favorites（create/delete 授全体
	// 成员），与菜单管理权限彻底分离，口径同 form-records 与 forms 的关系
	api.POST("/menu-favorites", a.AddFavorite)
	api.DELETE("/menu-favorites/:entryCode", a.RemoveFavorite)
}

func (a *MenuController) Name() string {
	return "ApplicationMenu"
}
