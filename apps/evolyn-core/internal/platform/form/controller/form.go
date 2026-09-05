// Package controller 表单资产域 HTTP 接口：解析请求、取当前成员、返回 httpx 统信封。
// 权限由租户域中间件链执行；URL 首段即 RBAC 资源名（/forms → forms、
// /form-records → form-records、bootstrap 挂 /applications 前缀 → applications:get）。
package controller

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

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

// formCodeFromParam 解析稳定表单公开编码，非法直接回 400。
func formCodeFromParam(c *gin.Context, name string) (string, bool) {
	code := strings.TrimSpace(c.Param(name))
	if !strings.HasPrefix(code, "form_") || len(code) <= len("form_") {
		httpx.ResponseFailed(c, http.StatusBadRequest, fmt.Errorf("无效的表单编码：%s", code))
		return "", false
	}
	return code, true
}

// @Summary 创建表单
// @Description 在当前租户创建表单资产（名称必填，归属指定应用）；事务内完成 forms 配额校验，草稿初始化为空目标协议文档。parentEntryCode 可选：传入时菜单节点挂到该分组下（须为同应用分组节点编码），否则挂应用根级
// @Accept json
// @Produce json
// @Tags 表单管理
// @Security JWT
// @Param form body formmodel.CreateFormRequest true "应用 ID、表单名称、表单类型与可选父分组编码"
// @Success 201 {object} httpx.Response{data=formmodel.FormDetail}
// @Failure 400 {object} httpx.Response "errCode=FORM_NAME_INVALID/FORM_TYPE_INVALID/FORM_APP_INVALID/APP_MENU_PARENT_INVALID"
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
// @Description 按公开编码查询表单详情，含表单类型、目标协议草稿全文、草稿修订口令（draftRevision）与最新发布号
// @Produce json
// @Tags 表单管理
// @Security JWT
// @Param code path string true "表单编码（form_ 前缀）"
// @Success 200 {object} httpx.Response{data=formmodel.FormDetail}
// @Failure 404 {object} httpx.Response "errCode=FORM_NOT_FOUND"
// @Router /api/v1/forms/{code} [get]
func (f *FormController) Get(c *gin.Context) {
	code, ok := formCodeFromParam(c, "code")
	if !ok {
		return
	}
	detail, err := f.formService.Get(c.Request.Context(), ginctx.GetUser(c), code)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, detail)
}

// @Summary 更新表单
// @Description 白名单字段更新：名称（trim 后 1–128 字符）与图标/颜色稳定键（空串清空，经菜单维护端口同事务同步到节点展示属性，ADR-011）
// @Accept json
// @Produce json
// @Tags 表单管理
// @Security JWT
// @Param code path string true "表单编码（form_ 前缀）"
// @Param form body formmodel.UpdateFormRequest true "更新字段（仅白名单，指针区分未提交）"
// @Success 200 {object} httpx.Response{data=formmodel.FormDetail}
// @Failure 400 {object} httpx.Response "errCode=FORM_NAME_INVALID/FORM_ICON_INVALID"
// @Failure 404 {object} httpx.Response "errCode=FORM_NOT_FOUND"
// @Router /api/v1/forms/{code} [patch]
func (f *FormController) Update(c *gin.Context) {
	code, ok := formCodeFromParam(c, "code")
	if !ok {
		return
	}
	req := new(formmodel.UpdateFormRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	detail, err := f.formService.Update(c.Request.Context(), ginctx.GetUser(c), code, req)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, detail)
}

// @Summary 保存表单草稿
// @Description 全量替换草稿（目标保存协议根结构，v4 支持子表单权限、冻结列与移动端展示配置，并保留默认列布局和 multitab 标签页布局）：先按 protocolVersion 与字段字典严格校验（失败返回 FORM_SCHEMA_INVALID，data 携带 issues），再按 draftRevision 乐观锁条件更新并递增口令
// @Accept json
// @Produce json
// @Tags 表单管理
// @Security JWT
// @Param code path string true "表单编码（form_ 前缀）"
// @Param draft body formmodel.SaveDraftRequest true "草稿口令与协议全文"
// @Success 200 {object} httpx.Response{data=formmodel.SaveDraftResult}
// @Failure 400 {object} httpx.Response "errCode=FORM_SCHEMA_INVALID（data.issues 为 JSON Path 级问题清单）"
// @Failure 409 {object} httpx.Response "errCode=FORM_REVISION_CONFLICT"
// @Router /api/v1/forms/{code}/draft [put]
func (f *FormController) SaveDraft(c *gin.Context) {
	code, ok := formCodeFromParam(c, "code")
	if !ok {
		return
	}
	req := new(formmodel.SaveDraftRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	result, err := f.formService.SaveDraft(c.Request.Context(), ginctx.GetUser(c), code, req)
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
// @Param code path string true "表单编码（form_ 前缀）"
// @Success 200 {object} httpx.Response
// @Failure 404 {object} httpx.Response "errCode=FORM_NOT_FOUND"
// @Router /api/v1/forms/{code} [delete]
func (f *FormController) Delete(c *gin.Context) {
	code, ok := formCodeFromParam(c, "code")
	if !ok {
		return
	}
	if err := f.formService.Delete(c.Request.Context(), ginctx.GetUser(c), code); err != nil {
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
// @Param code path string true "表单编码（form_ 前缀）"
// @Param publish body formmodel.PublishRequest true "发布所依据的草稿口令"
// @Success 200 {object} httpx.Response{data=formmodel.PublishResult}
// @Failure 400 {object} httpx.Response "errCode=FORM_PUBLISH_UNSUPPORTED_FIELD/FORM_SCHEMA_INVALID"
// @Failure 409 {object} httpx.Response "errCode=FORM_REVISION_CONFLICT"
// @Router /api/v1/forms/{code}/publish [post]
func (f *FormController) Publish(c *gin.Context) {
	code, ok := formCodeFromParam(c, "code")
	if !ok {
		return
	}
	req := new(formmodel.PublishRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	result, err := f.formService.Publish(c.Request.Context(), ginctx.GetUser(c), code, req)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, result)
}

// @Summary 表单运行时引导
// @Description 按应用编码与表单编码返回已发布快照与双口令（publishedVersion/schemaRevision），供最终渲染器初始化；全体成员可读（与菜单同口径），未发布返回 FORM_NOT_PUBLISHED
// @Produce json
// @Tags 表单管理
// @Security JWT
// @Param code path string true "应用编码（app_ 前缀）"
// @Param formCode path string true "表单编码（form_ 前缀）"
// @Success 200 {object} httpx.Response{data=formmodel.FormRuntime}
// @Failure 404 {object} httpx.Response "errCode=FORM_NOT_FOUND/FORM_NOT_PUBLISHED/FORM_APP_INVALID"
// @Router /api/v1/applications/code/{code}/forms/{formCode}/runtime [get]
func (f *FormController) GetRuntime(c *gin.Context) {
	// 参数名与既有 /applications/code/:code 系列保持一致（gin 同位置通配符必须同名，
	// 否则路由注册 panic）。
	appCode := c.Param("code")
	if appCode == "" {
		httpx.ResponseFailed(c, http.StatusBadRequest, fmt.Errorf("无效的应用编码"))
		return
	}
	formCode, ok := formCodeFromParam(c, "formCode")
	if !ok {
		return
	}
	runtime, err := f.formService.GetRuntime(c.Request.Context(), ginctx.GetUser(c), appCode, formCode)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, runtime)
}

// @Summary 提交表单记录
// @Description 携带应用/菜单上下文、客户端幂等键、发布双口令与字段 {data,visible} 快照提交记录：服务端按对应发布快照逐字段终审（必填/类型/范围/选项命中/可见状态/隐藏字段携值/未知键），失败返回 FORM_RECORD_INVALID 且 data.fieldErrors 按 widgetName 回填；历史发布版本仍可提交
// @Accept json
// @Produce json
// @Tags 表单管理
// @Security JWT
// @Param record body formmodel.SubmitRecordRequest true "提交上下文、发布双口令、幂等键与字段快照（键=widgetName，值={data?,visible}）"
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

// @Summary 查询表单记录
// @Description 仅接收受控 Query DSL JSON（请求体承载完整文档，避免复杂筛选条件超出 URL 长度上限）；筛选字段必须来自当前发布快照 fieldMappings，或系统字段命名空间 sys.submittedBy（提交人成员 ID，eq/neq/in/notIn）/sys.submittedAt、sys.updatedAt（秒级时间，比较与 between）；排序仅开放系统字段（sorts 最多 3 个，asc/desc）。行级 view 范围与字段矩阵仍由服务端合并裁决，禁止 JSONB 路径、物理列名或 SQL 输入。出网含系统字段 submittedByName（提交时固化的展示名快照）与 updatedAt。URL 门的鉴权动词经 request.go 特判归一化为 get（form-records:view），不落入 POST→create 的提交门
// @Accept json
// @Produce json
// @Tags 表单管理
// @Security JWT
// @Param code path string true "表单编码（form_ 前缀）"
// @Param query body formmodel.RecordQueryDocument false "Query DSL v1 JSON；缺省视为无筛选默认分页查询"
// @Success 200 {object} httpx.Response{data=formmodel.FormRecordPage}
// @Failure 400 {object} httpx.Response "errCode=FORM_RECORD_QUERY_INVALID"
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN/FORM_PERMISSION_DENIED"
// @Router /api/v1/forms/{code}/records [post]
func (f *FormController) ListRecords(c *gin.Context) {
	code, ok := formCodeFromParam(c, "code")
	if !ok {
		return
	}
	// Query DSL 以 JSON 请求体上送；空 body（io.EOF）与原 GET ?query= 缺省
	// 同语义，视为无筛选的默认分页查询
	query := formmodel.RecordQueryDocument{}
	if err := c.BindJSON(&query); err != nil && !errors.Is(err, io.EOF) {
		httpx.ResponseFailed(c, http.StatusBadRequest, fmt.Errorf("无效的 Query DSL"))
		return
	}
	page, err := f.formService.ListRecords(c.Request.Context(), ginctx.GetUser(c), code, query)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, page)
}

// @Summary 切换表单类型
// @Description standard↔workflow 互转（ADR-011）：流程表单切标准后原流程数据保留，仅不可再发起流程；草稿与发布快照不受影响；目标类型与当前相同返回 FORM_TYPE_UNCHANGED
// @Accept json
// @Produce json
// @Tags 表单管理
// @Security JWT
// @Param code path string true "表单编码（form_ 前缀）"
// @Param body body formmodel.SwitchFormTypeRequest true "目标表单类型"
// @Success 200 {object} httpx.Response{data=formmodel.FormDetail}
// @Failure 400 {object} httpx.Response "errCode=FORM_TYPE_INVALID/FORM_TYPE_UNCHANGED"
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN（须 form-actions:switch-type 动作授权）"
// @Failure 404 {object} httpx.Response "errCode=FORM_NOT_FOUND"
// @Router /api/v1/forms/{code}/switch-type [post]
func (f *FormController) SwitchType(c *gin.Context) {
	code, ok := formCodeFromParam(c, "code")
	if !ok {
		return
	}
	req := new(formmodel.SwitchFormTypeRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	detail, err := f.formService.SwitchType(c.Request.Context(), ginctx.GetUser(c), code, req)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, detail)
}

// @Summary 复制表单
// @Description 复制表单资产（ADR-011）：targetApplicationId 为空或等于源应用走 copy-in-app 动作，跨应用走 copy-cross-app 动作；复制草稿全文与表单类型（不复制发布快照与记录），名称追加「（副本）」，事务内占目标应用配额并挂目标应用菜单
// @Accept json
// @Produce json
// @Tags 表单管理
// @Security JWT
// @Param code path string true "表单编码（form_ 前缀）"
// @Param body body formmodel.CopyFormRequest true "目标应用与可选目标分组节点编码"
// @Success 201 {object} httpx.Response{data=formmodel.FormDetail}
// @Failure 400 {object} httpx.Response "errCode=FORM_APP_INVALID/APP_MENU_PARENT_INVALID"
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN（须 form-actions:copy-in-app / form-actions:copy-cross-app 动作授权）/QUOTA_EXCEEDED"
// @Failure 404 {object} httpx.Response "errCode=FORM_NOT_FOUND"
// @Router /api/v1/forms/{code}/copy [post]
func (f *FormController) Copy(c *gin.Context) {
	code, ok := formCodeFromParam(c, "code")
	if !ok {
		return
	}
	req := new(formmodel.CopyFormRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	detail, err := f.formService.Copy(c.Request.Context(), ginctx.GetUser(c), code, req)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.NewResponse(c, http.StatusCreated, detail, "复制成功")
}

// @Summary 查看表单引用视图
// @Description 跨应用反查引用指定表单的菜单节点（ADR-011）：返回应用编码/名称与节点编码/名称；只读诊断信息，持 forms:get 即可读取
// @Produce json
// @Tags 表单管理
// @Security JWT
// @Param code path string true "表单编码（form_ 前缀）"
// @Success 200 {object} httpx.Response{data=[]service.FormReference}
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN"
// @Failure 404 {object} httpx.Response "errCode=FORM_NOT_FOUND"
// @Router /api/v1/forms/{code}/references [get]
func (f *FormController) ListReferences(c *gin.Context) {
	code, ok := formCodeFromParam(c, "code")
	if !ok {
		return
	}
	references, err := f.formService.ListReferences(c.Request.Context(), ginctx.GetUser(c), code)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, references)
}

func (f *FormController) RegisterRoute(api *gin.RouterGroup) {
	api.POST("/forms", f.Create)
	api.GET("/forms", f.List)
	api.GET("/forms/:code", f.Get)
	api.PATCH("/forms/:code", f.Update)
	api.PUT("/forms/:code/draft", f.SaveDraft)
	api.DELETE("/forms/:code", f.Delete)
	api.POST("/forms/:code/publish", f.Publish)
	// ADR-011：切换类型/复制（URL 门 POST→forms:create，动作键由 Service
	// 按 form-actions:* 复核）与引用视图（GET→forms:get）
	api.POST("/forms/:code/switch-type", f.SwitchType)
	api.POST("/forms/:code/copy", f.Copy)
	api.GET("/forms/:code/references", f.ListReferences)
	// 记录查询以 POST body 承载完整 Query DSL（复杂筛选会超出 URL 长度
	// 上限）；URL 门动词由 request.go 特判归一化为 get → form-records:view
	api.POST("/forms/:code/records", f.ListRecords)
	api.POST("/form-records", f.SubmitRecord)
	// 与 /applications/code/:code 系列同前缀且通配符同名（gin radix tree 要求同
	// 位置同名，静态段 code 优先），鉴权解析为 applications:get，普通成员可读。
	api.GET("/applications/code/:code/forms/:formCode/runtime", f.GetRuntime)
}

func (f *FormController) Name() string {
	return "Form"
}
