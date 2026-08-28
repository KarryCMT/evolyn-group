// Package controller 消息中心域 HTTP 接口：/notifications 挂租户域链
// （Authentication → Tenant → TenantStatus → Authorization）。资源级权限由
// 中间件链执行：GET 集合路径解析为 list、GET /unread-summary 解析为 get
// （对应 notifications:view），PUT /:id/read 与 PUT /read-all 解析为 update
// （对应 notifications:update）；数据范围（仅本人收件箱）由 Service/Repository
// 的 tenant_id + member_id 双条件兜底。
package controller

import (
	"fmt"
	"net/http"
	"strconv"

	"evolyn/internal/contextx"
	platformcontroller "evolyn/internal/platform/controller"
	"evolyn/internal/platform/ginctx"
	"evolyn/internal/platform/httpx"
	"evolyn/internal/platform/notification/model"
	"evolyn/internal/platform/notification/service"

	"github.com/gin-gonic/gin"
)

// NotificationController 成员收件箱（/notifications）
type NotificationController struct {
	inboxService service.InboxService
}

// NewNotificationController 收件箱控制器工厂
func NewNotificationController(inboxService service.InboxService) platformcontroller.Controller {
	return &NotificationController{inboxService: inboxService}
}

func (n *NotificationController) Name() string {
	return "消息中心"
}

// swagger 出网类型别名（本包不直接构造，仅为文档解析提供定位）
type (
	UnreadSummaryView  = model.UnreadSummaryView
	CategoryUnreadView = model.CategoryUnreadView
	InboxPageView      = model.InboxPageView
	InboxItemView      = model.InboxItemView
	ReadAllRequest     = model.ReadAllRequest
)

func (n *NotificationController) RegisterRoute(api *gin.RouterGroup) {
	api.GET("/notifications/unread-summary", n.UnreadSummary)
	api.GET("/notifications", n.ListInbox)
	api.PUT("/notifications/read-all", n.MarkAllRead)
	api.PUT("/notifications/:id/read", n.MarkRead)
}

// resolveTenantAndMember 解析租户与成员上下文：收件箱操作只信任 JWT/Gin
// 上下文，不接受客户端传入的替代 tenantId/memberId
func resolveTenantAndMember(c *gin.Context) (tenantID, memberID uint, ok bool) {
	tenantID, valid := contextx.TenantIDFromContext(c.Request.Context())
	if !valid {
		httpx.ResponseFailed(c, http.StatusUnauthorized, fmt.Errorf("tenant context required"))
		return 0, 0, false
	}
	user := ginctx.GetUser(c)
	if user == nil {
		httpx.ResponseFailed(c, http.StatusUnauthorized, fmt.Errorf("member context required"))
		return 0, 0, false
	}
	return tenantID, user.ID, true
}

// UnreadSummary 顶栏未读摘要。
//
// @Summary 查询未读摘要
// @Description 当前成员的未读总数与分类未读数（只返回未读数大于 0 的分类）；已过期消息即使尚未物理清理也不计数
// @Produce json
// @Tags 消息中心
// @Security JWT
// @Success 200 {object} httpx.Response{data=controller.UnreadSummaryView}
// @Failure 401 {object} httpx.Response "errCode=UNAUTHORIZED"
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN"
// @Router /api/v1/notifications/unread-summary [get]
func (n *NotificationController) UnreadSummary(c *gin.Context) {
	tenantID, memberID, ok := resolveTenantAndMember(c)
	if !ok {
		return
	}
	summary, err := n.inboxService.UnreadSummary(c.Request.Context(), tenantID, memberID)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, summary)
}

// ListInbox 当前成员消息游标分页。
//
// @Summary 查询消息列表
// @Description 当前成员在指定分类下的消息列表（按事件时间倒序，游标分页）；eventCode 必须属于当前分类，unreadOnly 过滤未读；列表排除已过期消息
// @Produce json
// @Tags 消息中心
// @Security JWT
// @Param categoryId query string true "分类码（八个稳定分类之一）"
// @Param eventCode query string false "事件码（必须属于当前分类）"
// @Param unreadOnly query boolean false "只看未读，默认 false"
// @Param cursor query string false "不透明游标（上页 nextCursor 原样回传）"
// @Param limit query int false "每页条数，默认 20，上限 100"
// @Success 200 {object} httpx.Response{data=controller.InboxPageView}
// @Failure 400 {object} httpx.Response "errCode=NOTIFICATION_CATEGORY_UNKNOWN / NOTIFICATION_EVENT_UNKNOWN / NOTIFICATION_CURSOR_INVALID"
// @Failure 401 {object} httpx.Response "errCode=UNAUTHORIZED"
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN"
// @Router /api/v1/notifications [get]
func (n *NotificationController) ListInbox(c *gin.Context) {
	tenantID, memberID, ok := resolveTenantAndMember(c)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	unreadOnly := c.Query("unreadOnly") == "true"
	result, err := n.inboxService.ListInbox(c.Request.Context(), tenantID, memberID, model.InboxQuery{
		CategoryID: c.Query("categoryId"),
		EventCode:  c.Query("eventCode"),
		UnreadOnly: unreadOnly,
		Cursor:     c.Query("cursor"),
		Limit:      limit,
	})
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, result)
}

// MarkRead 幂等标记单条已读。
//
// @Summary 标记单条消息已读
// @Description 幂等：重复调用成功且不改写首次已读时间；响应携带最新未读摘要避免前端二次请求；只能标记当前成员自己收件箱中的消息
// @Produce json
// @Tags 消息中心
// @Security JWT
// @Param id path int true "收件箱行 ID（列表项 id）"
// @Success 200 {object} httpx.Response{data=controller.UnreadSummaryView}
// @Failure 400 {object} httpx.Response "errCode=VALIDATION"
// @Failure 401 {object} httpx.Response "errCode=UNAUTHORIZED"
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN"
// @Failure 404 {object} httpx.Response "errCode=NOTIFICATION_NOT_FOUND"
// @Router /api/v1/notifications/{id}/read [put]
func (n *NotificationController) MarkRead(c *gin.Context) {
	tenantID, memberID, ok := resolveTenantAndMember(c)
	if !ok {
		return
	}
	inboxID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest,
			httpx.NewBiz(httpx.CodeValidation, "消息 ID 必须为数字", http.StatusBadRequest))
		return
	}
	summary, err := n.inboxService.MarkRead(c.Request.Context(), tenantID, memberID, uint(inboxID))
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, summary)
}

// MarkAllRead 标记当前分类全部已读。
//
// @Summary 标记分类消息全部已读
// @Description 标记当前分类（可选事件）occurred_at 不晚于 through 的未读消息为已读；through 建议回传本次列表响应的 serverTime，不误伤操作同时新到达的消息；不影响其他分类
// @Accept json
// @Produce json
// @Tags 消息中心
// @Security JWT
// @Param body body controller.ReadAllRequest true "分类（必填）、事件（可选）与 through 时间上界"
// @Success 200 {object} httpx.Response{data=controller.UnreadSummaryView}
// @Failure 400 {object} httpx.Response "errCode=NOTIFICATION_CATEGORY_UNKNOWN / NOTIFICATION_EVENT_UNKNOWN / NOTIFICATION_CURSOR_INVALID"
// @Failure 401 {object} httpx.Response "errCode=UNAUTHORIZED"
// @Failure 403 {object} httpx.Response "errCode=FORBIDDEN"
// @Router /api/v1/notifications/read-all [put]
func (n *NotificationController) MarkAllRead(c *gin.Context) {
	tenantID, memberID, ok := resolveTenantAndMember(c)
	if !ok {
		return
	}
	req := new(model.ReadAllRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	summary, err := n.inboxService.MarkAllRead(c.Request.Context(), tenantID, memberID, *req)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, summary)
}
