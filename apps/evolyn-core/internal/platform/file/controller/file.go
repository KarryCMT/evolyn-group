// Package controller 提供文件上传会话接口；文件字节始终由浏览器直传 RustFS，
// 本层只处理 HTTP 与统一响应，不读取 multipart 文件流。
package controller

import (
	"errors"
	"net/http"
	"strings"

	filemodel "evolyn/internal/platform/file/model"
	fileservice "evolyn/internal/platform/file/service"
	"evolyn/internal/platform/ginctx"
	"evolyn/internal/platform/httpx"

	"github.com/gin-gonic/gin"
)

type FileController struct{ service fileservice.FileService }

func NewFileController(service fileservice.FileService) *FileController {
	return &FileController{service: service}
}

func (f *FileController) Name() string { return "files" }

func (f *FileController) RegisterRoute(router *gin.RouterGroup) {
	files := router.Group("/files")
	files.POST("/uploads", f.CreateUpload)
	files.POST("/:id/complete", f.Complete)
	files.GET("/:id", f.Get)
	files.GET("/:id/download-url", f.DownloadURL)
	files.DELETE("/:id", f.Delete)
}

func (f *FileController) RegisterPublicRoute(router *gin.RouterGroup) {
	router.GET("/files/:id/content", f.PublicDownload)
}

// @Summary 创建文件上传会话
// @Description 校验当前租户成员、文件元数据和存储配额后，返回 RustFS 私有对象的短期直传 URL；客户端不得传入 bucket、objectKey 或 tenantId
// @Accept json
// @Produce json
// @Tags 文件管理
// @Security JWT
// @Param file body filemodel.CreateUploadRequest true "文件元数据"
// @Success 201 {object} httpx.Response{data=filemodel.UploadDetail}
// @Failure 400 {object} httpx.Response "errCode=FILE_REQUEST_INVALID/FILE_TOO_LARGE"
// @Failure 403 {object} httpx.Response "errCode=QUOTA_EXCEEDED/FORBIDDEN"
// @Failure 503 {object} httpx.Response "errCode=FILE_STORAGE_DISABLED"
// @Router /api/v1/files/uploads [post]
func (f *FileController) CreateUpload(c *gin.Context) {
	req := new(filemodel.CreateUploadRequest)
	if err := c.BindJSON(req); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	detail, err := f.service.CreateUpload(c.Request.Context(), ginctx.GetUser(c), req)
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.NewResponse(c, http.StatusCreated, detail, "上传会话已创建")
}

// @Summary 确认文件上传完成
// @Description 服务端读取 RustFS 对象元数据并校验大小，成功后文件状态变为 ready；重复确认同一已完成文件返回成功
// @Produce json
// @Tags 文件管理
// @Security JWT
// @Param id path string true "文件 ID"
// @Success 200 {object} httpx.Response{data=filemodel.File}
// @Failure 400 {object} httpx.Response "errCode=FILE_OBJECT_INVALID"
// @Failure 404 {object} httpx.Response "errCode=FILE_NOT_FOUND"
// @Failure 409 {object} httpx.Response "errCode=FILE_STATE_INVALID"
// @Router /api/v1/files/{id}/complete [post]
func (f *FileController) Complete(c *gin.Context) {
	file, err := f.service.Complete(c.Request.Context(), ginctx.GetUser(c), fileCode(c))
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, file)
}

// @Summary 查询文件元数据
// @Description 查询当前成员创建且归属当前租户的文件元数据；业务附件引用权限落地后将改为按引用资源授权
// @Produce json
// @Tags 文件管理
// @Security JWT
// @Param id path string true "文件 ID"
// @Success 200 {object} httpx.Response{data=filemodel.File}
// @Failure 404 {object} httpx.Response "errCode=FILE_NOT_FOUND"
// @Router /api/v1/files/{id} [get]
func (f *FileController) Get(c *gin.Context) {
	file, err := f.service.Get(c.Request.Context(), ginctx.GetUser(c), fileCode(c))
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, file)
}

// @Summary 获取文件下载地址
// @Description 对已完成文件签发短期私有下载 URL，不返回固定公开地址
// @Produce json
// @Tags 文件管理
// @Security JWT
// @Param id path string true "文件 ID"
// @Success 200 {object} httpx.Response{data=filemodel.DownloadURLResponse}
// @Failure 404 {object} httpx.Response "errCode=FILE_NOT_FOUND"
// @Failure 409 {object} httpx.Response "errCode=FILE_STATE_INVALID"
// @Router /api/v1/files/{id}/download-url [get]
func (f *FileController) DownloadURL(c *gin.Context) {
	result, err := f.service.DownloadURL(c.Request.Context(), ginctx.GetUser(c), fileCode(c))
	if err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, result)
}

// PublicDownload 仅返回临时对象跳转地址，不泄露底层对象键。
func (f *FileController) PublicDownload(c *gin.Context) {
	result, err := f.service.PublicDownloadURL(c.Request.Context(), fileCode(c))
	if err != nil {
		responseError(c, err)
		return
	}
	http.Redirect(c.Writer, c.Request, result.URL, http.StatusTemporaryRedirect)
}

// @Summary 删除文件
// @Description 删除 RustFS 私有对象并软删除元数据；失败时保留元数据以便重试，不会静默释放存储配额
// @Produce json
// @Tags 文件管理
// @Security JWT
// @Param id path string true "文件 ID"
// @Success 200 {object} httpx.Response
// @Failure 404 {object} httpx.Response "errCode=FILE_NOT_FOUND"
// @Router /api/v1/files/{id} [delete]
func (f *FileController) Delete(c *gin.Context) {
	if err := f.service.Delete(c.Request.Context(), ginctx.GetUser(c), fileCode(c)); err != nil {
		responseError(c, err)
		return
	}
	httpx.ResponseSuccess(c, nil)
}

func fileCode(c *gin.Context) string { return strings.TrimSpace(c.Param("id")) }

// responseError 是文件域 Controller 的脱敏错误出口。BizError 保留稳定错误码，
// 其余存储/数据库错误统一 500，避免把 S3 内部错误和对象键回显给客户端。
func responseError(c *gin.Context, err error) {
	var biz *httpx.BizError
	if errors.As(err, &biz) && biz.HTTP != 0 {
		httpx.ResponseFailed(c, biz.HTTP, err)
		return
	}
	httpx.ResponseFailed(c, http.StatusInternalServerError, err)
}
