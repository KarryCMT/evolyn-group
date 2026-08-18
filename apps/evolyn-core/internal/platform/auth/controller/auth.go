package controller

import (
	"encoding/json"
	"net/http"

	"evolyn/internal/platform/auth"
	"evolyn/internal/platform/auth/oauth"
	platformcontroller "evolyn/internal/platform/controller"
	"evolyn/internal/platform/httpx"
	"evolyn/internal/platform/iam/model"
	"evolyn/internal/platform/iam/service"

	"github.com/gin-gonic/gin"
)

// AuthController 会话域：登录/注册/登出（ADR-006 账号×成员模型）
type AuthController struct {
	accountService service.AccountService
	jwtService     *auth.JWTService
	oauthManger    *oauth.OAuthManager
}

func NewAuthController(accountService service.AccountService, jwtService *auth.JWTService, oauthManager *oauth.OAuthManager) platformcontroller.Controller {
	return &AuthController{
		accountService: accountService,
		jwtService:     jwtService,
		oauthManger:    oauthManager,
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

// @Summary Login
// @Description Account login (name/phone + password), tenant optional
// @Accept json
// @Produce json
// @Tags auth
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
	if !oauth.IsEmptyAuthType(auser.AuthType) && auser.Name == "" && auser.Phone == "" {
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
	} else {
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

// @Summary Logout
// @Description Logout and clear cookies
// @Produce json
// @Tags auth
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

// @Summary Register
// @Description Create account and default-tenant membership
// @Accept json
// @Produce json
// @Tags auth
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

func (ac *AuthController) RegisterRoute(api *gin.RouterGroup) {
	api.POST("/auth/token", ac.Login)
	api.DELETE("/auth/token", ac.Logout)
	api.POST("/auth/user", ac.Register)
}

func (ac *AuthController) Name() string {
	return "Authentication"
}
