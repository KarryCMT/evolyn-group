package controller

import (
	"fmt"
	"net/http"

	"evolyn/internal/platform/auth/pki"
	securitymodel "evolyn/internal/platform/auth/security/model"
	"evolyn/internal/platform/auth/security/service"
	"evolyn/internal/platform/controller"
	"evolyn/internal/platform/ginctx"
	"evolyn/internal/platform/httpx"
	iammodel "evolyn/internal/platform/iam/model"
	iamservice "evolyn/internal/platform/iam/service"

	"github.com/gin-gonic/gin"
)

// SecurityController 账号安全子域（ADR-009 第 2 步：只读骨架 + 本人踢出会话）。
// 挂 /accounts/me/* 之下，与 iam 账号自助控制器同域不同段
type SecurityController struct {
	securityService service.SecurityService
	mfaService      service.MFAService
	accountService  iamservice.AccountService
	accountDeletion iamservice.AccountDeletionService
	pkiKeypair      *pki.Keypair
}

func NewSecurityController(securityService service.SecurityService, mfaService service.MFAService,
	accountService iamservice.AccountService, pkiKeypair *pki.Keypair,
	accountDeletion ...iamservice.AccountDeletionService) controller.Controller {
	securityController := &SecurityController{
		securityService: securityService,
		mfaService:      mfaService,
		accountService:  accountService,
		pkiKeypair:      pkiKeypair,
	}
	if len(accountDeletion) > 0 {
		securityController.accountDeletion = accountDeletion[0]
	}
	return securityController
}

func (sc *SecurityController) currentSID(c *gin.Context) string {
	claims := ginctx.GetSession(c)
	if claims == nil {
		return ""
	}
	return claims.SID
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
// @Failure 401 {object} httpx.Response "UNAUTHORIZED·登录态无效"
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
// @Success 200 {object} httpx.Response{data=[]securitymodel.AccountSession}
// @Failure 401 {object} httpx.Response "UNAUTHORIZED·登录态无效"
// @Router /api/v1/accounts/me/sessions [get]
func (sc *SecurityController) ListSessions(c *gin.Context) {
	accountID, ok := sc.myAccount(c)
	if !ok {
		return
	}

	var sessions []securitymodel.AccountSession
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
// @Failure 401 {object} httpx.Response "UNAUTHORIZED·登录态无效"
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

// reauthRequest 为高风险操作换取一次性短时凭证。已有 MFA 的账号可用 TOTP
// 或恢复码；尚未绑定因子时只能以登录密码证明当前操作者身份。
type reauthRequest struct {
	Password string `json:"password"`
	Method   string `json:"method"`
	Code     string `json:"code"`
}

// @Summary 重新验证身份
// @Description 高风险账号安全操作前，以密码或已绑定的 TOTP/恢复码换取五分钟一次性 reauthToken
// @Accept json
// @Produce json
// @Tags 账号
// @Security JWT
// @Param body body controller.reauthRequest true "重新验证信息"
// @Success 200 {object} httpx.Response{data=map[string]string}
// @Failure 400 {object} httpx.Response "AUTH_PASSWORD_DECRYPT_FAILED·密码传输校验失败"
// @Failure 401 {object} httpx.Response "AUTH_MFA_INVALID·验证信息错误或已过期"
// @Router /api/v1/accounts/me/security/reauth [post]
func (sc *SecurityController) Reauth(c *gin.Context) {
	accountID, ok := sc.myAccount(c)
	if !ok {
		return
	}
	if err := reauthPrerequisiteError(sc.mfaService != nil, sc.currentSID(c)); err != nil {
		// BizError 自带最终 HTTP 状态：MFA 未装配为 503，存量无 SID 登录态为
		// 401。不能把后者伪装成服务不可用，否则用户重复输入正确密码也无法恢复。
		httpx.ResponseFailed(c, http.StatusServiceUnavailable, err)
		return
	}
	req := new(reauthRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	var (
		token string
		err   error
	)
	if req.Password != "" {
		plain, decryptErr := sc.pkiKeypair.Decrypt(req.Password)
		if decryptErr != nil {
			httpx.ResponseFailed(c, http.StatusBadRequest, decryptErr)
			return
		}
		profile, profileErr := sc.accountService.GetProfile(c.Request.Context(), accountID)
		if profileErr != nil {
			httpx.ResponseFailed(c, http.StatusInternalServerError, profileErr)
			return
		}
		verified, _, verifyErr := sc.accountService.Auth(c.Request.Context(), &iammodel.AuthUser{Name: profile.Name, Password: plain})
		if verifyErr != nil || verified == nil || verified.ID != accountID {
			httpx.ResponseFailed(c, http.StatusUnauthorized, service.ErrMFAInvalid)
			return
		}
		token, err = sc.mfaService.IssueReauthToken(c.Request.Context(), accountID, sc.currentSID(c))
	} else {
		token, err = sc.mfaService.CreateReauthToken(c.Request.Context(), accountID, sc.currentSID(c), req.Method, req.Code)
	}
	if err != nil {
		httpx.ResponseFailed(c, http.StatusUnauthorized, err)
		return
	}
	httpx.ResponseSuccess(c, map[string]string{"reauthToken": token})
}

// reauthPrerequisiteError 区分基础设施未配置与存量登录态，避免两个可恢复方式
// 完全不同的问题共用 AUTH_MFA_UNAVAILABLE。
func reauthPrerequisiteError(mfaAvailable bool, sid string) error {
	if !mfaAvailable {
		return service.ErrMFAUnavailable
	}
	if sid == "" {
		return service.ErrMFAReauthLoginRequired
	}
	return nil
}

type reauthTokenRequest struct {
	ReauthToken string `json:"reauthToken" binding:"required"`
}

// @Summary 注销我的账号
// @Description 重新验证后永久删除本人账号、成员身份与第三方凭证；若仍是任一租户创建人，必须先转移创建人或注销租户
// @Accept json
// @Produce json
// @Tags 账号
// @Security JWT
// @Param body body controller.reauthTokenRequest true "重新验证令牌"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response "AUTH_REAUTH_REQUIRED·请先完成身份验证"
// @Failure 409 {object} httpx.Response "ACCOUNT_OWNS_TENANT·账号仍是租户创建人"
// @Router /api/v1/accounts/me [delete]
func (sc *SecurityController) CancelMyAccount(c *gin.Context) {
	accountID, ok := sc.myAccount(c)
	if !ok {
		return
	}
	if sc.accountDeletion == nil {
		httpx.ResponseFailed(c, http.StatusServiceUnavailable, service.ErrMFAUnavailable)
		return
	}
	req := new(reauthTokenRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	if err := reauthPrerequisiteError(sc.mfaService != nil, sc.currentSID(c)); err != nil {
		httpx.ResponseFailed(c, http.StatusServiceUnavailable, err)
		return
	}
	if err := sc.requireReauth(c, accountID, req.ReauthToken); err != nil {
		httpx.ResponseFailed(c, http.StatusUnauthorized, err)
		return
	}
	if err := sc.accountDeletion.Delete(c.Request.Context(), accountID); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, nil)
}

// @Summary 创建 TOTP 绑定向导
// @Description 重新验证后生成五分钟有效的验证器导入地址；前端仅用于生成二维码，不得持久化密钥
// @Accept json
// @Produce json
// @Tags 账号
// @Security JWT
// @Param body body controller.reauthTokenRequest true "重新验证令牌"
// @Success 200 {object} httpx.Response{data=service.TOTPEnrollment}
// @Failure 401 {object} httpx.Response "AUTH_REAUTH_REQUIRED·请先完成身份验证"
// @Failure 409 {object} httpx.Response "AUTH_MFA_ALREADY_ENABLED·登录二次验证已启用"
// @Router /api/v1/accounts/me/security/mfa/totp/enroll [post]
func (sc *SecurityController) EnrollTOTP(c *gin.Context) {
	accountID, ok := sc.myAccount(c)
	if !ok {
		return
	}
	req := new(reauthTokenRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	if err := sc.requireReauth(c, accountID, req.ReauthToken); err != nil {
		httpx.ResponseFailed(c, http.StatusUnauthorized, err)
		return
	}
	enrollment, err := sc.mfaService.Enroll(c.Request.Context(), accountID, "灵衍云", "account-"+fmt.Sprint(accountID))
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, enrollment)
}

type confirmTOTPRequest struct {
	EnrollmentID string `json:"enrollmentId" binding:"required"`
	Code         string `json:"code" binding:"required,len=6"`
}

// @Summary 确认 TOTP 绑定
// @Description 校验验证器首个动态码后启用 MFA，并仅一次返回恢复码；成功会撤销其他设备会话
// @Accept json
// @Produce json
// @Tags 账号
// @Security JWT
// @Param body body controller.confirmTOTPRequest true "绑定确认信息"
// @Success 200 {object} httpx.Response{data=map[string][]string}
// @Failure 401 {object} httpx.Response "AUTH_MFA_INVALID·验证信息错误或已过期"
// @Router /api/v1/accounts/me/security/mfa/totp/confirm [post]
func (sc *SecurityController) ConfirmTOTP(c *gin.Context) {
	accountID, ok := sc.myAccount(c)
	if !ok {
		return
	}
	req := new(confirmTOTPRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	// enroll 已要求并消费 reauthToken；enrollmentID 仅在该操作成功后写入 Redis，
	// 且在此处再次校验账号归属，因此确认不重复消费已使用的一次性凭证。
	codes, err := sc.mfaService.ConfirmEnrollment(c.Request.Context(), accountID, sc.currentSID(c), req.EnrollmentID, req.Code)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, map[string][]string{"recoveryCodes": codes})
}

// @Summary 关闭 TOTP 登录二次验证
// @Description 重新验证后关闭当前账号的 TOTP 因子并撤销其他设备会话
// @Accept json
// @Produce json
// @Tags 账号
// @Security JWT
// @Param body body controller.reauthTokenRequest true "重新验证令牌"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response "AUTH_REAUTH_REQUIRED/AUTH_MFA_INVALID·需重新验证或验证器未启用"
// @Router /api/v1/accounts/me/security/mfa/totp [delete]
func (sc *SecurityController) DisableTOTP(c *gin.Context) {
	accountID, ok := sc.myAccount(c)
	if !ok {
		return
	}
	req := new(reauthTokenRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	if err := sc.requireReauth(c, accountID, req.ReauthToken); err != nil {
		httpx.ResponseFailed(c, http.StatusUnauthorized, err)
		return
	}
	if err := sc.mfaService.Disable(c.Request.Context(), accountID, sc.currentSID(c)); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, nil)
}

type singleSessionRequest struct {
	ReauthToken string `json:"reauthToken" binding:"required"`
	Enabled     bool   `json:"enabled"`
}

// @Summary 设置禁止同时登录
// @Description 重新验证后更新账号级单会话开关；开启时立即撤销其他设备会话
// @Accept json
// @Produce json
// @Tags 账号
// @Security JWT
// @Param body body controller.singleSessionRequest true "单会话设置"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response "AUTH_REAUTH_REQUIRED·请先完成身份验证"
// @Router /api/v1/accounts/me/security/single-session [put]
func (sc *SecurityController) UpdateSingleSession(c *gin.Context) {
	accountID, ok := sc.myAccount(c)
	if !ok {
		return
	}
	req := new(singleSessionRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	if err := sc.requireReauth(c, accountID, req.ReauthToken); err != nil {
		httpx.ResponseFailed(c, http.StatusUnauthorized, err)
		return
	}
	if err := sc.securityService.UpdateSingleSession(c.Request.Context(), accountID, sc.currentSID(c), req.Enabled); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, nil)
}

func (sc *SecurityController) requireReauth(c *gin.Context, accountID uint, token string) error {
	if sc.mfaService == nil {
		return service.ErrMFAUnavailable
	}
	return sc.mfaService.RequireReauth(c.Request.Context(), accountID, sc.currentSID(c), token)
}

func (sc *SecurityController) RegisterRoute(api *gin.RouterGroup) {
	api.DELETE("/accounts/me", sc.CancelMyAccount)
	api.GET("/accounts/me/security", sc.Overview)
	api.POST("/accounts/me/security/reauth", sc.Reauth)
	api.POST("/accounts/me/security/mfa/totp/enroll", sc.EnrollTOTP)
	api.POST("/accounts/me/security/mfa/totp/confirm", sc.ConfirmTOTP)
	api.DELETE("/accounts/me/security/mfa/totp", sc.DisableTOTP)
	api.PUT("/accounts/me/security/single-session", sc.UpdateSingleSession)
	api.GET("/accounts/me/sessions", sc.ListSessions)
	api.DELETE("/accounts/me/sessions/:sid", sc.RevokeSession)
}

func (sc *SecurityController) Name() string {
	return "AccountSecurity"
}
