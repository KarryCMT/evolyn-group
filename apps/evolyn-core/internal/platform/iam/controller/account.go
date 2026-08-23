package controller

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	kernel "evolyn/internal/model"
	loginlogservice "evolyn/internal/platform/auth/loginlog/service"
	"evolyn/internal/platform/auth/pki"
	platformcontroller "evolyn/internal/platform/controller"
	"evolyn/internal/platform/ginctx"
	"evolyn/internal/platform/httpx"
	"evolyn/internal/platform/iam/model"
	"evolyn/internal/platform/iam/service"

	"github.com/gin-gonic/gin"
)

// AccountController 账号自助（/accounts/me，P3-2）：资料、密码与手机号换绑
type AccountController struct {
	accountService service.AccountService
	// 登录口令加密密钥对：改密请求的密码字段先解密再落库
	pkiKeypair *pki.Keypair
	// 登录日志查询（账号自查）：认证域 loginlog 服务的只读契约
	loginLogQuery loginlogservice.QueryService
	// smsVerifier 换绑手机号验证码校验（认证域 sms.Service 实现）：iam 不
	// import 认证域（域依赖方向），以窄接口在装配层注入
	smsVerifier PhoneCodeVerifier
}

// PhoneCodeVerifier 换绑流程所需的验证码校验能力（*sms.Service 天然满足）
type PhoneCodeVerifier interface {
	Verify(ctx context.Context, scene, phone, code string) error
}

// ScenePhoneRebind 换绑手机号验证码场景（与认证域 sms.SceneRebind 同值约定；
// 换绑码的发送走 POST /auth/sms/send，该场景要求已登录）
const ScenePhoneRebind = "rebind"

func NewAccountController(accountService service.AccountService, pkiKeypair *pki.Keypair, loginLogQuery loginlogservice.QueryService, smsVerifier PhoneCodeVerifier) platformcontroller.Controller {
	return &AccountController{
		accountService: accountService,
		pkiKeypair:     pkiKeypair,
		loginLogQuery:  loginLogQuery,
		smsVerifier:    smsVerifier,
	}
}

// myAccount 从会话取账号；未认证返回 false 并已写响应
func (a *AccountController) myAccount(c *gin.Context) (uint, bool) {
	claims := ginctx.GetSession(c)
	if claims == nil || claims.AccountID == 0 {
		httpx.ResponseFailed(c, http.StatusUnauthorized, nil)
		return 0, false
	}
	return claims.AccountID, true
}

// @Summary 我的账号资料
// @Produce json
// @Tags 账号
// @Security JWT
// @Success 200 {object} httpx.Response{data=model.Account}
// @Router /api/v1/accounts/me [get]
func (a *AccountController) GetMe(c *gin.Context) {
	accountID, ok := a.myAccount(c)
	if !ok {
		return
	}

	account, err := a.accountService.GetProfile(c.Request.Context(), accountID)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusInternalServerError, err)
		return
	}
	httpx.ResponseSuccess(c, account)
}

// updateProfileRequest 资料更新（不含密码与手机号：手机号是登录与找回
// 密码凭据，须走专用换绑流程 PUT /accounts/me/phone）
type updateProfileRequest struct {
	Nickname   string                  `json:"nickname"`
	Email      string                  `json:"email"`
	Avatar     string                  `json:"avatar"`
	Onboarding model.AccountOnboarding `json:"onboarding"` // 注册引导画像（角色/了解渠道），注册向导第 3 步提交
}

// @Summary 更新我的账号资料
// @Description 自助更新账号资料；onboarding 为注册向导「完善信息」提交的角色与了解渠道画像，
// 昵称非空时同步当前成员的租户内称呼。手机号不在此入口变更（走 PUT /accounts/me/phone 换绑流程）
// @Accept json
// @Produce json
// @Tags 账号
// @Security JWT
// @Param profile body controller.updateProfileRequest true "profile"
// @Success 200 {object} httpx.Response{data=model.Account}
// @Router /api/v1/accounts/me [put]
func (a *AccountController) UpdateMe(c *gin.Context) {
	accountID, ok := a.myAccount(c)
	if !ok {
		return
	}

	req := new(updateProfileRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	account, err := a.accountService.UpdateProfile(c.Request.Context(), &model.Account{
		ID:         accountID,
		Nickname:   req.Nickname,
		Email:      req.Email,
		Avatar:     req.Avatar,
		Onboarding: req.Onboarding,
	})
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, account)
}

// changePasswordRequest 密码修改/首次设置：短信免密注册账号（未设置过密码）
// 首次设置时 oldPassword 可不传，其余情况必填；两个密码字段均需经
// GET /app/conf 下发的 RSA 公钥加密后上送
type changePasswordRequest struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword" binding:"required"`
}

// @Summary 修改登录密码
// @Description 修改登录密码；两个密码字段需先经 GET /app/conf 下发的 RSA 公钥加密
// 后上送。短信免密注册的账号（服务端随机密码）首次设置免旧密码，设置成功后恢复
// 常规校验；新密码需 8-64 位且同时包含字母和数字，常见弱口令被拒绝
// @Accept json
// @Produce json
// @Tags 账号
// @Security JWT
// @Param body body controller.changePasswordRequest true "passwords"
// @Success 200 {object} httpx.Response
// @Router /api/v1/accounts/me/password [put]
func (a *AccountController) ChangePassword(c *gin.Context) {
	accountID, ok := a.myAccount(c)
	if !ok {
		return
	}

	req := new(changePasswordRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	// 密文先解密：oldPassword 可空（免密账号首设），newPassword 必有
	if req.OldPassword != "" {
		oldPlain, err := a.pkiKeypair.Decrypt(req.OldPassword)
		if err != nil {
			httpx.ResponseFailed(c, http.StatusBadRequest, err)
			return
		}
		req.OldPassword = oldPlain
	}
	newPlain, err := a.pkiKeypair.Decrypt(req.NewPassword)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	if err := a.accountService.ChangePassword(c.Request.Context(), accountID, req.OldPassword, newPlain); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, nil)
}

// changePhoneRequest 换绑手机号请求：oldSmsCode 为「当前手机号」收到的
// rebind 场景验证码（账号已有手机号时必填）；newSmsCode 为「新手机号」
// 收到的 rebind 场景验证码。两个码均经 POST /auth/sms/send（scene=rebind）
// 发送：旧号用 purpose=old（接收号码必须为当前绑定号），新号用 purpose=new
type changePhoneRequest struct {
	OldSmsCode string `json:"oldSmsCode"`                         // 旧手机号验证码（无手机号账号首绑免传）
	NewPhone   string `json:"newPhone" binding:"required,len=11"` // 新手机号
	NewSmsCode string `json:"newSmsCode" binding:"required,len=6"`
}

// @Summary 换绑手机号
// @Description 专用换绑流程（上线前整改 P2）：验证旧手机号验证码（原身份持有
// 证明；OAuth 等无手机号账号首次绑定免验）→ 新手机号可用性预检 → 验证新
// 手机号验证码（新号持有证明）→ 落库。验证码经 POST /auth/sms/send
// （scene=rebind）分别发送到旧/新手机号，均为一次性消费；预检失败后继续
// 失败的码已消耗需重发（60 秒冷却）。换绑不失效既有会话
// @Accept json
// @Produce json
// @Tags 账号
// @Security JWT
// @Param body body controller.changePhoneRequest true "旧/新手机号验证码与新手机号"
// @Success 200 {object} httpx.Response{data=model.Account}
// @Failure 400 {object} httpx.Response "VALIDATION·缺少旧手机号验证码/AUTH_PHONE_INVALID·新手机号格式非法"
// @Failure 401 {object} httpx.Response "AUTH_SMS_INVALID·验证码错误或已过期"
// @Failure 409 {object} httpx.Response "DUPLICATE_PHONE·新手机号已被其他账号绑定"
// @Router /api/v1/accounts/me/phone [put]
func (a *AccountController) ChangeMyPhone(c *gin.Context) {
	accountID, ok := a.myAccount(c)
	if !ok {
		return
	}

	req := new(changePhoneRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	ctx := c.Request.Context()
	account, err := a.accountService.GetProfile(ctx, accountID)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusInternalServerError, err)
		return
	}

	// 第一步：旧手机号验证码（原身份持有证明）。已有手机号的账号必验——
	// 令牌被盗的攻击者无法收到旧手机号的验证码，恶意换绑被阻断；
	// 无手机号账号（OAuth 首登）没有可验证的旧身份，首次绑定免验
	if account.Phone != "" {
		if req.OldSmsCode == "" {
			httpx.ResponseFailed(c, http.StatusBadRequest, fmt.Errorf("请先通过当前手机号获取验证码"))
			return
		}
		if err := a.smsVerifier.Verify(ctx, ScenePhoneRebind, account.Phone, req.OldSmsCode); err != nil {
			httpx.ResponseFailed(c, http.StatusUnauthorized, err)
			return
		}
	}

	// 第二步：新号可用性预检（消费新号验证码之前，避免号码已占用时白白耗码）
	if err := a.accountService.EnsurePhoneAvailable(ctx, req.NewPhone); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	// 第三步：新手机号验证码（新号持有证明），一次性消费
	if err := a.smsVerifier.Verify(ctx, ScenePhoneRebind, req.NewPhone, req.NewSmsCode); err != nil {
		httpx.ResponseFailed(c, http.StatusUnauthorized, err)
		return
	}

	// 第四步：落库（服务层二次查重 + 唯一索引兜底并发竞态）
	updated, err := a.accountService.ChangePhone(ctx, accountID, req.NewPhone)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, updated)
}

// loginLogItem 登录日志展示项：字段与个人中心「账号设置-登录日志」抽屉
// 一一对应；client/method 为稳定枚举，展示文案由前端映射
type loginLogItem struct {
	LoggedAt kernel.JSONTime `json:"loggedAt"` // 登录时间（JSONTime 秒级东八区）
	IP       string          `json:"ip"`       // 来源 IP
	Location string          `json:"location"` // IP 归属地（内网地址/未知等兜底文案）
	Client   string          `json:"client"`   // web/wap/unknown
	Method   string          `json:"method"`   // password/sms/oauth_*/register
}

// loginLogPage 登录日志分页结果
type loginLogPage struct {
	Items []loginLogItem `json:"items"`
	Total int64          `json:"total"`
}

// queryInt 取整型查询参数，缺失/非法返回 0（由服务层规范化为默认值）
func queryInt(c *gin.Context, name string) int {
	v, _ := strconv.Atoi(c.Query(name))
	return v
}

// @Summary 我的登录日志
// @Description 分页查询当前账号的登录日志（仅本人，account 以会话为准）；
// startDate/endDate 为 yyyy-MM-dd 闭区间，按东八区自然日过滤；client 枚举
// web 电脑网页 / wap 手机网页 / unknown，method 枚举 password 密码 /
// sms 短信验证码 / oauth_github、oauth_wechat 第三方 / register 注册即登录，
// 两者的展示文案由前端映射
// @Produce json
// @Tags 账号
// @Security JWT
// @Param page query int false "页码，默认 1"
// @Param pageSize query int false "每页条数，默认 20，上限 100"
// @Param startDate query string false "开始日期 yyyy-MM-dd（闭区间）"
// @Param endDate query string false "结束日期 yyyy-MM-dd（闭区间）"
// @Success 200 {object} httpx.Response{data=controller.loginLogPage}
// @Failure 400 {object} httpx.Response "日期参数格式非法"
// @Router /api/v1/accounts/me/login-logs [get]
func (a *AccountController) MyLoginLogs(c *gin.Context) {
	accountID, ok := a.myAccount(c)
	if !ok {
		return
	}

	result, err := a.loginLogQuery.ListByAccount(c.Request.Context(), accountID, loginlogservice.PageQuery{
		Page:      queryInt(c, "page"),
		PageSize:  queryInt(c, "pageSize"),
		StartDate: c.Query("startDate"),
		EndDate:   c.Query("endDate"),
	})
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	items := make([]loginLogItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, loginLogItem{
			LoggedAt: item.CreatedAt,
			IP:       item.IP,
			Location: item.Location,
			Client:   item.Client,
			Method:   item.Method,
		})
	}
	httpx.ResponseSuccess(c, loginLogPage{Items: items, Total: result.Total})
}

func (a *AccountController) RegisterRoute(api *gin.RouterGroup) {
	api.GET("/accounts/me", a.GetMe)
	api.PUT("/accounts/me", a.UpdateMe)
	api.PUT("/accounts/me/password", a.ChangePassword)
	api.PUT("/accounts/me/phone", a.ChangeMyPhone)
	api.GET("/accounts/me/login-logs", a.MyLoginLogs)
}

func (a *AccountController) Name() string {
	return "Account"
}
