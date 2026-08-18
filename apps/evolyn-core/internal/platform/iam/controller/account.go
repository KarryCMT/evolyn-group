package controller

import (
	"net/http"

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
}

func NewAccountController(accountService service.AccountService) platformcontroller.Controller {
	return &AccountController{
		accountService: accountService,
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

// @Summary My account profile
// @Produce json
// @Tags account
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
	Nickname string `json:"nickname"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Avatar   string `json:"avatar"`
}

// @Summary Update my profile
// @Accept json
// @Produce json
// @Tags account
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
		ID:       accountID,
		Nickname: req.Nickname,
		Phone:    req.Phone,
		Email:    req.Email,
		Avatar:   req.Avatar,
	})
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, account)
}

// changePasswordRequest 密码修改
type changePasswordRequest struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required"`
}

// @Summary Change my password
// @Accept json
// @Produce json
// @Tags account
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

	if err := a.accountService.ChangePassword(c.Request.Context(), accountID, req.OldPassword, req.NewPassword); err != nil {
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
