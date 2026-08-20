package controller

import (
	"encoding/json"
	"fmt"
	"net/http"

	"evolyn/internal/platform/auth"
	"evolyn/internal/platform/auth/oauth"
	"evolyn/internal/platform/auth/pki"
	authservice "evolyn/internal/platform/auth/service"
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
	// registrationService 注册编排：向导最终提交的单事务落库（账号+画像+租户+owner）
	registrationService authservice.RegistrationService
	jwtService          *auth.JWTService
	oauthManger         *oauth.OAuthManager
	tenantService       tenantservice.TenantService
	smsService          *sms.Service
	// 登录口令加密密钥对：密码登录分支先解密再走 bcrypt 校验
	pkiKeypair *pki.Keypair
}

func NewAuthController(accountService service.AccountService, registrationService authservice.RegistrationService, jwtService *auth.JWTService, oauthManager *oauth.OAuthManager, tenantService tenantservice.TenantService, smsService *sms.Service, pkiKeypair *pki.Keypair) platformcontroller.Controller {
	return &AuthController{
		accountService:      accountService,
		registrationService: registrationService,
		jwtService:          jwtService,
		oauthManger:         oauthManager,
		tenantService:       tenantService,
		smsService:          smsService,
		pkiKeypair:          pkiKeypair,
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
// @Description 账号登录（用户名/手机号 + 密码），租户可选；password 需先经
// GET /app/conf 下发的 RSA 公钥加密（pki 段）后上送，验证码登录（smsCode）不受影响
// @Accept json
// @Produce json
// @Tags 认证
// @Param user body model.AuthUser true "auth info"
// @Success 200 {object} httpx.Response{data=model.JWTToken}
// @Failure 401 {object} httpx.Response "UNAUTHORIZED·凭证错误（AUTH_SMS_INVALID/AUTH_CREDENTIALS_INVALID）"
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
		// 密码登录：password 为经 /app/conf 公钥 RSA 加密后的密文，
		// 先解密再走服务层 bcrypt 校验；解密失败不区分原因（防探测）。
		// 注意解密错误用独立变量名，避免 := 遮蔽外层 err 导致登录错误被吞
		plain, decryptErr := ac.pkiKeypair.Decrypt(auser.Password)
		if decryptErr != nil {
			httpx.ResponseFailed(c, http.StatusBadRequest, decryptErr)
			return
		}
		auser.Password = plain
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

// registerOnboarding 注册向导第 3 步采集的账号画像（「人」的属性挂账号）
type registerOnboarding struct {
	Role    string `json:"role"`    // 你的角色：ceo / manager / it / ...
	Channel string `json:"channel"` // 了解渠道：xiaohongshu / zhihu / referral / ...
}

// registerTenantProfile 注册向导第 2 步采集的企业画像（写入租户 Config）
type registerTenantProfile struct {
	Name     string `json:"name" binding:"required"` // 企业名称
	Demand   string `json:"demand"`                  // 你的需求（单选，选填）
	Industry string `json:"industry"`                // 所属行业（单选，选填）
}

// registerRequest 注册向导最终提交请求（POST /auth/register）：三步纯前端
// 采集的全量数据——手机号+验证码、企业画像、账号画像，点「进入产品」时
// 一次性上送，此前向导各步不产生任何服务端写副作用
type registerRequest struct {
	Phone      string                `json:"phone" binding:"required"`
	SmsCode    string                `json:"smsCode" binding:"required"`
	Nickname   string                `json:"nickname"`                  // 怎么称呼你（空串保留脱敏手机号默认）
	Onboarding registerOnboarding    `json:"onboarding"`                // 账号画像：角色/了解渠道
	Tenant     registerTenantProfile `json:"tenant" binding:"required"` // 企业画像
}

// registerTokenResult 注册结果：注册完成即登录，返回绑定新租户的会话令牌；
// created=false 表示手机号已注册（等价短信登录，向导重试幂等）
type registerTokenResult struct {
	Token    string `json:"token"`
	Describe string `json:"describe"`
	Created  bool   `json:"created"`
}

// @Summary 注册（注册向导最终提交）
// @Description 「进入产品」时一次性提交向导三步采集的全量数据（手机号+验证码、
// 企业画像、账号画像），服务端单事务完成：免密注册账号（已注册手机号等价
// 短信登录，created=false）→ 落账号昵称/角色/渠道画像 → 自助开通租户并绑定
// tenant-admin（名下已有自有租户则复用）→ 返回绑定新租户的会话令牌。
// 注册全程不设密码：账号为免密状态，密码由用户后续在个人中心首次设置。
// 验证码随本请求一次性校验，向导停留超有效期将返回 401 需重新获取
// @Accept json
// @Produce json
// @Tags 认证
// @Param body body controller.registerRequest true "向导三步采集的全量数据"
// @Success 200 {object} httpx.Response{data=controller.registerTokenResult}
// @Failure 401 {object} httpx.Response "AUTH_SMS_INVALID·验证码错误或已过期"
// @Failure 409 {object} httpx.Response "AUTH_PHONE_DUPLICATED·手机号已注册"
// @Failure 401 {object} httpx.Response "验证码错误或已过期"
// @Router /api/v1/auth/register [post]
func (ac *AuthController) RegisterComplete(c *gin.Context) {
	req := new(registerRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	// 验证码校验先行（scene=register 与登录验证码隔离；一次性消费）
	if err := ac.smsService.Verify(c.Request.Context(), sms.SceneRegister, req.Phone, req.SmsCode); err != nil {
		httpx.ResponseFailed(c, http.StatusUnauthorized, err)
		return
	}

	result, err := ac.registrationService.Complete(c.Request.Context(), &authservice.RegistrationRequest{
		Phone:            req.Phone,
		Nickname:         req.Nickname,
		Onboarding:       model.AccountOnboarding{Role: req.Onboarding.Role, Channel: req.Onboarding.Channel},
		TenantName:       req.Tenant.Name,
		TenantOnboarding: tenantmodel.OnboardingConfig{Demand: req.Tenant.Demand, Industry: req.Tenant.Industry},
	})
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	// 注册即登录：令牌直接绑定新租户 owner 成员（免客户端再走切换）
	token, err := ac.loginSession(c, result.Account, result.Member, false)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusInternalServerError, err)
		return
	}

	httpx.ResponseSuccess(c, registerTokenResult{
		Token:    token,
		Describe: "set token in Authorization Header, [Authorization: Bearer {token}]",
		Created:  result.Created,
	})
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

// smsSendRequest 发送验证码请求：scene 支持 login（登录）/ register（注册）
type smsSendRequest struct {
	Phone string `json:"phone" binding:"required"`
	Scene string `json:"scene" binding:"required"`
}

// smsSendResult 发送结果：仅本地联调（devEcho=true）时回显验证码
type smsSendResult struct {
	Code string `json:"code,omitempty"`
}

// @Summary 发送短信验证码
// @Description 按场景发送 6 位短信验证码（场景 login=登录 / register=注册）：默认 60 秒
// 重发冷却、5 分钟有效期、单码最多试错 5 次后作废需重发；开发通道（provider=dev）固定
// 验证码 666666，devEcho=true 时响应回显验证码（仅本地联调）
// @Accept json
// @Produce json
// @Tags 认证
// @Param body body controller.smsSendRequest true "phone and scene"
// @Success 200 {object} httpx.Response{data=controller.smsSendResult}
// @Failure 400 {object} httpx.Response "AUTH_PHONE_INVALID/AUTH_SMS_SCENE_INVALID·手机号或场景非法"
// @Failure 429 {object} httpx.Response "AUTH_COOLDOWN·发送冷却中"
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
		// 状态码/业务码由 BizError 自动映射（ADR-008）：冷却/试错 429，非法 400
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
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
	api.POST("/auth/register", ac.RegisterComplete)
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
