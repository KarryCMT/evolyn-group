package service

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"evolyn/internal/contextx"
	enginelabel "evolyn/internal/engine/label"
	kernel "evolyn/internal/model"
	"evolyn/internal/platform/httpx"
	iammodel "evolyn/internal/platform/iam/model"
	labelapp "evolyn/internal/platform/label"
	"evolyn/internal/platform/label/model"

	"github.com/stretchr/testify/require"
)

type qrTokenRepoStub struct {
	token       *model.QRToken
	increments  int
	ensureCalls int
}

func (s *qrTokenRepoStub) Ensure(context.Context, *model.QRToken) (*model.QRToken, error) {
	s.ensureCalls++
	return s.token, nil
}
func (s *qrTokenRepoStub) GetByToken(context.Context, string) (*model.QRToken, error) {
	return s.token, nil
}
func (s *qrTokenRepoStub) IncrementScan(context.Context, uint) error {
	s.increments++
	return nil
}
func (s *qrTokenRepoStub) Migrate() error { return nil }

type qrFormDirectoryStub struct{ view FormView }

func (s qrFormDirectoryStub) FormByCode(context.Context, string) (FormView, bool, error) {
	return s.view, false, nil
}
func (s qrFormDirectoryStub) PublishedForm(context.Context, uint) (FormView, bool, error) {
	return s.view, false, nil
}

type qrAppDirectoryStub struct{ code string }

func (s qrAppDirectoryStub) AppCodeByID(context.Context, uint) (string, error) { return s.code, nil }

func TestQRTokenShapeAndSchemaUsage(t *testing.T) {
	token, err := newQRToken()
	require.NoError(t, err)
	require.True(t, validQRToken(token))
	require.False(t, validQRToken("predictable"))
	require.True(t, usesScanToken(&enginelabel.Schema{Elements: []enginelabel.Element{{
		Type: "qrcode", Visible: true, Value: &enginelabel.ValueSource{Type: "system", Key: "recordId"},
	}}}))
}

func TestAttachQRURLReplacesRecordIDWithPublicScanURL(t *testing.T) {
	tokenValue := base64.RawURLEncoding.EncodeToString([]byte("0123456789abcdef01234567"))
	repo := &qrTokenRepoStub{token: &model.QRToken{Token: tokenValue}}
	service := NewTemplateService(
		immediateTx{}, &templateRepoStub{}, versionRepoStub{}, qrFormDirectoryStub{},
		recordResolverStub{}, accessStub{}, nil,
	)
	ConfigureQR(service, repo, qrAppDirectoryStub{}, "https://app.lingyanyun.com/")
	member := &iammodel.User{ID: 7}
	member.TenantID = 1
	system := map[string]any{"recordId": "42"}

	err := service.(*templateService).attachQRURL(
		contextx.NewTenantContext(context.Background(), 1), member, 2, 3, 4, 42, system,
	)
	require.NoError(t, err)
	require.Equal(t, 1, repo.ensureCalls)
	require.Equal(t, "https://app.lingyanyun.com/q/"+tokenValue, system["recordId"])
}

func TestResolveQRTokenRejectsDisabledOrExpiredWithoutCounting(t *testing.T) {
	tokenValue := base64.RawURLEncoding.EncodeToString([]byte("0123456789abcdef01234567"))
	for _, test := range []struct {
		name      string
		enabled   bool
		expiredAt *time.Time
	}{
		{name: "disabled", enabled: false},
		{name: "expired", enabled: true, expiredAt: ptr(time.Now().Add(-time.Minute))},
	} {
		t.Run(test.name, func(t *testing.T) {
			row := &model.QRToken{Token: tokenValue, TargetType: model.QRTargetFormRecord, Enabled: test.enabled}
			if test.expiredAt != nil {
				expires := kernel.JSONTime(*test.expiredAt)
				row.ExpireAt = &expires
			}
			repo := &qrTokenRepoStub{token: row}
			service := NewTemplateService(
				immediateTx{}, &templateRepoStub{}, versionRepoStub{}, qrFormDirectoryStub{},
				recordResolverStub{}, accessStub{}, nil,
			)
			ConfigureQR(service, repo, qrAppDirectoryStub{}, "https://app.lingyanyun.com")
			member := &iammodel.User{ID: 7}
			member.TenantID = 1

			_, err := service.ResolveQRToken(contextx.NewTenantContext(context.Background(), 1), member, tokenValue)
			require.True(t, errors.Is(err, labelapp.ErrQRTokenInvalid))
			require.Zero(t, repo.increments)
		})
	}
}

func ptr[T any](value T) *T { return &value }

func TestResolveQRTokenRechecksRecordPermissionBeforeCounting(t *testing.T) {
	tokenValue := base64.RawURLEncoding.EncodeToString([]byte("0123456789abcdef01234567"))
	repo := &qrTokenRepoStub{token: &model.QRToken{
		ID: 9, TenantID: 1, Token: tokenValue, AppID: 2, FormID: 3, RecordID: 4,
		TemplateID: 5, TargetType: model.QRTargetFormRecord, Enabled: true,
	}}
	ctx := contextx.NewTenantContext(context.Background(), 1)
	member := &iammodel.User{ID: 7}
	member.TenantID = 1
	service := NewTemplateService(
		immediateTx{}, &templateRepoStub{}, versionRepoStub{},
		qrFormDirectoryStub{view: FormView{ID: 3, Code: "form_assets", Published: true}},
		recordResolverStub{record: &RecordView{FormID: 3, Fields: map[string]any{}, System: map[string]any{}}},
		accessStub{}, nil,
	)
	ConfigureQR(service, repo, qrAppDirectoryStub{code: "app_assets"}, "https://app.lingyanyun.com")
	target, err := service.ResolveQRToken(ctx, member, tokenValue)
	require.NoError(t, err)
	require.Equal(t, &model.QRTokenTarget{
		TargetType: model.QRTargetFormRecord, AppCode: "app_assets", FormCode: "form_assets", RecordID: "4",
	}, target)
	require.Equal(t, 1, repo.increments)

	denied := NewTemplateService(
		immediateTx{}, &templateRepoStub{}, versionRepoStub{}, qrFormDirectoryStub{},
		recordResolverStub{err: httpx.NewBiz("FORM_RECORD_FORBIDDEN", "denied", 403)}, accessStub{}, nil,
	)
	ConfigureQR(denied, repo, qrAppDirectoryStub{code: "app_assets"}, "https://app.lingyanyun.com")
	_, err = denied.ResolveQRToken(ctx, member, tokenValue)
	require.True(t, errors.Is(err, labelapp.ErrQRTokenInvalid))
	require.Equal(t, 1, repo.increments)
}
