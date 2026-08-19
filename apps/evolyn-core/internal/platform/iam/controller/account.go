package controller

import (
	"net/http"

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
}

func NewAccountController(accountService service.AccountService, pkiKeypair *pki.Keypair) platformcontroller.Controller {
	return &AccountController{
		accountService: accountService,
		pkiKeypair:     pkiKeypair,
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

func (a *AccountController) RegisterRoute(api *gin.RouterGroup) {
	api.GET("/accounts/me", a.GetMe)
	api.PUT("/accounts/me", a.UpdateMe)
	api.PUT("/accounts/me/password", a.ChangePassword)
}

func (a *AccountController) Name() string {
	return "Account"
}
