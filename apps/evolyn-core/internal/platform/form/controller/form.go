// Package controller 表单资产域 HTTP 接口：解析请求、取当前成员、返回 httpx 统信封。
// 权限由租户域中间件链执行；URL 首段即 RBAC 资源名（/forms → forms、
// /form-records → form-records、bootstrap 挂 /applications 前缀 → applications:get）。
package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	platformcontroller "evolyn/internal/platform/controller"
	formmodel "evolyn/internal/platform/form/model"
	"evolyn/internal/platform/form/service"
	"evolyn/internal/platform/ginctx"
	"evolyn/internal/platform/httpx"

	"github.com/gin-gonic/gin"
)

// FormController 表单资产（/forms、/form-records、运行时 bootstrap）
type FormController struct {
	formService service.FormService
}

func NewFormController(formService service.FormService) platformcontroller.Controller {
	return &FormController{formService: formService}
}

// responseError 错误统一出口（ADR-008 脱敏）：BizError 按自带状态码出网
// （含 data 负载的码如 FORM_SCHEMA_INVALID/FORM_RECORD_INVALID 原样透传）；
// 未分类错误一律 500 脱敏。
func responseError(c *gin.Context, err error) {
	var biz *httpx.BizError
	if errors.As(err, &biz) && biz.HTTP != 0 {
		httpx.ResponseFailed(c, biz.HTTP, err)
		return
	}
	httpx.ResponseFailed(c, http.StatusInternalServerError, err)
}

// idFromParam 解析路径参数中的表单 ID，非法直接回 400
func idFromParam(c *gin.Context, name string) (uint, bool) {
	id, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil || id == 0 {
		httpx.ResponseFailed(c, http.StatusBadRequest, fmt.Errorf("无效的表单 ID：%s", c.Param(name)))
		return 0, false
	}
	return uint(id), true
}

// @Summary 创建表单
// @Description 在当前租户创建表单资产（名称必填，归属指定应用）；事务内完成 forms 配额校验，草稿初始化为空目标协议文档。parentEntryCode 可选：传入时菜单节点挂到该分组下（须为同应用分组节点编码），否则挂应用根级
// @Accept json
// @Produce json
// @Tags 表单管理
// @Security JWT
// @Param form body formmodel.CreateFormRequest true "应用 ID、表单名称与可选父分组编码"
// @Success 201 {object} httpx.Response{data=formmodel.FormDetail}
// @Failure 400 {object} httpx.Response "errCode=FORM_NAME_INVALID/FORM_APP_INVALID/APP_MENU_PARENT_INVALID"
// @Failure 403 {object} httpx.Response "errCode=QUOTA_EXCEEDED/FORBIDDEN"
// @Router /api/v1/forms [post]
func (f *FormController) Create(c *gin.Context) {
	req := new(formmodel.CreateFormRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	detail, err := f.formService.Create(c.Request.Context(), ginctx.GetUser(c), req)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.NewResponse(c, http.StatusCreated, detail, "创建成功")
}

// @Summary 应用内表单列表
// @Description 按应用过滤的表单游标分页（id 倒序，新表单靠前）；cursor 不透明值原样回传
// @Produce json
// @Tags 表单管理
// @Security JWT
// @Param applicationId query int true "应用 ID"
// @Param limit query int false "每页数量，默认 20，上限 100"
// @Param cursor query string false "分页游标（上一页 nextCursor 原样回传）"
// @Success 200 {object} httpx.Response{data=formmodel.FormPage}
// @Failure 400 {object} httpx.Response "errCode=FORM_APP_INVALID"
// @Router /api/v1/forms [get]
func (f *FormController) List(c *gin.Context) {
	applicationID, err := strconv.ParseUint(c.Query("applicationId"), 10, 64)
	if err != nil || applicationID == 0 {
		httpx.ResponseFailed(c, http.StatusBadRequest, fmt.Errorf("无效的应用 ID：%s", c.Query("applicationId")))
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	page, err := f.formService.List(c.Request.Context(), ginctx.GetUser(c), formmodel.ListFormsQuery{
		ApplicationID: uint(applicationID),
		Limit:         limit,
		Cursor:        c.Query("cursor"),
	})
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, page)
}

// @Summary 表单详情
// @Description 按 ID 查询表单详情，含目标协议草稿全文与草稿修订口令（draftRevision）、最新发布号
// @Produce json
// @Tags 表单管理
// @Security JWT
// @Param id path int true "表单 ID"
// @Success 200 {object} httpx.Response{data=formmodel.FormDetail}
// @Failure 404 {object} httpx.Response "errCode=FORM_NOT_FOUND"
// @Router /api/v1/forms/{id} [get]
func (f *FormController) Get(c *gin.Context) {
	id, ok := idFromParam(c, "id")
	if !ok {
		return
	}
	detail, err := f.formService.Get(c.Request.Context(), ginctx.GetUser(c), id)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, detail)
}

// @Summary 更新表单
// @Description 白名单字段更新：仅名称（trim 后 1–128 字符）
// @Accept json
// @Produce json
// @Tags 表单管理
// @Security JWT
// @Param id path int true "表单 ID"
// @Param form body formmodel.UpdateFormRequest true "更新字段（仅白名单）"
// @Success 200 {object} httpx.Response{data=formmodel.FormDetail}
// @Failure 400 {object} httpx.Response "errCode=FORM_NAME_INVALID"
// @Failure 404 {object} httpx.Response "errCode=FORM_NOT_FOUND"
// @Router /api/v1/forms/{id} [patch]
func (f *FormController) Update(c *gin.Context) {
	id, ok := idFromParam(c, "id")
	if !ok {
		return
	}
	req := new(formmodel.UpdateFormRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	detail, err := f.formService.Update(c.Request.Context(), ginctx.GetUser(c), id, req)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, detail)
}

// @Summary 保存表单草稿
// @Description 全量替换草稿（目标保存协议根结构）：先按字段字典严格校验（失败返回 FORM_SCHEMA_INVALID，data 携带 issues），再按 draftRevision 乐观锁条件更新并递增口令
// @Accept json
// @Produce json
// @Tags 表单管理
// @Security JWT
// @Param id path int true "表单 ID"
// @Param draft body formmodel.SaveDraftRequest true "草稿口令与协议全文"
// @Success 200 {object} httpx.Response{data=formmodel.SaveDraftResult}
// @Failure 400 {object} httpx.Response "errCode=FORM_SCHEMA_INVALID（data.issues 为 JSON Path 级问题清单）"
// @Failure 409 {object} httpx.Response "errCode=FORM_REVISION_CONFLICT"
// @Router /api/v1/forms/{id}/draft [put]
func (f *FormController) SaveDraft(c *gin.Context) {
	id, ok := idFromParam(c, "id")
	if !ok {
		return
	}
	req := new(formmodel.SaveDraftRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	result, err := f.formService.SaveDraft(c.Request.Context(), ginctx.GetUser(c), id, req)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, result)
}

// @Summary 删除表单
// @Description 软删除表单（配额释放；已发布版本快照保留供历史记录追溯）
// @Produce json
// @Tags 表单管理
// @Security JWT
// @Param id path int true "表单 ID"
// @Success 200 {object} httpx.Response
// @Failure 404 {object} httpx.Response "errCode=FORM_NOT_FOUND"
// @Router /api/v1/forms/{id} [delete]
func (f *FormController) Delete(c *gin.Context) {
	id, ok := idFromParam(c, "id")
	if !ok {
		return
	}
	if err := f.formService.Delete(c.Request.Context(), ginctx.GetUser(c), id); err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, nil)
}

// @Summary 发布表单
// @Description 按草稿当前口令发布：先执行能力白名单校验（白名单外控件返回 FORM_PUBLISH_UNSUPPORTED_FIELD + issues），再按字段字典严格校验；成功生成不可覆盖的发布快照并返回 publishedVersion/schemaRevision 双口令
// @Accept json
// @Produce json
// @Tags 表单管理
// @Security JWT
// @Param id path int true "表单 ID"
// @Param publish body formmodel.PublishRequest true "发布所依据的草稿口令"
// @Success 200 {object} httpx.Response{data=formmodel.PublishResult}
// @Failure 400 {object} httpx.Response "errCode=FORM_PUBLISH_UNSUPPORTED_FIELD/FORM_SCHEMA_INVALID"
// @Failure 409 {object} httpx.Response "errCode=FORM_REVISION_CONFLICT"
// @Router /api/v1/forms/{id}/publish [post]
func (f *FormController) Publish(c *gin.Context) {
	id, ok := idFromParam(c, "id")
	if !ok {
		return
	}
	req := new(formmodel.PublishRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	result, err := f.formService.Publish(c.Request.Context(), ginctx.GetUser(c), id, req)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, result)
}

// @Summary 表单运行时引导
// @Description 按应用编码与表单 ID 返回已发布快照与双口令（publishedVersion/schemaRevision），供最终渲染器初始化；全体成员可读（与菜单同口径），未发布返回 FORM_NOT_PUBLISHED
// @Produce json
// @Tags 表单管理
// @Security JWT
// @Param code path string true "应用编码（app_ 前缀）"
// @Param formId path int true "表单 ID"
// @Success 200 {object} httpx.Response{data=formmodel.FormRuntime}
// @Failure 404 {object} httpx.Response "errCode=FORM_NOT_FOUND/FORM_NOT_PUBLISHED/FORM_APP_INVALID"
// @Router /api/v1/applications/code/{code}/forms/{formId}/runtime [get]
func (f *FormController) GetRuntime(c *gin.Context) {
	// 参数名与既有 /applications/code/:code 系列保持一致（gin 同位置通配符必须同名，
	// 否则路由注册 panic）。
	appCode := c.Param("code")
	if appCode == "" {
		httpx.ResponseFailed(c, http.StatusBadRequest, fmt.Errorf("无效的应用编码"))
		return
	}
	formID, ok := idFromParam(c, "formId")
	if !ok {
		return
	}
	runtime, err := f.formService.GetRuntime(c.Request.Context(), appCode, formID)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, runtime)
}

// @Summary 提交表单记录
// @Description 携带发布双口令提交记录：服务端按对应发布快照逐字段终审（必填/类型/范围/选项命中/隐藏字段携值/未知键），失败返回 FORM_RECORD_INVALID 且 data.fieldErrors 按 widgetName 回填；历史发布版本仍可提交
// @Accept json
// @Produce json
// @Tags 表单管理
// @Security JWT
// @Param record body formmodel.SubmitRecordRequest true "表单 ID、发布双口令与字段值（键=widgetName）"
// @Success 200 {object} httpx.Response{data=formmodel.SubmitRecordResult}
// @Failure 400 {object} httpx.Response "errCode=FORM_RECORD_INVALID（data.fieldErrors 按字段键回填）"
// @Failure 409 {object} httpx.Response "errCode=FORM_VERSION_CONFLICT"
// @Failure 404 {object} httpx.Response "errCode=FORM_NOT_FOUND/FORM_APP_INVALID"
// @Router /api/v1/form-records [post]
func (f *FormController) SubmitRecord(c *gin.Context) {
	req := new(formmodel.SubmitRecordRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	result, err := f.formService.SubmitRecord(c.Request.Context(), ginctx.GetUser(c), req)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.NewResponse(c, http.StatusCreated, result, "提交成功")
}

func (f *FormController) RegisterRoute(api *gin.RouterGroup) {
	api.POST("/forms", f.Create)
	api.GET("/forms", f.List)
	api.GET("/forms/:id", f.Get)
	api.PATCH("/forms/:id", f.Update)
	api.PUT("/forms/:id/draft", f.SaveDraft)
	api.DELETE("/forms/:id", f.Delete)
	api.POST("/forms/:id/publish", f.Publish)
	api.POST("/form-records", f.SubmitRecord)
	// 与 /applications/code/:code 系列同前缀且通配符同名（gin radix tree 要求同
	// 位置同名，静态段 code 优先），鉴权解析为 applications:get，普通成员可读。
	api.GET("/applications/code/:code/forms/:formId/runtime", f.GetRuntime)
}

func (f *FormController) Name() string {
	return "Form"
}
