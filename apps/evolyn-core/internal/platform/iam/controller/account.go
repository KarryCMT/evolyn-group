package controller

import (
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

// AccountController 账号自助（/accounts/me，P3-2）：资料与密码
type AccountController struct {
	accountService service.AccountService
	// 登录口令加密密钥对：改密请求的密码字段先解密再落库
	pkiKeypair *pki.Keypair
	// 登录日志查询（账号自查）：认证域 loginlog 服务的只读契约
	loginLogQuery loginlogservice.QueryService
}

func NewAccountController(accountService service.AccountService, pkiKeypair *pki.Keypair, loginLogQuery loginlogservice.QueryService) platformcontroller.Controller {
	return &AccountController{
		accountService: accountService,
		pkiKeypair:     pkiKeypair,
		loginLogQuery:  loginLogQuery,
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

// updateProfileRequest 资料更新（不含密码）
type updateProfileRequest struct {
	Nickname   string                  `json:"nickname"`
	Phone      string                  `json:"phone"`
	Email      string                  `json:"email"`
	Avatar     string                  `json:"avatar"`
	Onboarding model.AccountOnboarding `json:"onboarding"` // 注册引导画像（角色/了解渠道），注册向导第 3 步提交
}

// @Summary 更新我的账号资料
// @Description 自助更新账号资料；onboarding 为注册向导「完善信息」提交的角色与了解渠道画像，昵称非空时同步当前成员的租户内称呼
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
		Phone:      req.Phone,
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
// 常规校验；新密码至少 6 位
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
	api.GET("/accounts/me/login-logs", a.MyLoginLogs)
}

func (a *AccountController) Name() string {
	return "Account"
}
