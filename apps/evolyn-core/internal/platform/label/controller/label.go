// Package controller 暴露标签模板与正式渲染 HTTP API。
package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	platformcontroller "evolyn/internal/platform/controller"
	"evolyn/internal/platform/ginctx"
	"evolyn/internal/platform/httpx"
	labelapp "evolyn/internal/platform/label"
	"evolyn/internal/platform/label/model"
	labelservice "evolyn/internal/platform/label/service"

	"github.com/gin-gonic/gin"
)

type LabelController struct {
	service labelservice.TemplateService
}

func NewLabelController(service labelservice.TemplateService) platformcontroller.Controller {
	return &LabelController{service: service}
}

func responseError(c *gin.Context, err error) {
	var biz *httpx.BizError
	if errors.As(err, &biz) && biz.HTTP != 0 {
		httpx.ResponseFailed(c, biz.HTTP, err)
		return
	}
	httpx.ResponseFailed(c, http.StatusInternalServerError, err)
}

func templateCode(c *gin.Context) (string, bool) {
	code := strings.TrimSpace(c.Param("code"))
	if !strings.HasPrefix(code, "label_") || len(code) <= len("label_") {
		httpx.ResponseFailed(c, http.StatusBadRequest, labelapp.ErrCodeInvalid)
		return "", false
	}
	return code, true
}

// @Summary 创建标签模板
// @Tags 二维码标签
// @Security JWT
// @Accept json
// @Produce json
// @Param body body model.CreateTemplateRequest true "模板信息与绑定表单"
// @Success 201 {object} httpx.Response{data=model.TemplateDetail}
// @Router /api/v1/label-templates [post]
func (f *LabelController) Create(c *gin.Context) {
	req := new(model.CreateTemplateRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	detail, err := f.service.Create(c.Request.Context(), ginctx.GetUser(c), req)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.NewResponse(c, http.StatusCreated, detail, "创建成功")
}

// @Summary 标签模板列表
// @Tags 二维码标签
// @Security JWT
// @Produce json
// @Router /api/v1/label-templates [get]
func (f *LabelController) List(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	page, err := f.service.List(c.Request.Context(), ginctx.GetUser(c), model.ListTemplatesQuery{
		Limit: limit, Cursor: c.Query("cursor"), Keyword: c.Query("keyword"),
		Status: c.Query("status"), FormCode: c.Query("formCode"),
	})
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, page)
}

// @Summary 标签模板详情
// @Tags 二维码标签
// @Security JWT
// @Produce json
// @Router /api/v1/label-templates/{code} [get]
func (f *LabelController) Get(c *gin.Context) {
	code, ok := templateCode(c)
	if !ok {
		return
	}
	detail, err := f.service.Get(c.Request.Context(), ginctx.GetUser(c), code)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, detail)
}

// @Summary 保存标签草稿
// @Tags 二维码标签
// @Security JWT
// @Accept json
// @Produce json
// @Router /api/v1/label-templates/{code}/draft [put]
func (f *LabelController) SaveDraft(c *gin.Context) {
	code, ok := templateCode(c)
	if !ok {
		return
	}
	req := new(model.SaveDraftRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	result, err := f.service.SaveDraft(c.Request.Context(), ginctx.GetUser(c), code, req)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, result)
}

// @Summary 发布标签模板
// @Tags 二维码标签
// @Security JWT
// @Accept json
// @Produce json
// @Router /api/v1/label-templates/{code}/publish [post]
func (f *LabelController) Publish(c *gin.Context) {
	code, ok := templateCode(c)
	if !ok {
		return
	}
	req := new(model.PublishRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	result, err := f.service.Publish(c.Request.Context(), ginctx.GetUser(c), code, req)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, result)
}

// @Summary 使用草稿和真实记录预览标签
// @Tags 二维码标签
// @Security JWT
// @Accept json
// @Produce image/svg+xml image/png application/pdf
// @Router /api/v1/label-templates/{code}/preview [post]
func (f *LabelController) Preview(c *gin.Context) {
	code, ok := templateCode(c)
	if !ok {
		return
	}
	req := new(model.PreviewRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	result, err := f.service.Preview(c.Request.Context(), ginctx.GetUser(c), code, req)
	if err != nil {
		responseError(c, err)
		return
	}
	c.Data(http.StatusOK, result.MIMEType, result.Content)
}

// @Summary 使用已发布版本正式渲染单个标签
// @Tags 二维码标签
// @Security JWT
// @Accept json
// @Produce image/svg+xml image/png application/pdf
// @Router /api/v1/labels/render [post]
func (f *LabelController) Render(c *gin.Context) {
	req := new(model.RenderRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	result, err := f.service.Render(c.Request.Context(), ginctx.GetUser(c), req)
	if err != nil {
		responseError(c, err)
		return
	}
	extension := "svg"
	if result.MIMEType == "image/png" {
		extension = "png"
	} else if result.MIMEType == "application/pdf" {
		extension = "pdf"
	}
	c.Header("Content-Disposition", "attachment; filename=label."+extension)
	c.Data(http.StatusOK, result.MIMEType, result.Content)
}

// BatchRender 创建固定发布快照的异步多页 PDF 任务。
// @Summary 创建批量标签渲染任务
// @Tags 二维码标签
// @Security JWT
// @Accept json
// @Produce json
// @Param body body model.BatchRenderRequest true "批量渲染参数"
// @Success 202 {object} httpx.Response{data=model.BatchRenderCreated}
// @Router /api/v1/labels/batch-render [post]
func (f *LabelController) BatchRender(c *gin.Context) {
	req := new(model.BatchRenderRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	result, err := f.service.BatchRender(c.Request.Context(), ginctx.GetUser(c), req)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.NewResponse(c, http.StatusAccepted, result, "任务已创建")
}

func (f *LabelController) GetRenderTask(c *gin.Context) {
	result, err := f.service.GetRenderTask(c.Request.Context(), ginctx.GetUser(c), strings.TrimSpace(c.Param("taskCode")))
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, result)
}

func (f *LabelController) DownloadRenderTask(c *gin.Context) {
	result, err := f.service.DownloadRenderTask(c.Request.Context(), ginctx.GetUser(c), strings.TrimSpace(c.Param("taskCode")))
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, result)
}

// ResolveQRToken 在认证和租户中间件之后解析扫码目标。服务层会再次执行
// 表单记录权限校验，并只返回前端定位所需的稳定公开编码。
// @Summary 解析二维码定位目标
// @Tags 二维码标签
// @Security JWT
// @Produce json
// @Param token path string true "二维码短 Token"
// @Success 200 {object} httpx.Response{data=model.QRTokenTarget}
// @Failure 404 {object} httpx.Response
// @Router /api/v1/labels/qr-tokens/{token} [get]
func (f *LabelController) ResolveQRToken(c *gin.Context) {
	result, err := f.service.ResolveQRToken(c.Request.Context(), ginctx.GetUser(c), strings.TrimSpace(c.Param("token")))
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, result)
}

func (f *LabelController) Delete(c *gin.Context) {
	code, ok := templateCode(c)
	if !ok {
		return
	}
	if err := f.service.Delete(c.Request.Context(), ginctx.GetUser(c), code); err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, nil)
}

func (f *LabelController) RegisterRoute(api *gin.RouterGroup) {
	api.POST("/label-templates", f.Create)
	api.GET("/label-templates", f.List)
	api.GET("/label-templates/:code", f.Get)
	api.PUT("/label-templates/:code/draft", f.SaveDraft)
	api.POST("/label-templates/:code/publish", f.Publish)
	api.POST("/label-templates/:code/preview", f.Preview)
	api.DELETE("/label-templates/:code", f.Delete)
	api.POST("/labels/render", f.Render)
	api.POST("/labels/batch-render", f.BatchRender)
	api.GET("/labels/render-tasks/:taskCode", f.GetRenderTask)
	api.GET("/labels/render-tasks/:taskCode/download", f.DownloadRenderTask)
	api.GET("/labels/qr-tokens/:token", f.ResolveQRToken)
}

func (f *LabelController) Name() string { return "Label" }
