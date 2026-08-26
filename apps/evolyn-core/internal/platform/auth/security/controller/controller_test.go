package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"evolyn/internal/platform/auth"
	"evolyn/internal/platform/auth/security/service"
	"evolyn/internal/platform/ginctx"
	"evolyn/internal/platform/httpx"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestReauthPrerequisiteError(t *testing.T) {
	tests := []struct {
		name         string
		mfaAvailable bool
		sid          string
		want         error
	}{
		{
			name:         "MFA 服务未装配",
			mfaAvailable: false,
			sid:          "session-1",
			want:         service.ErrMFAUnavailable,
		},
		{
			name:         "存量令牌缺少设备会话",
			mfaAvailable: true,
			want:         service.ErrMFAReauthLoginRequired,
		},
		{
			name:         "已满足前置条件",
			mfaAvailable: true,
			sid:          "session-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.ErrorIs(t, reauthPrerequisiteError(tt.mfaAvailable, tt.sid), tt.want)
		})
	}
}

func TestSecurityControllerReauthPrerequisitesResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name          string
		mfaService    service.MFAService
		sid           string
		wantStatus    int
		wantErrorCode string
	}{
		{
			name:          "MFA 服务未配置",
			sid:           "session-1",
			wantStatus:    http.StatusServiceUnavailable,
			wantErrorCode: "AUTH_MFA_UNAVAILABLE",
		},
		{
			name:          "存量令牌缺少设备会话",
			mfaService:    stubMFAService{},
			wantStatus:    http.StatusUnauthorized,
			wantErrorCode: "AUTH_REAUTH_LOGIN_REQUIRED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/accounts/me/security/reauth", nil)
			ginctx.SetSession(ctx, &auth.CustomClaims{AccountID: 7, SID: tt.sid})

			(&SecurityController{mfaService: tt.mfaService}).Reauth(ctx)

			body := new(httpx.Response)
			assert.NoError(t, json.Unmarshal(recorder.Body.Bytes(), body))
			assert.Equal(t, tt.wantStatus, recorder.Code)
			assert.Equal(t, tt.wantErrorCode, body.ErrCode)
		})
	}
}

func TestCancelMyAccountRequiresReauthAndDeletesCurrentAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodDelete, "/api/v1/accounts/me", strings.NewReader(`{"reauthToken":"token-1"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ginctx.SetSession(ctx, &auth.CustomClaims{AccountID: 7, SID: "session-1"})

	deletion := &stubAccountDeletion{}
	(&SecurityController{mfaService: stubMFAService{}, accountDeletion: deletion}).CancelMyAccount(ctx)

	body := new(httpx.Response)
	assert.NoError(t, json.Unmarshal(recorder.Body.Bytes(), body))
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, uint(7), deletion.accountID)
}

// stubMFAService 仅用于验证 Reauth 的前置条件分支；这些方法在测试中均不应执行。
type stubMFAService struct{}

func (stubMFAService) Enabled(context.Context, uint) (bool, error) { return false, nil }

func (stubMFAService) Enroll(context.Context, uint, string, string) (*service.TOTPEnrollment, error) {
	return nil, nil
}

func (stubMFAService) ConfirmEnrollment(context.Context, uint, string, string, string) ([]string, error) {
	return nil, nil
}

func (stubMFAService) VerifyCode(context.Context, uint, string, string) (string, error) {
	return "", nil
}

func (stubMFAService) Disable(context.Context, uint, string) error { return nil }

func (stubMFAService) CreateLoginChallenge(context.Context, service.LoginChallengeInput) (string, error) {
	return "", nil
}

func (stubMFAService) ConsumeLoginChallenge(context.Context, string, string, string) (*service.LoginChallenge, string, error) {
	return nil, "", nil
}

func (stubMFAService) CreateReauthToken(context.Context, uint, string, string, string) (string, error) {
	return "", nil
}

func (stubMFAService) IssueReauthToken(context.Context, uint, string) (string, error) {
	return "", nil
}

func (stubMFAService) RequireReauth(context.Context, uint, string, string) error { return nil }

type stubAccountDeletion struct {
	accountID uint
}

func (s *stubAccountDeletion) Delete(_ context.Context, accountID uint) error {
	s.accountID = accountID
	return nil
}
