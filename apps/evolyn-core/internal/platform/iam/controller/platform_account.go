package controller

import (
	"net/http"
	"strconv"

	platformcontroller "evolyn/internal/platform/controller"
	"evolyn/internal/platform/httpx"
	iamservice "evolyn/internal/platform/iam/service"

	"github.com/gin-gonic/gin"
)

// PlatformAccountController 平台运营侧账号删除入口；自助注销同样复用删除服务，
// 两者均不能绕过租户创建人转移与成员关系清理。
type PlatformAccountController struct {
	service iamservice.AccountDeletionService
}

func NewPlatformAccountController(service iamservice.AccountDeletionService) platformcontroller.Controller {
	return &PlatformAccountController{service: service}
}

// @Summary 删除账号（平台）
// @Description 物理删除非租户创建人账号及其所有成员身份；账号仍是任一租户创建人时拒绝删除。
// @Produce json
// @Tags 平台管理
// @Security JWT
// @Param id path int true "account id"
// @Success 200 {object} httpx.Response
// @Failure 409 {object} httpx.Response "ACCOUNT_OWNS_TENANT·账号仍是租户创建人"
// @Router /api/v1/platform/accounts/{id} [delete]
func (a *PlatformAccountController) Delete(c *gin.Context) {
	accountID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || accountID == 0 {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	if err := a.service.Delete(c.Request.Context(), uint(accountID)); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, nil)
}

func (a *PlatformAccountController) RegisterRoute(api *gin.RouterGroup) {
	api.DELETE("/accounts/:id", a.Delete)
}

func (a *PlatformAccountController) Name() string { return "PlatformAccount" }

func (a *PlatformAccountController) Platform() bool { return true }
