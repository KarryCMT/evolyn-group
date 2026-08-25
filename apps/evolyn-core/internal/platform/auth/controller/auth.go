package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"evolyn/internal/platform/auth"
	loginlogmodel "evolyn/internal/platform/auth/loginlog/model"
	loginlogservice "evolyn/internal/platform/auth/loginlog/service"
	"evolyn/internal/platform/auth/oauth"
	"evolyn/internal/platform/auth/pki"
	secmodel "evolyn/internal/platform/auth/security/model"
	securityservice "evolyn/internal/platform/auth/security/service"
	authservice "evolyn/internal/platform/auth/service"
	"evolyn/internal/platform/auth/sms"
	platformcontroller "evolyn/internal/platform/controller"
	"evolyn/internal/platform/ginctx"
	"evolyn/internal/platform/httpx"
	"evolyn/internal/platform/iam/authorization"
	"evolyn/internal/platform/iam/model"
	"evolyn/internal/platform/iam/service"
	"evolyn/internal/platform/middleware"
	tenantmodel "evolyn/internal/platform/tenant/model"
	tenantservice "evolyn/internal/platform/tenant/service"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
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
	// 令牌吊销器：登出时拉黑 jti（P2-8，存量无 sid 令牌兼容）
	revoker *auth.TokenRevoker
	// 设备会话服务（ADR-009）：登录链路统一签发/校验/撤销 account_sessions
	sessions securityservice.SessionService
	// 登录口令加密密钥对：密码登录分支先解密再走 bcrypt 校验
	pkiKeypair *pki.Keypair
	// 登录日志记录器：令牌签发成功后 best-effort 落一条会话建立流水
	loginLog loginlogservice.Recorder
	// 登录失败锁定（上线前整改 P2）：密码分支以登录名/手机号为维度计
	// 连续失败，达上限临时锁定，防在线爆破与撞库
	loginGuard *auth.LoginGuard
}

func NewAuthController(accountService service.AccountService, registrationService authservice.RegistrationService, jwtService *auth.JWTService, oauthManager *oauth.OAuthManager, tenantService tenantservice.TenantService, smsService *sms.Service, pkiKeypair *pki.Keypair, loginLog loginlogservice.Recorder, revoker *auth.TokenRevoker, sessions securityservice.SessionService, loginGuard *auth.LoginGuard) platformcontroller.Controller {
	return &AuthController{
		accountService:      accountService,
		registrationService: registrationService,
		jwtService:          jwtService,
		oauthManger:         oauthManager,
		tenantService:       tenantService,
		smsService:          smsService,
		pkiKeypair:          pkiKeypair,
		loginLog:            loginLog,
		revoker:             revoker,
		loginGuard:          loginGuard,
	}
}

// errLoginLocked 登录失败锁定（上线前整改 P2）：窗口内连续失败达上限，
// 登录名/手机号被临时锁定。定义在控制器层——auth 包不可 import httpx
// （ginctx→auth→httpx 循环依赖），LoginGuard 只暴露布尔锁定状态
var errLoginLocked = httpx.NewBiz("AUTH_LOGIN_LOCKED", "失败次数过多，请稍后再试", http.StatusTooManyRequests)

// recordLogin 落一条登录日志：member 为本次会话绑定的成员（可空，如未选
// 租户场景）；IP/UA 由服务层从请求元数据自动补全，写入失败只告警不影响登录
func (ac *AuthController) recordLogin(c *gin.Context, account *model.Account, member *model.User, method string) {
	if ac.loginLog == nil || account == nil {
		return
	}
	entry := loginlogservice.Entry{AccountID: account.ID, Method: method}
	if member != nil {
		entry.TenantID = member.TenantID
		entry.MemberID = member.ID
	}
	ac.loginLog.Record(c.Request.Context(), entry)
}

// loginSession 登录成功后的签发与 Cookie 写入：token 绑定
// 「账号+成员+租户+设备会话（sid）」（ADR-009）。authMethod 声明第一步
// 通过的证据类别（password/sms/oauth/register），落入会话流水
func (ac *AuthController) loginSession(c *gin.Context, account *model.Account, member *model.User, setCookie bool, authMethod string) (string, error) {
	var session *secmodel.AccountSession
	if ac.sessions != nil {
		ua := ""
		if c.Request != nil {
			ua = c.Request.UserAgent()
		}
		var err error
		session, err = ac.sessions.Issue(c.Request.Context(), securityservice.IssueRequest{
			AccountID:  account.ID,
			AuthMethod: authMethod,
			IP:         c.ClientIP(),
			UserAgent:  ua,
		})
		if err != nil {
			return "", err
		}
	}

	token, err := ac.jwtService.CreateToken(account, member, session)
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
// GET /app/conf 下发的 RSA 公钥加密（pki 段）后上送，验证码登录（smsCode）不受影响。
// 密码登录连续失败达上限（默认 15 分钟内 5 次）将临时锁定该登录名/手机号
// @Accept json
// @Produce json
// @Tags 认证
// @Param user body model.AuthUser true "auth info"
// @Success 200 {object} httpx.Response{data=model.JWTToken}
// @Failure 401 {object} httpx.Response "UNAUTHORIZED·凭证错误（AUTH_SMS_INVALID/AUTH_CREDENTIALS_INVALID）"
// @Failure 429 {object} httpx.Response "AUTH_LOGIN_LOCKED·失败次数过多已临时锁定"
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
		// 登录方式随分支确定，登录成功后落登录日志（loginlog method 列）
		method = loginlogmodel.MethodPassword
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
		method = loginlogmodel.MethodSMS
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

		method = loginlogmodel.MethodOAuth + auser.AuthType
		account, member, err = ac.accountService.CreateOAuthAccount(c.Request.Context(), userInfo.Account())
	default:
		// 密码登录：password 为经 /app/conf 公钥 RSA 加密后的密文，
		// 先解密再走服务层 bcrypt 校验；解密失败不区分原因（防探测）。
		// 注意解密错误用独立变量名，避免 := 遮蔽外层 err 导致登录错误被吞。
		// 失败锁定（上线前整改 P2）：以登录名/手机号为标识，凭证校验前
		// 先查锁，凭证失败计一次，成功清零——不存在的账号同样计数，防枚举
		ident := auser.Name
		if ident == "" {
			ident = auser.Phone
		}
		if ac.loginGuard != nil && ac.loginGuard.Locked(c.Request.Context(), ident) {
			httpx.ResponseFailed(c, http.StatusTooManyRequests, errLoginLocked)
			return
		}
		plain, decryptErr := ac.pkiKeypair.Decrypt(auser.Password)
		if decryptErr != nil {
			httpx.ResponseFailed(c, http.StatusBadRequest, decryptErr)
			return
		}
		auser.Password = plain
		account, member, err = ac.accountService.Auth(c.Request.Context(), auser)
		if ac.loginGuard != nil && ident != "" {
			if errors.Is(err, service.ErrCredentialsInvalid) {
				ac.loginGuard.RecordFailure(c.Request.Context(), ident)
			} else if err == nil {
				ac.loginGuard.Reset(c.Request.Context(), ident)
			}
		}
	}
	if err != nil {
		httpx.ResponseFailed(c, http.StatusUnauthorized, err)
		return
	}

	token, err := ac.loginSession(c, account, member, auser.SetCookie, method)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusInternalServerError, err)
		return
	}

	ac.recordLogin(c, account, member, method)

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
// bearerToken 取当前请求令牌：Authorization Header 优先，回落 Cookie
func bearerToken(c *gin.Context) string {
	if fields := strings.Fields(c.Request.Header.Get("Authorization")); len(fields) == 2 &&
		strings.ToLower(fields[0]) == "bearer" && fields[1] != "" {
		return fields[1]
	}
	token, _ := c.Cookie(httpx.CookieTokenName)
	return token
}

// Logout 登出（P2-8）：清 Cookie 并吊销当前 Bearer 令牌（jti 拉黑至自然
// 过期）——仅清 Cookie 无法让已签发/泄露的 token 失效。
// failClosed 模式（auth.revokeFailClosed=true）下吊销写失败如实返回 503：
// 黑名单没写成时令牌仍有效，静默清 Cookie 装作登出成功会误导用户；
// 默认 fail-open 模式维持「告警不阻断」（可用性优先）
func (ac *AuthController) Logout(c *gin.Context) {
	claims, _ := ac.jwtService.ParseToken(bearerToken(c))

	// 会话化登出（ADR-009）：撤销设备会话；存量无 sid 令牌回退 jti 黑名单
	if ac.sessions != nil && claims != nil && claims.SID != "" {
		if err := ac.sessions.Revoke(c.Request.Context(), claims.SID, secmodel.RevokeLogout); err != nil {
			logrus.Warnf("revoke session on logout: %v", err)
		}
	} else if ac.revoker != nil && claims != nil && claims.ExpiresAt != nil {
		ttl := time.Until(claims.ExpiresAt.Time)
		if err := ac.revoker.Revoke(c.Request.Context(), claims.ID, ttl); err != nil {
			logrus.Warnf("revoke token on logout: %v", err)
			if ac.revoker.FailClosed() {
				// 吊销写失败如实报错（与读侧同码 AUTH_REVOKE_CHECK_FAILED）：
				// 黑名单没写成时令牌仍有效，静默清 Cookie 装作登出成功会误导用户
				httpx.ResponseFailed(c, http.StatusServiceUnavailable, middleware.ErrRevokerUnavailable)
				return
			}
		}
	}

	c.SetCookie(httpx.CookieTokenName, "", -1, "/", "", true, true)
	c.SetCookie(httpx.CookieLoginUser, "", -1, "/", "", true, false)
	httpx.ResponseSuccess(c, nil)
}

// registerOnboarding 注册向导第 3 步采集的账号画像（「人」的属性挂账号）。
// role/channel 与前端向导一致为必填（上线前整改 P2 统一契约）：防止非前端
// 客户端绕过表单校验写入不完整画像
type registerOnboarding struct {
	Role    string `json:"role" binding:"required"`    // 你的角色：ceo / manager / it / ...
	Channel string `json:"channel" binding:"required"` // 了解渠道：xiaohongshu / zhihu / referral / ...
}

// registerTenantProfile 注册向导第 2 步采集的企业画像（写入租户 Config）
type registerTenantProfile struct {
	Name     string `json:"name" binding:"required"`     // 企业名称
	Demand   string `json:"demand"`                      // 你的需求（单选，选填）
	Industry string `json:"industry" binding:"required"` // 所属行业（单选，与前端一致必填）
}

// registerRequest 注册向导最终提交请求（POST /auth/register）：三步纯前端
// 采集的全量数据——手机号+验证码、企业画像、账号画像，点「进入产品」时
// 一次性上送，此前向导各步不产生任何服务端写副作用
type registerRequest struct {
	Phone      string                `json:"phone" binding:"required,len=11"`          // 手机号（格式再经 sms 校验）
	SmsCode    string                `json:"smsCode" binding:"required,len=6"`         // 短信验证码
	Nickname   string                `json:"nickname" binding:"required,min=2,max=20"` // 怎么称呼你（与前端表单一致 2-20 必填）
	Onboarding registerOnboarding    `json:"onboarding" binding:"required"`            // 账号画像：角色/了解渠道（必填，枚举白名单校验）
	Tenant     registerTenantProfile `json:"tenant" binding:"required"`                // 企业画像
}

// 注册画像枚举白名单（P2-5：与前端选项一一对应，防任意客户端写入
// 超长/无效画像造成脏 JSONB；「其他」兜底项保证人工输入也能通过）
var (
	registerRoleEnums = map[string]struct{}{
		"ceo": {}, "manager": {}, "it": {}, "member": {}, "teacher": {}, "student": {},
	}
	registerChannelEnums = map[string]struct{}{
		"xiaohongshu": {}, "zhihu": {}, "referral": {}, "ai": {}, "search": {},
		"toutiao": {}, "shortvideo": {}, "wechat": {}, "other": {},
	}
	registerDemandEnums = map[string]struct{}{
		"低代码应用搭建": {}, "流程自动化": {}, "数据分析与报表": {}, "团队协作": {}, "其他": {},
	}
	registerIndustryEnums = map[string]struct{}{
		"互联网/软件": {}, "制造业": {}, "零售/电商": {}, "教育": {}, "金融": {},
		"医疗健康": {}, "建筑/房地产": {}, "专业服务": {}, "政府/事业单位": {}, "其他": {},
	}
)

// validateRegisterEnums 画像枚举校验：非空字段必须命中白名单（role/channel/
// industry 的必填性已由 binding 标签保证，此处只挡非法枚举值）
func validateRegisterEnums(req *registerRequest) error {
	invalid := httpx.NewBiz(httpx.CodeValidation, "注册画像包含无效选项", http.StatusBadRequest)
	if req.Onboarding.Role != "" {
		if _, ok := registerRoleEnums[req.Onboarding.Role]; !ok {
			return invalid
		}
	}
	if req.Onboarding.Channel != "" {
		if _, ok := registerChannelEnums[req.Onboarding.Channel]; !ok {
			return invalid
		}
	}
	if req.Tenant.Name != "" && (len([]rune(req.Tenant.Name)) < 2 || len([]rune(req.Tenant.Name)) > 50) {
		return httpx.NewBiz(httpx.CodeValidation, "企业名称需为 2 - 50 个字符", http.StatusBadRequest)
	}
	if req.Tenant.Demand != "" {
		if _, ok := registerDemandEnums[req.Tenant.Demand]; !ok {
			return invalid
		}
	}
	if req.Tenant.Industry != "" {
		if _, ok := registerIndustryEnums[req.Tenant.Industry]; !ok {
			return invalid
		}
	}
	return nil
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
// @Router /api/v1/auth/register [post]
func (ac *AuthController) RegisterComplete(c *gin.Context) {
	req := new(registerRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	if err := validateRegisterEnums(req); err != nil {
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
	token, err := ac.loginSession(c, result.Account, result.Member, false, secmodel.AuthMethodRegister)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusInternalServerError, err)
		return
	}

	// 已注册手机号的幂等重放同样构成一次会话建立，落登录日志
	ac.recordLogin(c, result.Account, result.Member, loginlogmodel.MethodRegister)

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
// @Failure 403 {object} httpx.Response "QUOTA_EXCEEDED·配额已用尽"
// @Failure 409 {object} httpx.Response "TENANT_CODE_DUPLICATED·租户编码已存在"
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

// rebind 场景用途（上线前复查 P3）：old=向「当前绑定手机号」发码（原身份
// 验证，目标必须等于会话账号当前手机号）；new=向「新手机号」发码（新号
// 持有验证，任意合法目标）。区分用途把「旧号码发」收窄到本人在用号码，
// 缩小登录态被盗后的短信骚扰面
const (
	smsPurposeRebindOld = "old"
	smsPurposeRebindNew = "new"
)

// smsSendRequest 发送验证码请求：scene 支持 login（登录）/ register（注册）/
// reset（密码找回）/ rebind（换绑手机号，需已登录且携带 purpose=old/new）
type smsSendRequest struct {
	Phone string `json:"phone" binding:"required"`
	Scene string `json:"scene" binding:"required"`
	// Purpose rebind 场景专用用途：old=验证当前绑定号 / new=验证新号（其余场景忽略）
	Purpose string `json:"purpose"`
}

// smsSendResult 发送结果：仅本地联调（devEcho=true）时回显验证码
type smsSendResult struct {
	Code string `json:"code,omitempty"`
}

// @Summary 发送短信验证码
// @Description 按场景发送 6 位短信验证码（场景 login=登录 / register=注册 /
// reset=密码找回 / rebind=换绑手机号）：login 场景仅向已注册手机号发送
// （未注册返回 401 引导注册）；rebind 需已登录并携带 purpose——purpose=old
// 时接收号码必须为账号当前绑定手机号，purpose=new 为待换绑的新号码。
// 默认 60 秒重发冷却、5 分钟有效期、单码最多试错 5 次后作废需重发；
// 单手机号与单 IP 各有自然日发送上限（跨场景合计）。开发通道（provider=dev）
// 固定验证码 666666，devEcho=true 时响应回显验证码（仅本地联调）
// @Accept json
// @Produce json
// @Tags 认证
// @Param body body controller.smsSendRequest true "phone and scene"
// @Success 200 {object} httpx.Response{data=controller.smsSendResult}
// @Failure 400 {object} httpx.Response "AUTH_PHONE_INVALID/AUTH_SMS_SCENE_INVALID·手机号或场景非法/VALIDATION·rebind 用途或接收号码不符合"
// @Failure 401 {object} httpx.Response "UNAUTHORIZED·rebind 场景未登录/AUTH_ACCOUNT_NOT_FOUND·手机号未注册（login 场景）"
// @Failure 429 {object} httpx.Response "AUTH_COOLDOWN·发送冷却中/AUTH_SMS_TOO_MANY_TRIES·试错超限/AUTH_SMS_DAILY_LIMIT·单手机号达日上限/AUTH_SMS_IP_LIMIT·当前网络达日上限"
// @Router /api/v1/auth/sms/send [post]
func (ac *AuthController) SendSmsCode(c *gin.Context) {
	req := new(smsSendRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	// rebind 换绑场景：必须已登录，且按用途收窄接收号码（复查 P3）
	if req.Scene == sms.SceneRebind {
		claims := ginctx.GetSession(c)
		if claims == nil || claims.AccountID == 0 {
			httpx.ResponseFailed(c, http.StatusUnauthorized, fmt.Errorf("rebind scene requires authentication"))
			return
		}
		switch req.Purpose {
		case smsPurposeRebindOld:
			// 旧号码验证：接收号码必须是本账号当前绑定手机号——登录态被盗的
			// 攻击者无法借 old 用途向任意第三方号码发骚扰短信
			account, err := ac.accountService.GetProfile(c.Request.Context(), claims.AccountID)
			if err != nil {
				httpx.ResponseFailed(c, http.StatusInternalServerError, err)
				return
			}
			if account.Phone == "" || account.Phone != req.Phone {
				httpx.ResponseFailed(c, http.StatusBadRequest,
					fmt.Errorf("验证当前手机号时，接收号码须为账号绑定的手机号"))
				return
			}
		case smsPurposeRebindNew:
			// 新号码验证：任意合法目标（格式与防刷限额由 sms 域把关）
		default:
			httpx.ResponseFailed(c, http.StatusBadRequest,
				fmt.Errorf("rebind 场景需指定 purpose（old=验证当前手机号 / new=验证新手机号）"))
			return
		}
	}

	// 登录场景发送前校验账号存在（产品确认口径）：未注册手机号不消耗短信，
	// 直接返回未注册稳定码，前端据此引导转注册。register/reset 场景不查——
	// register 的未注册是正常路径（已注册号也需支持幂等重放）；该口径以
	// 手机号注册状态可探测为已知代价，由通用 IP 限流与发码限额兜底
	if req.Scene == sms.SceneLogin {
		registered, err := ac.accountService.PhoneRegistered(c.Request.Context(), req.Phone)
		if err != nil {
			httpx.ResponseFailed(c, http.StatusInternalServerError, err)
			return
		}
		if !registered {
			httpx.ResponseFailed(c, http.StatusUnauthorized, service.ErrAccountNotFound)
			return
		}
	}

	// 来源 IP 透传：单 IP 日限额维度（跨手机号合计，防轮换手机号刷短信成本）
	code, err := ac.smsService.Send(c.Request.Context(), req.Scene, req.Phone, c.ClientIP())
	if err != nil {
		// 状态码/业务码由 BizError 自动映射（ADR-008）：冷却/试错/日限额 429，非法 400
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

// passwordResetRequest 密码找回请求：手机号 + reset 场景验证码 + RSA 加密的新密码。
// 密码字段与登录/个人中心改密共用 PKI 传输约定，控制器解密后才交服务层 bcrypt。
type passwordResetRequest struct {
	Phone       string `json:"phone" binding:"required,len=11"`
	SmsCode     string `json:"smsCode" binding:"required,len=6"`
	NewPassword string `json:"newPassword" binding:"required"`
}

// @Summary 重设密码（忘记密码）
// @Description 凭「密码找回」场景（scene=reset）短信验证码一次性重设密码；
// newPassword 需经 GET /app/conf 下发的 RSA 公钥加密。验证码错误/过期、手机号
// 未注册均返回 401；重设成功会失效该账号全部既有会话，需用新密码重新登录
// @Accept json
// @Produce json
// @Tags 认证
// @Param body body controller.passwordResetRequest true "手机号/验证码/新密码"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response "AUTH_SMS_INVALID·验证码错误或已过期"
// @Failure 401 {object} httpx.Response "AUTH_ACCOUNT_NOT_FOUND·手机号未注册"
// @Router /api/v1/auth/password/reset [post]
func (ac *AuthController) ResetPassword(c *gin.Context) {
	req := new(passwordResetRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	// 先解密再消费验证码：浏览器公钥过期等传输错误不应浪费一次性验证码。
	plain, err := ac.pkiKeypair.Decrypt(req.NewPassword)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	// 验证码一次性校验（scene=reset 与登录/注册隔离）
	if err := ac.smsService.Verify(c.Request.Context(), sms.SceneReset, req.Phone, req.SmsCode); err != nil {
		httpx.ResponseFailed(c, http.StatusUnauthorized, err)
		return
	}

	if err := ac.accountService.ResetPasswordByPhone(c.Request.Context(), req.Phone, plain); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	httpx.ResponseSuccess(c, nil)
}

func (ac *AuthController) RegisterRoute(api *gin.RouterGroup) {
	api.POST("/auth/token", ac.Login)
	api.DELETE("/auth/token", ac.Logout)
	api.POST("/auth/register", ac.RegisterComplete)
	api.POST("/auth/password/reset", ac.ResetPassword)
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
// @Failure 403 {object} httpx.Response "AUTH_NOT_MEMBER·账号不属于目标租户"
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

	// 会话化重签（ADR-009）：复用 sid 递增 token_version——租户切换是同一次
	// 设备会话内换发令牌，不算新登录；存量无 sid 令牌回退为新签会话
	var session *secmodel.AccountSession
	if ac.sessions != nil && claims.SID != "" {
		session, err = ac.sessions.SwitchBump(c.Request.Context(), claims.SID)
		if err != nil {
			httpx.ResponseFailed(c, http.StatusUnauthorized, err)
			return
		}
	}

	token, err := ac.jwtService.CreateToken(account, member, session)
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
