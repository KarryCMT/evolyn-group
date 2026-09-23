package service

import (
	"context"
	"errors"
	"testing"

	"evolyn/internal/contextx"
	"evolyn/internal/platform/httpx"
	iammodel "evolyn/internal/platform/iam/model"
	labelapp "evolyn/internal/platform/label"
	"evolyn/internal/platform/label/model"
	"evolyn/internal/platform/label/repository"

	"github.com/stretchr/testify/require"
)

type immediateTx struct{}

func (immediateTx) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type templateRepoStub struct{ template *model.Template }

func (s *templateRepoStub) Create(context.Context, *model.Template) (*model.Template, error) {
	panic("unexpected Create")
}
func (s *templateRepoStub) GetByCode(context.Context, string) (*model.Template, error) {
	return s.template, nil
}
func (s *templateRepoStub) GetByCodeForUpdate(context.Context, string) (*model.Template, error) {
	return s.template, nil
}
func (*templateRepoStub) List(context.Context, repository.ListParams) ([]model.Template, bool, error) {
	panic("unexpected List")
}
func (*templateRepoStub) SaveDraft(context.Context, uint, int64, model.SchemaContent, float64, float64, string, int) (bool, error) {
	panic("unexpected SaveDraft")
}
func (*templateRepoStub) MarkPreviewed(context.Context, uint, int64) (bool, error) {
	panic("unexpected MarkPreviewed")
}
func (*templateRepoStub) MarkPublished(context.Context, uint, uint, int, int64) error {
	panic("unexpected MarkPublished")
}
func (*templateRepoStub) SoftDelete(context.Context, *model.Template) error {
	panic("unexpected SoftDelete")
}
func (*templateRepoStub) Migrate() error { return nil }

type versionRepoStub struct{}

func (versionRepoStub) MaxVersionNo(context.Context, uint) (int, error) {
	panic("unexpected MaxVersionNo")
}
func (versionRepoStub) Create(context.Context, *model.TemplateVersion) (*model.TemplateVersion, error) {
	panic("unexpected Create")
}
func (versionRepoStub) GetByID(context.Context, uint) (*model.TemplateVersion, error) {
	panic("unexpected GetByID")
}
func (versionRepoStub) Migrate() error { return nil }

type accessStub struct{ permissions map[string]bool }

func (s accessStub) Permissions(context.Context, *iammodel.User) map[string]bool {
	return s.permissions
}

type recordResolverStub struct {
	record *RecordView
	err    error
}

func (s recordResolverStub) GetRecord(context.Context, *iammodel.User, uint, uint) (*RecordView, error) {
	return s.record, s.err
}

func TestPublishRequiresRealDataPreviewForCurrentDraft(t *testing.T) {
	template := &model.Template{ID: 1, Code: "label_test", DraftRevision: 3, PreviewedDraftRevision: 2}
	template.TenantID = 1
	service := NewTemplateService(
		immediateTx{}, &templateRepoStub{template: template}, versionRepoStub{}, nil, nil,
		accessStub{permissions: map[string]bool{iammodel.LabelTemplateResource + ":create": true}}, nil,
	)
	ctx := contextx.NewTenantContext(context.Background(), 1)
	member := &iammodel.User{ID: 2}
	member.TenantID = 1
	_, err := service.Publish(ctx, member, "label_test", &model.PublishRequest{DraftRevision: 3})
	require.Error(t, err)
	require.True(t, errors.Is(err, labelapp.ErrRealPreviewRequired))
}

func TestTemplateAccessRejectsMissingPermissionAndCrossTenantRows(t *testing.T) {
	template := &model.Template{ID: 1, Code: "label_test"}
	template.TenantID = 1
	ctx := contextx.NewTenantContext(context.Background(), 2)
	member := &iammodel.User{ID: 2}
	member.TenantID = 2

	denied := NewTemplateService(
		immediateTx{}, &templateRepoStub{template: template}, versionRepoStub{}, nil, nil,
		accessStub{permissions: map[string]bool{}}, nil,
	)
	_, err := denied.Get(ctx, member, template.Code)
	require.ErrorIs(t, err, labelapp.ErrForbidden)

	permitted := NewTemplateService(
		immediateTx{}, &templateRepoStub{template: template}, versionRepoStub{}, nil, nil,
		accessStub{permissions: map[string]bool{iammodel.LabelTemplateResource + ":get": true}}, nil,
	)
	_, err = permitted.Get(ctx, member, template.Code)
	require.ErrorIs(t, err, labelapp.ErrTemplateNotFound)
}

func TestPreviewHidesRecordAndFieldAuthorizationDetails(t *testing.T) {
	template := &model.Template{
		ID: 1, Code: "label_test", FormID: 7, FormCode: "form_assets", DraftRevision: 1,
		DraftSchema: model.SchemaContent(`{
			"schemaVersion":"1.0","name":"资产标签",
			"page":{"width":80,"height":50,"unit":"mm","dpi":300,"background":"#ffffff"},
			"source":{"type":"form","formId":"form_assets"},
			"elements":[{"id":"field-1","type":"field","x":1,"y":1,"width":20,"height":5,"rotation":0,"zIndex":1,"visible":true,"locked":false,"value":{"type":"field","fieldId":"asset_code"},"style":{"fontFamily":"sans-serif","fontSize":4,"fontWeight":400,"color":"#111111","lineHeight":1.2,"textAlign":"left","verticalAlign":"top","overflow":"clip"}}],
			"settings":{"snapToGrid":true,"gridSize":1,"showGrid":false}
		}`),
	}
	template.TenantID = 1
	ctx := contextx.NewTenantContext(context.Background(), 1)
	member := &iammodel.User{ID: 2}
	member.TenantID = 1
	permissions := accessStub{permissions: map[string]bool{iammodel.LabelTemplateResource + ":create": true}}

	noRecord := NewTemplateService(
		immediateTx{}, &templateRepoStub{template: template}, versionRepoStub{}, nil,
		recordResolverStub{err: httpx.NewBiz("FORM_RECORD_FORBIDDEN", "internal detail", 403)},
		permissions, nil,
	)
	_, err := noRecord.Preview(ctx, member, template.Code, &model.PreviewRequest{RecordID: "1"})
	require.ErrorIs(t, err, labelapp.ErrRecordNoPermission)

	noField := NewTemplateService(
		immediateTx{}, &templateRepoStub{template: template}, versionRepoStub{}, nil,
		recordResolverStub{record: &RecordView{FormID: 7, Fields: map[string]any{}, System: map[string]any{}}},
		permissions, nil,
	)
	_, err = noField.Preview(ctx, member, template.Code, &model.PreviewRequest{RecordID: "1"})
	require.ErrorIs(t, err, labelapp.ErrFieldNoPermission)
}
