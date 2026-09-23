// Package service 编排标签模板草稿、发布版本、真实记录解析与正式渲染。
package service

import (
	"context"
	"strings"

	enginelabel "evolyn/internal/engine/label"
	kernel "evolyn/internal/model"
	iammodel "evolyn/internal/platform/iam/model"
	"evolyn/internal/platform/label/model"
	"evolyn/internal/platform/label/repository"
)

type TxManager interface {
	WithinTransaction(ctx context.Context, fn func(context.Context) error) error
}

type AccessEvaluator interface {
	Permissions(ctx context.Context, member *iammodel.User) map[string]bool
}

type FormView struct {
	ID        uint
	AppID     uint
	Code      string
	Name      string
	Published bool
	Fields    map[string]bool
}

// FormDirectory 由表单域仓储适配，标签域不读取表单 JSONB 或物理表。
type FormDirectory interface {
	FormByCode(ctx context.Context, code string) (FormView, bool, error)
	PublishedForm(ctx context.Context, id uint) (FormView, bool, error)
}

type RecordView struct {
	FormID uint
	Fields map[string]any
	System map[string]any
}

type RecordResolver interface {
	GetRecord(ctx context.Context, member *iammodel.User, formID, recordID uint) (*RecordView, error)
}

type MemberDirectory interface {
	MemberByID(ctx context.Context, memberID uint) (*iammodel.User, error)
}

type AppDirectory interface {
	AppCodeByID(ctx context.Context, appID uint) (string, error)
}

type BatchQueue interface {
	Available() bool
	Enqueue(ctx context.Context, tenantID uint, taskCode string) error
}

type ArtifactDownload struct {
	Method    string
	URL       string
	Headers   map[string]string
	ExpiresAt kernel.JSONTime
}

type ArtifactStore interface {
	Available() bool
	StorePDF(ctx context.Context, member *iammodel.User, filename, relativePath string, content []byte) (string, error)
	Download(ctx context.Context, member *iammodel.User, fileCode string) (*ArtifactDownload, error)
}

type TemplateService interface {
	Create(ctx context.Context, member *iammodel.User, req *model.CreateTemplateRequest) (*model.TemplateDetail, error)
	List(ctx context.Context, member *iammodel.User, query model.ListTemplatesQuery) (*model.TemplatePage, error)
	Get(ctx context.Context, member *iammodel.User, code string) (*model.TemplateDetail, error)
	SaveDraft(ctx context.Context, member *iammodel.User, code string, req *model.SaveDraftRequest) (*model.SaveDraftResult, error)
	Publish(ctx context.Context, member *iammodel.User, code string, req *model.PublishRequest) (*model.PublishResult, error)
	Delete(ctx context.Context, member *iammodel.User, code string) error
	Preview(ctx context.Context, member *iammodel.User, code string, req *model.PreviewRequest) (*enginelabel.RenderResult, error)
	Render(ctx context.Context, member *iammodel.User, req *model.RenderRequest) (*enginelabel.RenderResult, error)
	BatchRender(ctx context.Context, member *iammodel.User, req *model.BatchRenderRequest) (*model.BatchRenderCreated, error)
	GetRenderTask(ctx context.Context, member *iammodel.User, taskCode string) (*model.RenderTaskDetail, error)
	DownloadRenderTask(ctx context.Context, member *iammodel.User, taskCode string) (*ArtifactDownload, error)
	ProcessBatch(ctx context.Context, taskCode string) error
	FailBatch(ctx context.Context, taskCode string, cause error) error
	ResolveQRToken(ctx context.Context, member *iammodel.User, token string) (*model.QRTokenTarget, error)
}

// BatchDependencies 在应用装配层注入；构造函数保持现有模板单测所需的最小依赖。
func ConfigureBatch(service TemplateService, tasks repository.RenderTaskRepository, queue BatchQueue, artifacts ArtifactStore, members MemberDirectory) {
	if target, ok := service.(*templateService); ok {
		target.tasks, target.queue, target.artifacts, target.members = tasks, queue, artifacts, members
	}
}

func ConfigureQR(service TemplateService, tokens repository.QRTokenRepository, apps AppDirectory, publicBaseURL string) {
	if target, ok := service.(*templateService); ok {
		target.tokens, target.apps, target.publicBaseURL = tokens, apps, strings.TrimRight(strings.TrimSpace(publicBaseURL), "/")
	}
}
