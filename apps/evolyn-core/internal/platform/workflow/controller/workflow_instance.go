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

// WorkflowInstanceController 最小流程运行时（/workflow-instances、/workflow-tasks）。
// 权限由租户域中间件链执行：URL 首段即 RBAC 资源名（workflow-instances /
// workflow-tasks）；实例级权限由引擎 TaskActor 校验兜底（第 21 章）。
type WorkflowInstanceController struct {
	runtimeService wfservice.RuntimeService
}

func NewWorkflowInstanceController(runtimeService wfservice.RuntimeService) platformcontroller.Controller {
	return &WorkflowInstanceController{runtimeService: runtimeService}
}

// responseError 复用定义域控制器的错误出口逻辑（同包内已存在，直接引用）。

// @Summary 发起流程实例
// @Description 按已发布流程定义发起实例（第 14 章双层幂等）：同业务键运行中实例唯一（WORKFLOW_INSTANCE_ALREADY_RUNNING），携带 idempotencyKey 时重发重放返回同一实例；事务内推进至首个审批节点挂起或直接完成
// @Accept json
// @Produce json
// @Tags 流程管理
// @Security JWT
// @Param instance body model.StartInstanceRequest true "流程编码、业务绑定与可选幂等键/表单绑定"
// @Success 201 {object} httpx.Response{data=model.InstanceDetail}
// @Failure 400 {object} httpx.Response "errCode=WORKFLOW_CODE_INVALID/WORKFLOW_FORM_VERSION_INVALID/WORKFLOW_VERSION_NOT_PUBLISHED"
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN"
// @Failure 404 {object} httpx.Response "errCode=WORKFLOW_NOT_FOUND"
// @Failure 409 {object} httpx.Response "errCode=WORKFLOW_INSTANCE_ALREADY_RUNNING"
// @Router /api/v1/workflow-instances [post]
func (f *WorkflowInstanceController) Start(c *gin.Context) {
	req := new(model.StartInstanceRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	detail, err := f.runtimeService.Start(c.Request.Context(), ginctx.GetUser(c), req)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.NewResponse(c, http.StatusCreated, detail, "发起成功")
}

// @Summary 流程实例详情
// @Description 返回实例绑定关系（定义版本/业务/表单）、节点实例、任务与参与人快照、操作时间线
// @Produce json
// @Tags 流程管理
// @Security JWT
// @Param id path int true "实例 ID"
// @Success 200 {object} httpx.Response{data=model.InstanceDetail}
// @Failure 404 {object} httpx.Response "errCode=WORKFLOW_INSTANCE_NOT_FOUND"
// @Router /api/v1/workflow-instances/{id} [get]
func (f *WorkflowInstanceController) GetInstance(c *gin.Context) {
	instanceID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || instanceID == 0 {
		httpx.ResponseFailed(c, http.StatusBadRequest, errors.New("无效的实例 ID"))
		return
	}
	detail, err := f.runtimeService.GetInstance(c.Request.Context(), ginctx.GetUser(c), uint(instanceID))
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, detail)
}

// @Summary 审批同意
// @Description 同意待办任务（第 13.2 章事务模板）：行锁防双击（重复提交返回 WORKFLOW_TASK_NOT_PENDING）、参与人实例级校验（WORKFLOW_TASK_FORBIDDEN）、节点完成判定（单人/或签首个通过即完成）与同事务推进
// @Accept json
// @Produce json
// @Tags 流程管理
// @Security JWT
// @Param taskId path int true "任务 ID"
// @Param body body model.ApproveTaskRequest true "审批意见（可选）"
// @Success 200 {object} httpx.Response{data=model.ApproveTaskResult}
// @Failure 403 {object} httpx.Response "errCode=WORKFLOW_TASK_FORBIDDEN/FORBIDDEN"
// @Failure 404 {object} httpx.Response "errCode=WORKFLOW_TASK_NOT_FOUND"
// @Failure 409 {object} httpx.Response "errCode=WORKFLOW_TASK_NOT_PENDING/WORKFLOW_INSTANCE_NOT_RUNNING"
// @Router /api/v1/workflow-tasks/{taskId}/approve [post]
func (f *WorkflowInstanceController) Approve(c *gin.Context) {
	taskID, err := strconv.ParseUint(c.Param("taskId"), 10, 64)
	if err != nil || taskID == 0 {
		httpx.ResponseFailed(c, http.StatusBadRequest, errors.New("无效的任务 ID"))
		return
	}
	req := new(model.ApproveTaskRequest)
	if c.Request.Body != nil && c.Request.ContentLength != 0 {
		if err := c.BindJSON(req); err != nil {
			httpx.ResponseFailed(c, http.StatusBadRequest, err)
			return
		}
	}
	req.TaskID = uint(taskID)
	result, err := f.runtimeService.Approve(c.Request.Context(), ginctx.GetUser(c), req)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, result)
}

func (f *WorkflowInstanceController) RegisterRoute(api *gin.RouterGroup) {
	api.POST("/workflow-instances", f.Start)
	api.GET("/workflow-instances/:id", f.GetInstance)
	api.POST("/workflow-tasks/:taskId/approve", f.Approve)
}

func (f *WorkflowInstanceController) Name() string {
	return "WorkflowInstance"
}
