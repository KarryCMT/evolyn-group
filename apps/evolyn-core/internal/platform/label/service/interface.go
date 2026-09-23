// Package service 编排标签模板草稿、发布版本、真实记录解析与正式渲染。
package service

import (
	"context"

	enginelabel "evolyn/internal/engine/label"
	iammodel "evolyn/internal/platform/iam/model"
	"evolyn/internal/platform/label/model"
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

type TemplateService interface {
	Create(ctx context.Context, member *iammodel.User, req *model.CreateTemplateRequest) (*model.TemplateDetail, error)
	List(ctx context.Context, member *iammodel.User, query model.ListTemplatesQuery) (*model.TemplatePage, error)
	Get(ctx context.Context, member *iammodel.User, code string) (*model.TemplateDetail, error)
	SaveDraft(ctx context.Context, member *iammodel.User, code string, req *model.SaveDraftRequest) (*model.SaveDraftResult, error)
	Publish(ctx context.Context, member *iammodel.User, code string, req *model.PublishRequest) (*model.PublishResult, error)
	Delete(ctx context.Context, member *iammodel.User, code string) error
	Preview(ctx context.Context, member *iammodel.User, code string, req *model.PreviewRequest) (*enginelabel.RenderResult, error)
	Render(ctx context.Context, member *iammodel.User, req *model.RenderRequest) (*enginelabel.RenderResult, error)
}
