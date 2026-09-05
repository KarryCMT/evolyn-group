package controller

import (
	"errors"
	"net/http"
	"strconv"

	platformcontroller "evolyn/internal/platform/controller"
	"evolyn/internal/platform/ginctx"
	"evolyn/internal/platform/httpx"
	"evolyn/internal/platform/workflow/model"
	wfservice "evolyn/internal/platform/workflow/service"

	"github.com/gin-gonic/gin"
)

// WorkflowTaskController 完整人工任务动作与审批中心查询（Phase 4，
// /workflow-tasks 与 /workflow-instances 审批中心面）。
// 权限由租户域中间件链执行：URL 首段即 RBAC 资源名（workflow-tasks /
// workflow-instances）；实例级权限由引擎 TaskActor 快照校验兜底（第 21 章）。
type WorkflowTaskController struct {
	runtimeService wfservice.RuntimeService
}

func NewWorkflowTaskController(runtimeService wfservice.RuntimeService) platformcontroller.Controller {
	return &WorkflowTaskController{runtimeService: runtimeService}
}

func parseUintParam(c *gin.Context, name string) (uint, bool) {
	value, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil || value == 0 {
		httpx.ResponseFailed(c, http.StatusBadRequest, errors.New("无效的 "+name))
		return 0, false
	}
	return uint(value), true
}

// Reject 驳回任务（V1 terminate 语义）。
//
// @Summary 驳回审批任务
// @Description 驳回待办任务（第 10.2 章 terminate 语义）：任一审批人驳回即节点 REJECTED、其余待办取消、整个实例终止为 REJECTED；重复操作返回 WORKFLOW_TASK_NOT_PENDING，非参与人返回 WORKFLOW_TASK_FORBIDDEN
// @Accept json
// @Produce json
// @Tags 流程管理
// @Security JWT
// @Param taskId path int true "任务 ID"
// @Param body body model.RejectTaskRequest true "审批意见（可选）"
// @Success 200 {object} httpx.Response{data=model.ActionTaskResult}
// @Failure 403 {object} httpx.Response "errCode=WORKFLOW_TASK_FORBIDDEN/FORBIDDEN"
// @Failure 404 {object} httpx.Response "errCode=WORKFLOW_TASK_NOT_FOUND"
// @Failure 409 {object} httpx.Response "errCode=WORKFLOW_TASK_NOT_PENDING/WORKFLOW_INSTANCE_NOT_RUNNING"
// @Router /api/v1/workflow-tasks/{taskId}/reject [post]
func (f *WorkflowTaskController) Reject(c *gin.Context) {
	taskID, ok := parseUintParam(c, "taskId")
	if !ok {
		return
	}
	req := new(model.RejectTaskRequest)
	if c.Request.Body != nil && c.Request.ContentLength != 0 {
		if err := c.BindJSON(req); err != nil {
			httpx.ResponseFailed(c, http.StatusBadRequest, err)
			return
		}
	}
	req.TaskID = taskID
	result, err := f.runtimeService.RejectTask(c.Request.Context(), ginctx.GetUser(c), req)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, result)
}

// ReturnToStarter 退回发起人。
//
// @Summary 退回发起人
// @Description 退回待办任务给发起人修改（第 10.3 章）：不等价于驳回，实例保持 RUNNING 并进入发起人修改等待态；发起人重新提交后从退回节点继续
// @Accept json
// @Produce json
// @Tags 流程管理
// @Security JWT
// @Param taskId path int true "任务 ID"
// @Param body body model.ReturnTaskRequest true "退回原因（可选）"
// @Success 200 {object} httpx.Response{data=model.ActionTaskResult}
// @Failure 403 {object} httpx.Response "errCode=WORKFLOW_TASK_FORBIDDEN/FORBIDDEN"
// @Failure 404 {object} httpx.Response "errCode=WORKFLOW_TASK_NOT_FOUND"
// @Failure 409 {object} httpx.Response "errCode=WORKFLOW_TASK_NOT_PENDING/WORKFLOW_INSTANCE_NOT_RUNNING"
// @Router /api/v1/workflow-tasks/{taskId}/return-to-starter [post]
func (f *WorkflowTaskController) ReturnToStarter(c *gin.Context) {
	taskID, ok := parseUintParam(c, "taskId")
	if !ok {
		return
	}
	req := new(model.ReturnTaskRequest)
	if c.Request.Body != nil && c.Request.ContentLength != 0 {
		if err := c.BindJSON(req); err != nil {
			httpx.ResponseFailed(c, http.StatusBadRequest, err)
			return
		}
	}
	req.TaskID = taskID
	result, err := f.runtimeService.ReturnTask(c.Request.Context(), ginctx.GetUser(c), req)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, result)
}

// Transfer 转办任务。
//
// @Summary 转办任务
// @Description 将待办任务转办给同租户有效成员（第 10.5 章）：原任务关闭为 TRANSFERRED 并另建新任务，历史链可追溯；节点不因转办自动完成
// @Accept json
// @Produce json
// @Tags 流程管理
// @Security JWT
// @Param taskId path int true "任务 ID"
// @Param body body model.TransferTaskRequest true "目标成员 ID（必填）与意见（可选）"
// @Success 200 {object} httpx.Response{data=model.ActionTaskResult}
// @Failure 400 {object} httpx.Response "errCode=WORKFLOW_TASK_NOT_FOUND（参数缺失）"
// @Failure 403 {object} httpx.Response "errCode=WORKFLOW_TASK_FORBIDDEN/FORBIDDEN"
// @Failure 404 {object} httpx.Response "errCode=WORKFLOW_TASK_NOT_FOUND"
// @Failure 409 {object} httpx.Response "errCode=WORKFLOW_TASK_NOT_PENDING/WORKFLOW_INSTANCE_NOT_RUNNING"
// @Router /api/v1/workflow-tasks/{taskId}/transfer [post]
func (f *WorkflowTaskController) Transfer(c *gin.Context) {
	taskID, ok := parseUintParam(c, "taskId")
	if !ok {
		return
	}
	req := new(model.TransferTaskRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	req.TaskID = taskID
	result, err := f.runtimeService.TransferTask(c.Request.Context(), ginctx.GetUser(c), req)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, result)
}

// ListTasks 审批中心任务查询。
//
// @Summary 审批中心任务查询
// @Description 按 scope 查询任务：pending=我的待办（默认）/ completed=我的已办 / cc-to-me=抄送我的（第 20.4 章）；游标分页（id 倒序，limit 默认 20、上限 100）
// @Produce json
// @Tags 流程管理
// @Security JWT
// @Param scope query string false "pending/completed/cc-to-me，默认 pending"
// @Param formCode query string false "流程型表单编码（仅 pending 可用）"
// @Param limit query int false "分页大小（默认 20，上限 100）"
// @Param cursor query string false "游标（上一页 nextCursor）"
// @Success 200 {object} httpx.Response{data=model.TaskPage}
// @Failure 400 {object} httpx.Response "errCode=WORKFLOW_CODE_INVALID（scope/游标非法）"
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN"
// @Router /api/v1/workflow-tasks [get]
func (f *WorkflowTaskController) ListTasks(c *gin.Context) {
	query := model.ListTasksQuery{
		Scope:    c.Query("scope"),
		Limit:    atoiDefault(c.Query("limit")),
		Cursor:   c.Query("cursor"),
		FormCode: c.Query("formCode"),
	}
	page, err := f.runtimeService.ListTasks(c.Request.Context(), ginctx.GetUser(c), query)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, page)
}

// PendingSummary 返回我的待办总量及流程表单分组数量，供流程菜单显示真实徽标。
//
// @Summary 待办菜单摘要
// @Description 当前成员待办总数与按流程型表单的聚合数
// @Produce json
// @Tags 流程管理
// @Security JWT
// @Success 200 {object} httpx.Response{data=model.PendingTaskSummary}
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN"
// @Router /api/v1/workflow-tasks/summary [get]
func (f *WorkflowTaskController) PendingSummary(c *gin.Context) {
	summary, err := f.runtimeService.PendingTaskSummary(c.Request.Context(), ginctx.GetUser(c))
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, summary)
}

// GetTask 任务详情上下文。
//
// @Summary 任务详情
// @Description 返回审批详情上下文（第 4 章协议）：任务与参与人快照、实例绑定、节点字段权限、允许动作、表单冻结快照与业务数据、操作时间线；仅任务参与人或发起人可读
// @Produce json
// @Tags 流程管理
// @Security JWT
// @Param taskId path int true "任务 ID"
// @Success 200 {object} httpx.Response{data=model.TaskDetail}
// @Failure 403 {object} httpx.Response "errCode=WORKFLOW_TASK_FORBIDDEN/FORBIDDEN"
// @Failure 404 {object} httpx.Response "errCode=WORKFLOW_TASK_NOT_FOUND/WORKFLOW_INSTANCE_NOT_FOUND"
// @Router /api/v1/workflow-tasks/{taskId} [get]
func (f *WorkflowTaskController) GetTask(c *gin.Context) {
	taskID, ok := parseUintParam(c, "taskId")
	if !ok {
		return
	}
	detail, err := f.runtimeService.GetTask(c.Request.Context(), ginctx.GetUser(c), taskID)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, detail)
}

// ListInstances 审批中心实例查询。
//
// @Summary 审批中心实例查询
// @Description 我发起的流程实例（scope=started-by-me，第 20.4 章）；游标分页（id 倒序）
// @Produce json
// @Tags 流程管理
// @Security JWT
// @Param scope query string false "started-by-me（当前唯一取值）"
// @Param limit query int false "分页大小（默认 20，上限 100）"
// @Param cursor query string false "游标（上一页 nextCursor）"
// @Success 200 {object} httpx.Response{data=model.InstancePage}
// @Failure 400 {object} httpx.Response "errCode=WORKFLOW_CODE_INVALID（scope/游标非法）"
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN"
// @Router /api/v1/workflow-instances [get]
func (f *WorkflowTaskController) ListInstances(c *gin.Context) {
	query := model.ListInstancesQuery{
		Scope:  c.Query("scope"),
		Limit:  atoiDefault(c.Query("limit")),
		Cursor: c.Query("cursor"),
	}
	page, err := f.runtimeService.ListInstances(c.Request.Context(), ginctx.GetUser(c), query)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, page)
}

// Withdraw 发起人撤回。
//
// @Summary 撤回流程实例
// @Description 发起人撤回（第 10.4 章）：仅当流程尚不存在任何已完成的人工审批任务时允许（否则 WORKFLOW_ACTION_NOT_ALLOWED）；撤回后实例 CANCELLED、全部待办取消
// @Accept json
// @Produce json
// @Tags 流程管理
// @Security JWT
// @Param id path int true "实例 ID"
// @Param body body model.InstanceActionRequest false "撤回说明（可选）"
// @Success 200 {object} httpx.Response{data=model.ActionTaskResult}
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN（非发起人）"
// @Failure 404 {object} httpx.Response "errCode=WORKFLOW_INSTANCE_NOT_FOUND"
// @Failure 409 {object} httpx.Response "errCode=WORKFLOW_ACTION_NOT_ALLOWED/WORKFLOW_INSTANCE_NOT_RUNNING"
// @Router /api/v1/workflow-instances/{id}/withdraw [post]
func (f *WorkflowTaskController) Withdraw(c *gin.Context) {
	instanceID, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	req := new(model.InstanceActionRequest)
	if c.Request.Body != nil && c.Request.ContentLength != 0 {
		if err := c.BindJSON(req); err != nil {
			httpx.ResponseFailed(c, http.StatusBadRequest, err)
			return
		}
	}
	result, err := f.runtimeService.WithdrawInstance(c.Request.Context(), ginctx.GetUser(c), instanceID, req)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, result)
}

// Terminate 管理员终止。
//
// @Summary 终止流程实例
// @Description 管理员终止运行中实例（独立 workflow-instances:update 权限，不等价于撤回）：实例 CANCELLED、全部待办取消；历史完整保留
// @Accept json
// @Produce json
// @Tags 流程管理
// @Security JWT
// @Param id path int true "实例 ID"
// @Param body body model.InstanceActionRequest false "终止说明（可选）"
// @Success 200 {object} httpx.Response{data=model.ActionTaskResult}
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN"
// @Failure 404 {object} httpx.Response "errCode=WORKFLOW_INSTANCE_NOT_FOUND"
// @Failure 409 {object} httpx.Response "errCode=WORKFLOW_INSTANCE_NOT_RUNNING"
// @Router /api/v1/workflow-instances/{id}/terminate [post]
func (f *WorkflowTaskController) Terminate(c *gin.Context) {
	instanceID, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	req := new(model.InstanceActionRequest)
	if c.Request.Body != nil && c.Request.ContentLength != 0 {
		if err := c.BindJSON(req); err != nil {
			httpx.ResponseFailed(c, http.StatusBadRequest, err)
			return
		}
	}
	result, err := f.runtimeService.TerminateInstance(c.Request.Context(), ginctx.GetUser(c), instanceID, req)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, result)
}

// Resubmit 发起人重新提交。
//
// @Summary 发起人重新提交
// @Description 退回修改后重新提交（第 10.3 章）：values 携带修改后的表单字段值（可选，按发起时冻结的表单快照整体校验后同事务写回），流程从退回节点继续；仅发起人可操作
// @Accept json
// @Produce json
// @Tags 流程管理
// @Security JWT
// @Param id path int true "实例 ID"
// @Param body body model.ResubmitInstanceRequest false "修改后的表单字段值（可选）"
// @Success 200 {object} httpx.Response{data=model.ActionTaskResult}
// @Failure 400 {object} httpx.Response "errCode=FORM_RECORD_INVALID/WORKFLOW_FORM_FIELD_FORBIDDEN"
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN（非发起人）"
// @Failure 404 {object} httpx.Response "errCode=WORKFLOW_INSTANCE_NOT_FOUND"
// @Failure 409 {object} httpx.Response "errCode=WORKFLOW_ACTION_NOT_ALLOWED/WORKFLOW_INSTANCE_NOT_RUNNING"
// @Router /api/v1/workflow-instances/{id}/resubmit [post]
func (f *WorkflowTaskController) Resubmit(c *gin.Context) {
	instanceID, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	req := new(model.ResubmitInstanceRequest)
	if c.Request.Body != nil && c.Request.ContentLength != 0 {
		if err := c.BindJSON(req); err != nil {
			httpx.ResponseFailed(c, http.StatusBadRequest, err)
			return
		}
	}
	result, err := f.runtimeService.ResubmitInstance(c.Request.Context(), ginctx.GetUser(c), instanceID, req)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, result)
}

// atoiDefault 查询参数整数兜底。
func atoiDefault(raw string) int {
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return value
}

func (f *WorkflowTaskController) RegisterRoute(api *gin.RouterGroup) {
	api.GET("/workflow-tasks", f.ListTasks)
	api.GET("/workflow-tasks/summary", f.PendingSummary)
	api.GET("/workflow-tasks/:taskId", f.GetTask)
	api.POST("/workflow-tasks/:taskId/reject", f.Reject)
	api.POST("/workflow-tasks/:taskId/return-to-starter", f.ReturnToStarter)
	api.POST("/workflow-tasks/:taskId/transfer", f.Transfer)
	api.GET("/workflow-instances", f.ListInstances)
	api.POST("/workflow-instances/:id/withdraw", f.Withdraw)
	api.POST("/workflow-instances/:id/terminate", f.Terminate)
	api.POST("/workflow-instances/:id/resubmit", f.Resubmit)
}

func (f *WorkflowTaskController) Name() string {
	return "WorkflowTask"
}
