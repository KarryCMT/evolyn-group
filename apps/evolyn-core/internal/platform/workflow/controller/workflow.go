// Package controller 流程引擎 HTTP 接口：解析请求、取当前成员、返回 httpx 统一
// 信封。权限由租户域中间件链执行；URL 首段即 RBAC 资源名（/workflows →
// workflows，POST→create / GET→get / PUT→update / PATCH→patch / DELETE→delete；
// publish 复用 create 动词，与表单域同口径）。
package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	platformcontroller "evolyn/internal/platform/controller"
	"evolyn/internal/platform/ginctx"
	"evolyn/internal/platform/httpx"
	wfapp "evolyn/internal/platform/workflow"
	"evolyn/internal/platform/workflow/model"
	wfservice "evolyn/internal/platform/workflow/service"

	"github.com/gin-gonic/gin"
)

// WorkflowController 流程定义（/workflows）
type WorkflowController struct {
	definitionService wfservice.DefinitionService
}

func NewWorkflowController(definitionService wfservice.DefinitionService) platformcontroller.Controller {
	return &WorkflowController{definitionService: definitionService}
}

// responseError 错误统一出口（ADR-008 脱敏）：BizError 按自带状态码出网
// （含 data 负载的码如 WORKFLOW_DEFINITION_INVALID 原样透传）；未分类错误一律 500 脱敏。
func responseError(c *gin.Context, err error) {
	var biz *httpx.BizError
	if errors.As(err, &biz) && biz.HTTP != 0 {
		httpx.ResponseFailed(c, biz.HTTP, err)
		return
	}
	httpx.ResponseFailed(c, http.StatusInternalServerError, err)
}

// workflowCodeFromParam 解析稳定流程公开编码，非法直接回 400。
func workflowCodeFromParam(c *gin.Context) (string, bool) {
	code := strings.TrimSpace(c.Param("code"))
	if !strings.HasPrefix(code, "wf_") || len(code) <= len("wf_") {
		httpx.ResponseFailed(c, http.StatusBadRequest, wfapp.ErrWorkflowCodeInvalid)
		return "", false
	}
	return code, true
}

// @Summary 创建流程定义
// @Description 在当前租户创建流程定义（名称必填），草稿初始化为最小合法 DSL（start → end），开箱即可发布；formCode 可选绑定流程型表单（一条表单租户内至多一条定义）
// @Accept json
// @Produce json
// @Tags 流程管理
// @Security JWT
// @Param workflow body model.CreateWorkflowRequest true "流程名称、可选描述与可选绑定表单编码"
// @Success 201 {object} httpx.Response{data=model.WorkflowDetail}
// @Failure 400 {object} httpx.Response "errCode=WORKFLOW_NAME_INVALID/WORKFLOW_DESCRIPTION_INVALID/WORKFLOW_FORM_CODE_INVALID"
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN"
// @Router /api/v1/workflows [post]
func (f *WorkflowController) Create(c *gin.Context) {
	req := new(model.CreateWorkflowRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	detail, err := f.definitionService.Create(c.Request.Context(), ginctx.GetUser(c), req)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.NewResponse(c, http.StatusCreated, detail, "创建成功")
}

// @Summary 流程定义列表
// @Description 租户内流程定义游标分页（id 倒序，新定义靠前）；cursor 不透明值原样回传
// @Produce json
// @Tags 流程管理
// @Security JWT
// @Param limit query int false "每页数量，默认 20，上限 100"
// @Param cursor query string false "分页游标（上一页 nextCursor 原样回传）"
// @Param formCode query string false "按绑定表单编码精确过滤（流程设计页定位定义）"
// @Success 200 {object} httpx.Response{data=model.WorkflowPage}
// @Router /api/v1/workflows [get]
func (f *WorkflowController) List(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	page, err := f.definitionService.List(c.Request.Context(), ginctx.GetUser(c), model.ListWorkflowsQuery{
		Limit:    limit,
		Cursor:   c.Query("cursor"),
		FormCode: strings.TrimSpace(c.Query("formCode")),
	})
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, page)
}

// @Summary 流程定义详情
// @Description 按公开编码查询定义详情，含 DSL v1 草稿全文与修订口令（draftRevision）与最新发布号
// @Produce json
// @Tags 流程管理
// @Security JWT
// @Param code path string true "流程编码（wf_ 前缀）"
// @Success 200 {object} httpx.Response{data=model.WorkflowDetail}
// @Failure 404 {object} httpx.Response "errCode=WORKFLOW_NOT_FOUND"
// @Router /api/v1/workflows/{code} [get]
func (f *WorkflowController) Get(c *gin.Context) {
	code, ok := workflowCodeFromParam(c)
	if !ok {
		return
	}
	detail, err := f.definitionService.Get(c.Request.Context(), ginctx.GetUser(c), code)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, detail)
}

// @Summary 更新流程定义元信息
// @Description 白名单字段更新：名称（trim 后 1–128 字符）与描述（≤512 字符）；不触碰草稿与发布指针
// @Accept json
// @Produce json
// @Tags 流程管理
// @Security JWT
// @Param code path string true "流程编码（wf_ 前缀）"
// @Param workflow body model.UpdateWorkflowRequest true "更新字段（仅白名单，指针区分未提交）"
// @Success 200 {object} httpx.Response{data=model.WorkflowDetail}
// @Failure 400 {object} httpx.Response "errCode=WORKFLOW_NAME_INVALID/WORKFLOW_DESCRIPTION_INVALID"
// @Failure 404 {object} httpx.Response "errCode=WORKFLOW_NOT_FOUND"
// @Router /api/v1/workflows/{code} [patch]
func (f *WorkflowController) Update(c *gin.Context) {
	code, ok := workflowCodeFromParam(c)
	if !ok {
		return
	}
	req := new(model.UpdateWorkflowRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	detail, err := f.definitionService.Update(c.Request.Context(), ginctx.GetUser(c), code, req)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, detail)
}

// @Summary 保存流程草稿
// @Description 全量替换草稿（Workflow DSL v1 全文档）：先经引擎严格校验器校验（失败返回 WORKFLOW_DEFINITION_INVALID，data.issues 为协议级问题清单），再按 draftRevision 乐观锁条件更新并递增口令
// @Accept json
// @Produce json
// @Tags 流程管理
// @Security JWT
// @Param code path string true "流程编码（wf_ 前缀）"
// @Param draft body model.SaveDraftRequest true "草稿口令与 DSL 全文"
// @Success 200 {object} httpx.Response{data=model.SaveDraftResult}
// @Failure 400 {object} httpx.Response "errCode=WORKFLOW_DEFINITION_INVALID（data.issues 为 {path,code,message} 清单）"
// @Failure 409 {object} httpx.Response "errCode=WORKFLOW_REVISION_CONFLICT"
// @Router /api/v1/workflows/{code} [put]
func (f *WorkflowController) SaveDraft(c *gin.Context) {
	code, ok := workflowCodeFromParam(c)
	if !ok {
		return
	}
	req := new(model.SaveDraftRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	result, err := f.definitionService.SaveDraft(c.Request.Context(), ginctx.GetUser(c), code, req)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, result)
}

// @Summary 删除流程定义
// @Description 软删除流程定义（仅允许无运行中实例；发布版本快照与运行态历史保留供追溯）
// @Produce json
// @Tags 流程管理
// @Security JWT
// @Param code path string true "流程编码（wf_ 前缀）"
// @Success 200 {object} httpx.Response
// @Failure 404 {object} httpx.Response "errCode=WORKFLOW_NOT_FOUND"
// @Router /api/v1/workflows/{code} [delete]
func (f *WorkflowController) Delete(c *gin.Context) {
	code, ok := workflowCodeFromParam(c)
	if !ok {
		return
	}
	if err := f.definitionService.Delete(c.Request.Context(), ginctx.GetUser(c), code); err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, nil)
}

// @Summary 发布流程定义
// @Description 按草稿当前口令发布：先经 DSL 严格校验与 Expr 预编译（失败返回 WORKFLOW_DEFINITION_INVALID + issues），成功在事务内冻结不可变发布快照并递增 versionNo；运行实例（Phase 2）固定绑定发布版本
// @Accept json
// @Produce json
// @Tags 流程管理
// @Security JWT
// @Param code path string true "流程编码（wf_ 前缀）"
// @Param publish body model.PublishRequest true "发布所依据的草稿口令"
// @Success 200 {object} httpx.Response{data=model.PublishResult}
// @Failure 400 {object} httpx.Response "errCode=WORKFLOW_DEFINITION_INVALID"
// @Failure 409 {object} httpx.Response "errCode=WORKFLOW_REVISION_CONFLICT"
// @Failure 404 {object} httpx.Response "errCode=WORKFLOW_NOT_FOUND"
// @Router /api/v1/workflows/{code}/publish [post]
func (f *WorkflowController) Publish(c *gin.Context) {
	code, ok := workflowCodeFromParam(c)
	if !ok {
		return
	}
	req := new(model.PublishRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	result, err := f.definitionService.Publish(c.Request.Context(), ginctx.GetUser(c), code, req)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, result)
}

// @Summary 流程版本列表
// @Description 指定定义的发布版本列表（versionNo 降序）；快照全文经版本详情接口获取
// @Produce json
// @Tags 流程管理
// @Security JWT
// @Param code path string true "流程编码（wf_ 前缀）"
// @Success 200 {object} httpx.Response{data=[]model.VersionSummary}
// @Failure 404 {object} httpx.Response "errCode=WORKFLOW_NOT_FOUND"
// @Router /api/v1/workflows/{code}/versions [get]
func (f *WorkflowController) ListVersions(c *gin.Context) {
	code, ok := workflowCodeFromParam(c)
	if !ok {
		return
	}
	versions, err := f.definitionService.ListVersions(c.Request.Context(), ginctx.GetUser(c), code)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, versions)
}

// @Summary 流程版本详情
// @Description 按版本号读取不可变发布快照全文（历史版本均可读；DSL 只读预览与运行期事实源）
// @Produce json
// @Tags 流程管理
// @Security JWT
// @Param code path string true "流程编码（wf_ 前缀）"
// @Param versionNo path int true "发布版本号"
// @Success 200 {object} httpx.Response{data=model.VersionDetail}
// @Failure 404 {object} httpx.Response "errCode=WORKFLOW_NOT_FOUND/WORKFLOW_VERSION_NOT_FOUND"
// @Router /api/v1/workflows/{code}/versions/{versionNo} [get]
func (f *WorkflowController) GetVersion(c *gin.Context) {
	code, ok := workflowCodeFromParam(c)
	if !ok {
		return
	}
	versionNo, err := strconv.Atoi(c.Param("versionNo"))
	if err != nil || versionNo <= 0 {
		httpx.ResponseFailed(c, http.StatusBadRequest, fmt.Errorf("无效的版本号：%s", c.Param("versionNo")))
		return
	}
	detail, err := f.definitionService.GetVersion(c.Request.Context(), ginctx.GetUser(c), code, versionNo)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, detail)
}

func (f *WorkflowController) RegisterRoute(api *gin.RouterGroup) {
	api.POST("/workflows", f.Create)
	api.GET("/workflows", f.List)
	api.GET("/workflows/:code", f.Get)
	api.PATCH("/workflows/:code", f.Update)
	api.PUT("/workflows/:code", f.SaveDraft)
	api.DELETE("/workflows/:code", f.Delete)
	api.POST("/workflows/:code/publish", f.Publish)
	api.GET("/workflows/:code/versions", f.ListVersions)
	api.GET("/workflows/:code/versions/:versionNo", f.GetVersion)
}

func (f *WorkflowController) Name() string {
	return "Workflow"
}
