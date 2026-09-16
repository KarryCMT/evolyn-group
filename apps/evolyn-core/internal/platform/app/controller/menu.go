// Package controller 应用菜单 HTTP 接口（M2-菜单-1）：解析请求、取当前
// 成员，返回 httpx 统信封。权限由租户域中间件链执行（verb=get），Service
// 内部再经 AppAccessEvaluator 复核（§6.1）
package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	appmodel "evolyn/internal/platform/app/model"
	"evolyn/internal/platform/app/service"
	platformcontroller "evolyn/internal/platform/controller"
	"evolyn/internal/platform/ginctx"
	"evolyn/internal/platform/httpx"

	"github.com/gin-gonic/gin"
)

// MenuController 应用菜单（/apps/code/:code/menu）
type MenuController struct {
	menuService service.AppMenuService
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
// @Router /api/v1/apps/code/{code}/menu/groups [post]
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

func NewMenuController(menuService service.AppMenuService) platformcontroller.Controller {
	return &MenuController{menuService: menuService}
}

// GetMenu 获取应用菜单
// @Summary 获取应用菜单
// @Description 按应用编码读取当前成员可见的菜单树：分组/表单/仪表盘/页面统一为菜单节点，nodeMap 仅含可见节点、无可见后代的分组被裁剪；返回 menuRevision 供后续管理接口做乐观并发。表单/仪表盘资产域落地前菜单为空（空树是合法的 200，触发空应用引导）
// @Produce json
// @Tags 应用管理
// @Security JWT
// @Param code path string true "应用编码（app_ 前缀）"
// @Success 200 {object} httpx.Response{data=appmodel.MenuSnapshot}
// @Failure 404 {object} httpx.Response "errCode=APP_NOT_FOUND"
// @Failure 500 {object} httpx.Response "errCode=APP_MENU_INVALID"
// @Router /api/v1/apps/code/{code}/menu [get]
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

// UpdateNode 菜单节点管理更新（ADR-011）
// @Summary 更新应用菜单节点
// @Description 分组改名 / 资产节点对成员隐藏（须 form-actions:hide 动作授权）/ 移动节点（换父分组或根级，追加到目标末位）；携带 baseMenuRevision 乐观并发口令，冲突返回 409
// @Accept json
// @Produce json
// @Tags 应用管理
// @Security JWT
// @Param code path string true "应用编码（app_ 前缀）"
// @Param menuCode path string true "菜单节点编码（menu_ 前缀）"
// @Param node body appmodel.UpdateMenuNodeRequest true "更新字段（name 仅分组 / hidden 仅资产节点 / parentMenuCode 空串移动到根级）"
// @Success 200 {object} httpx.Response{data=appmodel.MenuNodeMutation}
// @Failure 400 {object} httpx.Response "errCode=APP_MENU_NAME_INVALID/APP_MENU_NODE_RENAME_FORBIDDEN/APP_MENU_HIDDEN_INVALID/APP_MENU_PARENT_INVALID/APP_MENU_MOVE_INVALID/APP_MENU_DEPTH_EXCEEDED"
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN"
// @Failure 404 {object} httpx.Response "errCode=APP_NOT_FOUND/APP_MENU_NOT_FOUND"
// @Failure 409 {object} httpx.Response "errCode=APP_MENU_VERSION_CONFLICT/APP_STATUS_INVALID/APP_PROVISIONING"
// @Router /api/v1/apps/code/{code}/menu/nodes/{menuCode} [patch]
func (a *MenuController) UpdateNode(c *gin.Context) {
	code, ok := codeFromParam(c)
	if !ok {
		return
	}
	menuCode := strings.TrimSpace(c.Param("menuCode"))
	if !strings.HasPrefix(menuCode, "menu_") || len(menuCode) <= len("menu_") {
		httpx.ResponseFailed(c, http.StatusBadRequest, fmt.Errorf("无效的菜单节点编码：%s", menuCode))
		return
	}
	var req appmodel.UpdateMenuNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	updated, err := a.menuService.UpdateNode(c.Request.Context(), ginctx.GetUser(c), code, menuCode, &req)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, updated)
}

// AddFavorite 收藏菜单节点
// @Summary 收藏应用菜单节点
// @Description 当前成员收藏指定应用的可打开资产节点（个人状态）。服务端按 canFavorite 统一策略复核：仅 form/dashboard/page 资产叶子节点、应用可用（active 且非初始化中）且节点在当前成员有效可见集内（隐藏、资产软删、表单入口权限失效均拒绝）；分组节点与不可见节点返回 APP_MENU_FAVORITE_INVALID。重复收藏幂等成功，不递增菜单修订号
// @Accept json
// @Produce json
// @Tags 应用管理
// @Security JWT
// @Param favorite body appmodel.CreateMenuFavoriteRequest true "应用编码与节点编码"
// @Success 200 {object} httpx.Response{data=appmodel.MenuFavoriteMutation}
// @Failure 400 {object} httpx.Response "errCode=APP_MENU_FAVORITE_INVALID"
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN"
// @Failure 404 {object} httpx.Response "errCode=APP_NOT_FOUND"
// @Failure 409 {object} httpx.Response "errCode=APP_STATUS_INVALID/APP_PROVISIONING"
// @Router /api/v1/menu-favorites [post]
func (a *MenuController) AddFavorite(c *gin.Context) {
	var req appmodel.CreateMenuFavoriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	updated, err := a.menuService.AddFavorite(c.Request.Context(), ginctx.GetUser(c),
		strings.TrimSpace(req.AppCode), strings.TrimSpace(req.MenuCode))
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, updated)
}

// RemoveFavorite 取消收藏菜单节点
// @Summary 取消收藏应用菜单节点
// @Description 按节点编码取消当前成员的收藏；目标收藏不存在时幂等成功（返回 Favorited=false）。取消不要求目标仍可见——用户需能清除已被隐藏或已删除前留下的个人状态
// @Produce json
// @Tags 应用管理
// @Security JWT
// @Param menuCode path string true "菜单节点编码（menu_ 前缀）"
// @Success 200 {object} httpx.Response{data=appmodel.MenuFavoriteMutation}
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN"
// @Router /api/v1/menu-favorites/{menuCode} [delete]
func (a *MenuController) RemoveFavorite(c *gin.Context) {
	menuCode := strings.TrimSpace(c.Param("menuCode"))
	if !strings.HasPrefix(menuCode, "menu_") || len(menuCode) <= len("menu_") {
		httpx.ResponseFailed(c, http.StatusBadRequest, fmt.Errorf("无效的菜单节点编码：%s", menuCode))
		return
	}
	updated, err := a.menuService.RemoveFavorite(c.Request.Context(), ginctx.GetUser(c), menuCode)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, updated)
}

// ListFavorites 我的收藏列表
// @Summary 获取当前成员的跨应用收藏列表
// @Description 按收藏时间倒序返回当前成员收藏的菜单入口（跨应用），游标分页（不透明 cursor，禁止用页码推断）；应用归档、节点隐藏、资产软删或表单入口权限失效的记录仅过滤不删除，恢复后自然恢复展示
// @Produce json
// @Tags 应用管理
// @Security JWT
// @Param cursor query string false "分页游标（上一页返回的 nextCursor 原样回传）"
// @Param limit query int false "每页数量（默认 20，上限 100）" default(20)
// @Success 200 {object} httpx.Response{data=appmodel.MenuFavoritePage}
// @Failure 400 {object} httpx.Response "errCode=APP_CURSOR_INVALID"
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN"
// @Router /api/v1/menu-favorites [get]
func (a *MenuController) ListFavorites(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	page, err := a.menuService.ListFavorites(c.Request.Context(), ginctx.GetUser(c), appmodel.ListMenuFavoritesQuery{
		Cursor: c.Query("cursor"),
		Limit:  limit,
	})
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, page)
}

func (a *MenuController) RegisterRoute(api *gin.RouterGroup) {
	// 与既有 /apps/code/:code 同前缀（gin radix tree 静态段优先，
	// 不会被 /apps/:id 捕获）；URL 鉴权解析为 resource=apps
	// verb=get，即 apps:get
	api.GET("/apps/code/:code/menu", a.GetMenu)
	// POST 映射 apps:create；Service 内再次复核相同权限。
	api.POST("/apps/code/:code/menu/groups", a.CreateGroup)
	// PATCH 映射 apps:patch；隐藏开关另经 form-actions:hide 动作复核
	//（ADR-011：动作授权键不随菜单管理权限放大）
	api.PATCH("/apps/code/:code/menu/nodes/:menuCode", a.UpdateNode)
	// 个人收藏（ADR-011）：独立资源 menu-favorites（create/delete/list 授
	// 全体成员），与菜单管理权限彻底分离，口径同 form-records 与 forms 的
	// 关系；GET 列表映射 menu-favorites:list（跨应用「我的收藏」，P2）
	api.POST("/menu-favorites", a.AddFavorite)
	api.DELETE("/menu-favorites/:menuCode", a.RemoveFavorite)
	api.GET("/menu-favorites", a.ListFavorites)
}

func (a *MenuController) Name() string {
	return "AppMenu"
}
