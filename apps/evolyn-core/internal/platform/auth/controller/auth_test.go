package controller

import (
	"context"
	"testing"

	"evolyn/internal/platform/auth"
	securitymodel "evolyn/internal/platform/auth/security/model"
	securityservice "evolyn/internal/platform/auth/security/service"
	"evolyn/internal/platform/iam/model"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAuthControllerInjectsSessionService(t *testing.T) {
	sessions := new(stubSessionService)

	controller, ok := NewAuthController(
		nil, nil, nil, nil, nil, nil, nil, nil, nil, sessions, nil, nil,
	).(*AuthController)

	require.True(t, ok)
	assert.Same(t, sessions, controller.sessions)
}

func TestLoginSessionDoesNotIssueBeforeTokenPreparation(t *testing.T) {
	sessions := new(countingSessionService)
	controller := &AuthController{
		jwtService: auth.NewJWTService("test-secret"),
		sessions:   sessions,
	}
	ctx, _ := gin.CreateTestContext(nil)

	// account 为空会使 JWT 准备失败；此时不能触发单会话撤销事务。
	_, err := controller.loginSession(ctx, nil, &model.User{}, false, securitymodel.AuthMethodPassword, "")
	require.Error(t, err)
	assert.Zero(t, sessions.issueCalls)
}

// stubSessionService 仅验证控制器构造时的依赖注入；该用例不执行会话操作。
type stubSessionService struct{}

func (*stubSessionService) Issue(context.Context, securityservice.IssueRequest) (*securitymodel.AccountSession, error) {
	return nil, nil
}

func (*stubSessionService) Validate(context.Context, string, int64) error { return nil }

func (*stubSessionService) Revoke(context.Context, string, string) error { return nil }

func (*stubSessionService) SwitchBump(context.Context, string) (*securitymodel.AccountSession, error) {
	return nil, nil
}

// countingSessionService 用于验证登录输出尚未准备完成时不会触碰设备会话。
type countingSessionService struct {
	stubSessionService
	issueCalls int
}

func (s *countingSessionService) Issue(context.Context, securityservice.IssueRequest) (*securitymodel.AccountSession, error) {
	s.issueCalls++
	return nil, nil
}
