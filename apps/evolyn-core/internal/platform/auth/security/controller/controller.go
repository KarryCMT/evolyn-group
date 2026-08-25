package controller

import (
	"net/http"

	"evolyn/internal/platform/auth/security/service"
	"evolyn/internal/platform/controller"
	"evolyn/internal/platform/ginctx"
	"evolyn/internal/platform/httpx"

	"github.com/gin-gonic/gin"
)

// SecurityController 账号安全子域（ADR-009 第 2 步：只读骨架 + 本人踢出会话）。
// 挂 /accounts/me/* 之下，与 iam 账号自助控制器同域不同段
type SecurityController struct {
	securityService service.SecurityService
}

func NewSecurityController(securityService service.SecurityService) controller.Controller {
	return &SecurityController{securityService: securityService}
}

// myAccount 从会话取账号 ID；未认证返回 false 并已写响应
func (sc *SecurityController) myAccount(c *gin.Context) (uint, bool) {
	claims := ginctx.GetSession(c)
	if claims == nil || claims.AccountID == 0 {
		httpx.ResponseFailed(c, http.StatusUnauthorized, nil)
		return 0, false
	}
	return claims.AccountID, true
}

// @Summary 我的安全概览
// @Description 账号安全开关状态、TOTP 注册态、恢复码余量与活跃会话数
// @Produce json
// @Tags 账号
// @Security JWT
// @Success 200 {object} httpx.Response{data=service.SecurityOverview}
// @Router /api/v1/accounts/me/security [get]
func (sc *SecurityController) Overview(c *gin.Context) {
	accountID, ok := sc.myAccount(c)
	if !ok {
		return
	}

	overview, err := sc.securityService.Overview(c.Request.Context(), accountID)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusInternalServerError, err)
		return
	}
	httpx.ResponseSuccess(c, overview)
}

// @Summary 我的活跃会话
// @Description 当前账号的活跃设备会话列表（按最近活跃倒序）
// @Produce json
// @Tags 账号
// @Security JWT
// @Success 200 {object} httpx.Response{data=[]model.AccountSession}
// @Router /api/v1/accounts/me/sessions [get]
func (sc *SecurityController) ListSessions(c *gin.Context) {
	accountID, ok := sc.myAccount(c)
	if !ok {
		return
	}

	sessions, err := sc.securityService.ListSessions(c.Request.Context(), accountID)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusInternalServerError, err)
		return
	}
	httpx.ResponseSuccess(c, sessions)
}

// @Summary 踢出我的会话
// @Description 撤销本人指定设备会话（校验归属；他人会话按不存在处理）
// @Produce json
// @Tags 账号
// @Security JWT
// @Param sid path string true "会话标识"
// @Success 200 {object} httpx.Response
// @Failure 404 {object} httpx.Response "NOT_FOUND·会话不存在"
// @Router /api/v1/accounts/me/sessions/{sid} [delete]
func (sc *SecurityController) RevokeSession(c *gin.Context) {
	accountID, ok := sc.myAccount(c)
	if !ok {
		return
	}

	if err := sc.securityService.RevokeSession(c.Request.Context(), accountID, c.Param("sid")); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, nil)
}

func (sc *SecurityController) RegisterRoute(api *gin.RouterGroup) {
	api.GET("/accounts/me/security", sc.Overview)
	api.GET("/accounts/me/sessions", sc.ListSessions)
	api.DELETE("/accounts/me/sessions/:sid", sc.RevokeSession)
}

func (sc *SecurityController) Name() string {
	return "AccountSecurity"
}
