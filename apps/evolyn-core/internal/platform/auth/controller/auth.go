package controller

import (
	"encoding/json"
	"evolyn/internal/platform/httpx"
	"net/http"

	"evolyn/internal/platform/auth"
	"evolyn/internal/platform/auth/oauth"
	platformcontroller "evolyn/internal/platform/controller"
	"evolyn/internal/platform/iam/model"
	"evolyn/internal/platform/iam/service"

	"github.com/gin-gonic/gin"
)

type AuthController struct {
	userService service.UserService
	jwtService  *auth.JWTService
	oauthManger *oauth.OAuthManager
}

func NewAuthController(userService service.UserService, jwtService *auth.JWTService, oauthManager *oauth.OAuthManager) platformcontroller.Controller {
	return &AuthController{
		userService: userService,
		jwtService:  jwtService,
		oauthManger: oauthManager,
	}
}

// @Summary Login
// @Description User login
// @Accept json
// @Produce json
// @Tags auth
// @Param user body model.AuthUser true "auth user info"
// @Success 200 {object} httpx.Response{data=model.JWTToken}
// @Router /api/v1/auth/token [post]
func (ac *AuthController) Login(c *gin.Context) {
	auser := new(model.AuthUser)
	if err := c.BindJSON(auser); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	var user *model.User
	var err error
	if !oauth.IsEmptyAuthType(auser.AuthType) && auser.Name == "" {
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

		user, err = ac.userService.CreateOAuthUser(c.Request.Context(), userInfo.User())
	} else {
		user, err = ac.userService.Auth(c.Request.Context(), auser)
	}
	if err != nil {
		httpx.ResponseFailed(c, http.StatusUnauthorized, err)
		return
	}

	token, err := ac.jwtService.CreateToken(user)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusInternalServerError, err)
		return
	}

	userJson, err := json.Marshal(user)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusInternalServerError, err)
		return
	}
	if auser.SetCookie {
		c.SetCookie(httpx.CookieTokenName, token, 3600*24, "/", "", true, true)
		c.SetCookie(httpx.CookieLoginUser, string(userJson), 3600*24, "/", "", true, false)
	}

	httpx.ResponseSuccess(c, model.JWTToken{
		Token:    token,
		Describe: "set token in Authorization Header, [Authorization: Bearer {token}]",
	})
}

// @Summary Logout
// @Description User logout
// @Produce json
// @Tags auth
// @Success 200 {object} httpx.Response
// @Router /api/v1/auth/token [delete]
func (ac *AuthController) Logout(c *gin.Context) {
	c.SetCookie(httpx.CookieTokenName, "", -1, "/", "", true, true)
	c.SetCookie(httpx.CookieLoginUser, "", -1, "/", "", true, false)
	httpx.ResponseSuccess(c, nil)
}

// @Summary Register user
// @Description Create user and storage
// @Accept json
// @Produce json
// @Tags auth
// @Param user body model.CreatedUser true "user info"
// @Success 200 {object} httpx.Response{data=model.User}
// @Router /api/v1/auth/user [post]
func (ac *AuthController) Register(c *gin.Context) {
	createdUser := new(model.CreatedUser)
	if err := c.BindJSON(createdUser); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	user := createdUser.GetUser()
	if err := ac.userService.Validate(user); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	ac.userService.Default(user)
	user, err := ac.userService.Create(c.Request.Context(), user)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusInternalServerError, err)
	}

	httpx.ResponseSuccess(c, user)
}

func (ac *AuthController) RegisterRoute(api *gin.RouterGroup) {
	api.POST("/auth/token", ac.Login)
	api.DELETE("/auth/token", ac.Logout)
	api.POST("/auth/user", ac.Register)
}

func (ac *AuthController) Name() string {
	return "Authentication"
}
