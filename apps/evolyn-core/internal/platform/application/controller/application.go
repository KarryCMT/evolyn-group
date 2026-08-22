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

	appmodel "evolyn/internal/platform/application/model"
	"evolyn/internal/platform/application/service"
	platformcontroller "evolyn/internal/platform/controller"
	"evolyn/internal/platform/ginctx"
	"evolyn/internal/platform/httpx"

	"github.com/gin-gonic/gin"
)

// ApplicationController 应用管理（/applications）
type ApplicationController struct {
	appService service.ApplicationService
}

func NewApplicationController(appService service.ApplicationService) platformcontroller.Controller {
	return &ApplicationController{appService: appService}
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

// @Summary 创建空白应用
// @Description 在当前租户创建空白应用（名称必填，图标/颜色可省略取默认）；服务端生成应用编码，事务内完成配额校验与应用/安装记录写入
// @Accept json
// @Produce json
// @Tags 应用管理
// @Security JWT
// @Param application body appmodel.CreateBlankRequest true "应用名称与外观"
// @Success 201 {object} httpx.Response{data=appmodel.ApplicationDetail}
// @Failure 400 {object} httpx.Response "errCode=APP_NAME_INVALID/APP_ICON_INVALID/APP_COLOR_INVALID"
// @Failure 403 {object} httpx.Response "errCode=QUOTA_EXCEEDED/APP_MEMBER_INVALID/FORBIDDEN"
// @Router /api/v1/applications [post]
func (a *ApplicationController) CreateBlank(c *gin.Context) {
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
// @Success 200 {object} httpx.Response{data=appmodel.ApplicationPage}
// @Failure 400 {object} httpx.Response "errCode=APP_QUERY_INVALID/APP_CURSOR_INVALID"
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN"
// @Router /api/v1/applications [get]
func (a *ApplicationController) List(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	page, err := a.appService.List(c.Request.Context(), ginctx.GetUser(c), appmodel.ListApplicationsQuery{
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
// @Success 200 {object} httpx.Response{data=appmodel.ApplicationDetail}
// @Failure 404 {object} httpx.Response "errCode=APP_NOT_FOUND"
// @Router /api/v1/applications/{id} [get]
func (a *ApplicationController) Get(c *gin.Context) {
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

// @Summary 更新应用
// @Description 白名单字段更新：名称/图标/颜色/排序；status 仅允许 active↔archived 互转（承载归档与恢复）
// @Accept json
// @Produce json
// @Tags 应用管理
// @Security JWT
// @Param id path int true "应用 ID"
// @Param application body appmodel.UpdateApplicationRequest true "更新字段（仅白名单）"
// @Success 200 {object} httpx.Response{data=appmodel.ApplicationDetail}
// @Failure 400 {object} httpx.Response "errCode=APP_NAME_INVALID/APP_ICON_INVALID/APP_COLOR_INVALID"
// @Failure 404 {object} httpx.Response "errCode=APP_NOT_FOUND"
// @Failure 409 {object} httpx.Response "errCode=APP_STATUS_INVALID/APP_PROVISIONING"
// @Router /api/v1/applications/{id} [patch]
func (a *ApplicationController) Update(c *gin.Context) {
	id, ok := idFromParam(c)
	if !ok {
		return
	}

	req := new(appmodel.UpdateApplicationRequest)
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
// @Router /api/v1/applications/{id} [delete]
func (a *ApplicationController) Delete(c *gin.Context) {
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

func (a *ApplicationController) RegisterRoute(api *gin.RouterGroup) {
	api.POST("/applications", a.CreateBlank)
	api.GET("/applications", a.List)
	api.GET("/applications/:id", a.Get)
	api.PATCH("/applications/:id", a.Update)
	api.DELETE("/applications/:id", a.Delete)
}

func (a *ApplicationController) Name() string {
	return "Application"
}
