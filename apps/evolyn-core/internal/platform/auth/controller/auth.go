package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"evolyn/internal/platform/auth"
	"evolyn/internal/platform/auth/oauth"
	"evolyn/internal/platform/auth/sms"
	platformcontroller "evolyn/internal/platform/controller"
	"evolyn/internal/platform/ginctx"
	"evolyn/internal/platform/httpx"
	"evolyn/internal/platform/iam/authorization"
	"evolyn/internal/platform/iam/model"
	"evolyn/internal/platform/iam/service"
	tenantmodel "evolyn/internal/platform/tenant/model"
	tenantservice "evolyn/internal/platform/tenant/service"

	"github.com/gin-gonic/gin"
)

// AuthController 会话域：登录/注册/登出（ADR-006 账号×成员模型）
type AuthController struct {
	accountService service.AccountService
	jwtService     *auth.JWTService
	oauthManger    *oauth.OAuthManager
	tenantService  tenantservice.TenantService
	smsService     *sms.Service
}

func NewAuthController(accountService service.AccountService, jwtService *auth.JWTService, oauthManager *oauth.OAuthManager, tenantService tenantservice.TenantService, smsService *sms.Service) platformcontroller.Controller {
	return &AuthController{
		accountService: accountService,
		jwtService:     jwtService,
		oauthManger:    oauthManager,
		tenantService:  tenantService,
		smsService:     smsService,
	}
}

// loginSession 登录成功后的签发与 Cookie 写入：token 绑定「账号+成员+租户」
func (ac *AuthController) loginSession(c *gin.Context, account *model.Account, member *model.User, setCookie bool) (string, error) {
	token, err := ac.jwtService.CreateToken(account, member)
	if err != nil {
		return "", err
	}

	if setCookie {
		memberJson, err := json.Marshal(member)
		if err != nil {
			return "", err
		}
		c.SetCookie(httpx.CookieTokenName, token, 3600*24, "/", "", true, true)
		c.SetCookie(httpx.CookieLoginUser, string(memberJson), 3600*24, "/", "", true, false)
	}

	return token, nil
}

// @Summary 登录
// @Description 账号登录（用户名/手机号 + 密码），租户可选
// @Accept json
// @Produce json
// @Tags 认证
// @Param user body model.AuthUser true "auth info"
// @Success 200 {object} httpx.Response{data=model.JWTToken}
// @Router /api/v1/auth/token [post]
func (ac *AuthController) Login(c *gin.Context) {
	auser := new(model.AuthUser)
	if err := c.BindJSON(auser); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	var (
		account *model.Account
		member  *model.User
		err     error
	)
	switch {
	case auser.SmsCode != "":
		// 验证码登录：仅接受 phone + smsCode（免密），先校验验证码再解析账号
		if auser.Phone == "" || auser.Password != "" || auser.Name != "" {
			httpx.ResponseFailed(c, http.StatusBadRequest, fmt.Errorf("sms login accepts phone and smsCode only"))
			return
		}
		if err := ac.smsService.Verify(c.Request.Context(), sms.SceneLogin, auser.Phone, auser.SmsCode); err != nil {
			httpx.ResponseFailed(c, http.StatusUnauthorized, err)
			return
		}
		account, member, err = ac.accountService.AuthByPhone(c.Request.Context(), auser.Phone, auser.TenantCode)
	case !oauth.IsEmptyAuthType(auser.AuthType) && auser.Name == "" && auser.Phone == "":
		provider, err := ac.oauthManger.GetAuthProvider(auser.AuthType)
		if err != nil {
			httpx.ResponseFailed(c, http.StatusBadRequest, err)
			return
		}
		authToken, err := provider.GetToken(auser.AuthCode)
		if err != nil {
			httpx.ResponseFailed(c, http.StatusBadRequest, err)
			return
		}

		userInfo, err := provider.GetUserInfo(authToken)
		if err != nil {
			httpx.ResponseFailed(c, http.StatusBadRequest, err)
			return
		}

		account, member, err = ac.accountService.CreateOAuthAccount(c.Request.Context(), userInfo.Account())
	default:
		account, member, err = ac.accountService.Auth(c.Request.Context(), auser)
	}
	if err != nil {
		httpx.ResponseFailed(c, http.StatusUnauthorized, err)
		return
	}

	token, err := ac.loginSession(c, account, member, auser.SetCookie)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusInternalServerError, err)
		return
	}

	httpx.ResponseSuccess(c, model.JWTToken{
		Token:    token,
		Describe: "set token in Authorization Header, [Authorization: Bearer {token}]",
	})
}

// @Summary 退出登录
// @Description 退出登录并清除会话 Cookie
// @Produce json
// @Tags 认证
// @Success 200 {object} httpx.Response
// @Router /api/v1/auth/token [delete]
func (ac *AuthController) Logout(c *gin.Context) {
	c.SetCookie(httpx.CookieTokenName, "", -1, "/", "", true, true)
	c.SetCookie(httpx.CookieLoginUser, "", -1, "/", "", true, false)
	httpx.ResponseSuccess(c, nil)
}

// registerResult 注册结果：账号 + 默认租户成员
type registerResult struct {
	Account *model.Account `json:"account"`
	Member  *model.User    `json:"member"`
}

// @Summary 注册账号
// @Description 创建平台账号及默认租户成员身份
// @Accept json
// @Produce json
// @Tags 认证
// @Param account body model.CreatedAccount true "account info"
// @Success 200 {object} httpx.Response{data=controller.registerResult}
// @Router /api/v1/auth/user [post]
func (ac *AuthController) Register(c *gin.Context) {
	createdAccount := new(model.CreatedAccount)
	if err := c.BindJSON(createdAccount); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	account := createdAccount.GetAccount()
	if err := ac.accountService.Validate(account); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	ac.accountService.Default(account)
	account, member, err := ac.accountService.Register(c.Request.Context(), account)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusInternalServerError, err)
		return
	}

	httpx.ResponseSuccess(c, registerResult{Account: account, Member: member})
}

// openTenantRequest 自助开通租户请求：名称必填；其余为注册向导采集的
// 企业画像（选填，写入租户 Config.onboarding 供个性化模板/运营统计）
type openTenantRequest struct {
	Name            string   `json:"name" binding:"required,min=2,max=50"`
	Demand          string   `json:"demand"`          // 你的需求（单选）
	Industry        string   `json:"industry"`        // 所属行业（单选）
	ManagementNeeds []string `json:"managementNeeds"` // 企业内部管理需求（多选）
}

// @Summary 自助开通租户
// @Description 注册向导「创建团队」：当前账号自助开通租户并成为所有者（绑定 tenant-admin 角色）；
// 套餐默认免费版，编码由服务端随机生成；demand/industry/managementNeeds 为企业画像采集（选填）
// @Accept json
// @Produce json
// @Tags 认证
// @Security JWT
// @Param body body controller.openTenantRequest true "tenant name and onboarding profile"
// @Success 200 {object} httpx.Response{data=tenantmodel.Tenant}
// @Router /api/v1/auth/tenant [post]
func (ac *AuthController) OpenTenant(c *gin.Context) {
	claims, ok := ac.sessionFrom(c)
	if !ok {
		return
	}

	req := new(openTenantRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	var tenant *tenantmodel.Tenant
	tenant, err := ac.tenantService.SelfOpen(c.Request.Context(), claims.AccountID, req.Name, tenantmodel.OnboardingConfig{
		Demand:          req.Demand,
		Industry:        req.Industry,
		ManagementNeeds: req.ManagementNeeds,
	})
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	httpx.ResponseSuccess(c, tenant)
}

// smsSendRequest 发送验证码请求：scene 一期仅 login
type smsSendRequest struct {
	Phone string `json:"phone" binding:"required"`
	Scene string `json:"scene" binding:"required"`
}

// smsSendResult 发送结果：仅本地联调（devEcho=true）时回显验证码
type smsSendResult struct {
	Code string `json:"code,omitempty"`
}

// @Summary 发送短信验证码
// @Description 按场景发送 6 位短信验证码（一期场景 login）：默认 60 秒重发冷却、
// 5 分钟有效期、单码最多试错 5 次后作废需重发；开发通道 devEcho=true 时响应回显验证码
// @Accept json
// @Produce json
// @Tags 认证
// @Param body body controller.smsSendRequest true "phone and scene"
// @Success 200 {object} httpx.Response{data=controller.smsSendResult}
// @Failure 429 {object} httpx.Response "发送冷却中"
// @Router /api/v1/auth/sms/send [post]
func (ac *AuthController) SendSmsCode(c *gin.Context) {
	req := new(smsSendRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	code, err := ac.smsService.Send(c.Request.Context(), req.Scene, req.Phone)
	if err != nil {
		// 冷却中用 429，其余（场景/手机号非法）用 400
		status := http.StatusBadRequest
		if errors.Is(err, sms.ErrCooldown) {
			status = http.StatusTooManyRequests
		}
		httpx.ResponseFailed(c, status, err)
		return
	}

	result := smsSendResult{}
	// devEcho 仅本地联调配置开启，生产环境恒为空
	if ac.smsService.EchoEnabled() {
		result.Code = code
	}
	httpx.ResponseSuccess(c, result)
}

func (ac *AuthController) RegisterRoute(api *gin.RouterGroup) {
	api.POST("/auth/token", ac.Login)
	api.DELETE("/auth/token", ac.Logout)
	api.POST("/auth/user", ac.Register)
	api.POST("/auth/tenant", ac.OpenTenant)
	api.POST("/auth/sms/send", ac.SendSmsCode)
	api.GET("/auth/tenants", ac.ListTenants)
	api.POST("/auth/token/switch", ac.SwitchTenant)
	api.GET("/auth/userinfo", ac.UserInfo)
	api.GET("/auth/permissions", ac.Permissions)
}

func (ac *AuthController) Name() string {
	return "Authentication"
}

// sessionFrom 从请求取出会话 claims；未认证返回 false 并已写响应
func (ac *AuthController) sessionFrom(c *gin.Context) (*auth.CustomClaims, bool) {
	claims := ginctx.GetSession(c)
	if claims == nil || claims.AccountID == 0 {
		httpx.ResponseFailed(c, http.StatusUnauthorized, fmt.Errorf("authentication required"))
		return nil, false
	}
	return claims, true
}

// @Summary 我的租户列表
// @Description 查询账号加入的租户列表（含是否所有者标记）
// @Produce json
// @Tags 认证
// @Security JWT
// @Success 200 {object} httpx.Response{data=[]service.TenantMembership}
// @Router /api/v1/auth/tenants [get]
func (ac *AuthController) ListTenants(c *gin.Context) {
	claims, ok := ac.sessionFrom(c)
	if !ok {
		return
	}

	memberships, err := ac.accountService.ListTenants(c.Request.Context(), claims.AccountID)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusInternalServerError, err)
		return
	}

	httpx.ResponseSuccess(c, memberships)
}

// switchTenantRequest 切换租户请求
type switchTenantRequest struct {
	TenantID uint `json:"tenantId" binding:"required"`
}

// @Summary 切换租户
// @Description 切换当前租户成员身份并重新签发令牌
// @Accept json
// @Produce json
// @Tags 认证
// @Security JWT
// @Param body body controller.switchTenantRequest true "tenant id"
// @Success 200 {object} httpx.Response{data=model.JWTToken}
// @Router /api/v1/auth/token/switch [post]
func (ac *AuthController) SwitchTenant(c *gin.Context) {
	claims, ok := ac.sessionFrom(c)
	if !ok {
		return
	}

	req := new(switchTenantRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	account, member, err := ac.accountService.SwitchTenant(c.Request.Context(), claims.AccountID, req.TenantID)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusForbidden, err)
		return
	}

	token, err := ac.loginSession(c, account, member, false)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusInternalServerError, err)
		return
	}

	httpx.ResponseSuccess(c, model.JWTToken{
		Token:    token,
		Describe: "tenant switched, replace your Authorization token",
	})
}

// @Summary 当前用户信息（聚合）
// @Description 平台账号资料 + 当前租户成员身份 + 租户配置/套餐/配额聚合
// @Produce json
// @Tags 认证
// @Security JWT
// @Success 200 {object} httpx.Response{data=service.UserInfoResult}
// @Router /api/v1/auth/userinfo [get]
func (ac *AuthController) UserInfo(c *gin.Context) {
	claims, ok := ac.sessionFrom(c)
	if !ok {
		return
	}

	member := ginctx.GetUser(c)
	if member == nil {
		httpx.ResponseFailed(c, http.StatusUnauthorized, fmt.Errorf("member not loaded"))
		return
	}

	info, err := ac.accountService.GetUserInfo(c.Request.Context(), claims.AccountID, member)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusInternalServerError, err)
		return
	}

	httpx.ResponseSuccess(c, info)
}

// @Summary 我的权限集
// @Description 由成员角色推导的权限布尔集合（resource:verb）
// @Produce json
// @Tags 认证
// @Security JWT
// @Success 200 {object} httpx.Response{data=map[string]bool}
// @Router /api/v1/auth/permissions [get]
func (ac *AuthController) Permissions(c *gin.Context) {
	member := ginctx.GetUser(c)
	if member == nil {
		httpx.ResponseFailed(c, http.StatusUnauthorized, fmt.Errorf("authentication required"))
		return
	}

	httpx.ResponseSuccess(c, authorization.PermissionsOf(member))
}
