// Package controller 应用管理域 HTTP 接口（M2-A）：解析请求、取当前成员、
// 返回 httpx 统信封。权限由租户域中间件链执行（Authentication → Tenant →
// TenantStatus → Authorization），动词复用既有集合（§5.4：
// POST=create / GET(list)=list / GET=get / PATCH=patch / DELETE=delete）
package controller

import (
	"errors"
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

// AppController 应用管理（/apps）
type AppController struct {
	appService service.AppService
}

func NewAppController(appService service.AppService) platformcontroller.Controller {
	return &AppController{appService: appService}
}

// responseError 错误统一出口（ADR-008 脱敏）：BizError 按自带状态码出网；
// 非 BizError（数据库/连接等未分类错误）一律 500 进 ResponseFailed 脱敏
// 分支——原文只入日志，避免以 4xx 回显内部细节
func responseError(c *gin.Context, err error) {
	var biz *httpx.BizError
	if errors.As(err, &biz) && biz.HTTP != 0 {
		httpx.ResponseFailed(c, biz.HTTP, err)
		return
	}
	httpx.ResponseFailed(c, http.StatusInternalServerError, err)
}

// idFromParam 解析路径参数中的应用 ID，非法直接回 400（参数类文案）
func idFromParam(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		httpx.ResponseFailed(c, http.StatusBadRequest, fmt.Errorf("无效的应用 ID：%s", c.Param("id")))
		return 0, false
	}
	return uint(id), true
}

// codeFromParam 解析路径参数中的应用编码，去空格后为空即回 400（参数类文案）
func codeFromParam(c *gin.Context) (string, bool) {
	code := strings.TrimSpace(c.Param("code"))
	if code == "" {
		httpx.ResponseFailed(c, http.StatusBadRequest, fmt.Errorf("无效的应用编码：%s", c.Param("code")))
		return "", false
	}
	return code, true
}

// @Summary 创建空白应用
// @Description 在当前租户创建空白应用（名称必填，图标/颜色可省略取默认）；服务端生成应用编码，事务内完成配额校验与应用/安装记录写入
// @Accept json
// @Produce json
// @Tags 应用管理
// @Security JWT
// @Param app body appmodel.CreateBlankRequest true "应用名称与外观"
// @Success 201 {object} httpx.Response{data=appmodel.AppDetail}
// @Failure 400 {object} httpx.Response "errCode=APP_NAME_INVALID/APP_ICON_INVALID/APP_COLOR_INVALID"
// @Failure 403 {object} httpx.Response "errCode=QUOTA_EXCEEDED/APP_MEMBER_INVALID/FORBIDDEN"
// @Router /api/v1/apps [post]
func (a *AppController) CreateBlank(c *gin.Context) {
	req := new(appmodel.CreateBlankRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	detail, err := a.appService.CreateBlank(c.Request.Context(), ginctx.GetUser(c), req)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.NewResponse(c, http.StatusCreated, detail, "创建成功")
}

// @Summary 应用列表
// @Description 当前租户应用列表：keyword 按名称模糊、status 过滤（active/archived），cursor 游标分页（不透明值原样回传）
// @Produce json
// @Tags 应用管理
// @Security JWT
// @Param keyword query string false "名称关键词"
// @Param status query string false "状态过滤：active/archived"
// @Param limit query int false "每页数量，默认 20，上限 100"
// @Param cursor query string false "分页游标（上一页 nextCursor 原样回传）"
// @Success 200 {object} httpx.Response{data=appmodel.AppPage}
// @Failure 400 {object} httpx.Response "errCode=APP_QUERY_INVALID/APP_CURSOR_INVALID"
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN"
// @Router /api/v1/apps [get]
func (a *AppController) List(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	page, err := a.appService.List(c.Request.Context(), ginctx.GetUser(c), appmodel.ListAppsQuery{
		Keyword: c.Query("keyword"),
		Status:  c.Query("status"),
		Limit:   limit,
		Cursor:  c.Query("cursor"),
	})
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, page)
}

// @Summary 应用详情
// @Description 按 ID 查询应用详情，含当前成员运行时能力（capabilities 读取时派生）
// @Produce json
// @Tags 应用管理
// @Security JWT
// @Param id path int true "应用 ID"
// @Success 200 {object} httpx.Response{data=appmodel.AppDetail}
// @Failure 404 {object} httpx.Response "errCode=APP_NOT_FOUND"
// @Router /api/v1/apps/{id} [get]
func (a *AppController) Get(c *gin.Context) {
	id, ok := idFromParam(c)
	if !ok {
		return
	}

	detail, err := a.appService.Get(c.Request.Context(), ginctx.GetUser(c), id)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, detail)
}

// @Summary 按编码查询应用详情
// @Description 按应用编码（code，租户内唯一）查询应用详情，响应结构与按 ID 查询一致，含当前成员运行时能力（capabilities 读取时派生）；工作区等以 code 定位应用的入口使用
// @Produce json
// @Tags 应用管理
// @Security JWT
// @Param code path string true "应用编码（app_ 前缀）"
// @Success 200 {object} httpx.Response{data=appmodel.AppDetail}
// @Failure 400 {object} httpx.Response "应用编码为空"
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN"
// @Failure 404 {object} httpx.Response "errCode=APP_NOT_FOUND"
// @Router /api/v1/apps/code/{code} [get]
func (a *AppController) GetByCode(c *gin.Context) {
	code, ok := codeFromParam(c)
	if !ok {
		return
	}

	detail, err := a.appService.GetByCode(c.Request.Context(), ginctx.GetUser(c), code)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, detail)
}

// @Summary 更新应用
// @Description 白名单字段更新：名称/图标/颜色/排序；status 仅允许 active↔archived 互转（承载归档与恢复）
// @Accept json
// @Produce json
// @Tags 应用管理
// @Security JWT
// @Param id path int true "应用 ID"
// @Param app body appmodel.UpdateAppRequest true "更新字段（仅白名单）"
// @Success 200 {object} httpx.Response{data=appmodel.AppDetail}
// @Failure 400 {object} httpx.Response "errCode=APP_NAME_INVALID/APP_ICON_INVALID/APP_COLOR_INVALID"
// @Failure 404 {object} httpx.Response "errCode=APP_NOT_FOUND"
// @Failure 409 {object} httpx.Response "errCode=APP_STATUS_INVALID/APP_PROVISIONING"
// @Router /api/v1/apps/{id} [patch]
func (a *AppController) Update(c *gin.Context) {
	id, ok := idFromParam(c)
	if !ok {
		return
	}

	req := new(appmodel.UpdateAppRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	detail, err := a.appService.Update(c.Request.Context(), ginctx.GetUser(c), id, req)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, detail)
}

// @Summary 删除应用
// @Description 软删除应用（仅写 deleted_at，立即从列表隐藏并释放配额）；初始化进行中的应用不可删除
// @Produce json
// @Tags 应用管理
// @Security JWT
// @Param id path int true "应用 ID"
// @Success 200 {object} httpx.Response
// @Failure 404 {object} httpx.Response "errCode=APP_NOT_FOUND"
// @Failure 409 {object} httpx.Response "errCode=APP_PROVISIONING"
// @Router /api/v1/apps/{id} [delete]
func (a *AppController) Delete(c *gin.Context) {
	id, ok := idFromParam(c)
	if !ok {
		return
	}

	if err := a.appService.Delete(c.Request.Context(), ginctx.GetUser(c), id); err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, nil)
}

func (a *AppController) RegisterRoute(api *gin.RouterGroup) {
	api.POST("/apps", a.CreateBlank)
	api.GET("/apps", a.List)
	// 静态段 code 与参数段 :id 同层共存（gin radix tree 静态优先），
	// code 路径不会被 :id 捕获
	api.GET("/apps/code/:code", a.GetByCode)
	api.GET("/apps/:id", a.Get)
	api.PATCH("/apps/:id", a.Update)
	api.DELETE("/apps/:id", a.Delete)
}

func (a *AppController) Name() string {
	return "App"
}
