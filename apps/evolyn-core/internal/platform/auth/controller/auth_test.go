package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	kernel "evolyn/internal/model"
	"evolyn/internal/platform/auth"
	"evolyn/internal/platform/auth/oauth"
	securitymodel "evolyn/internal/platform/auth/security/model"
	securityservice "evolyn/internal/platform/auth/security/service"
	"evolyn/internal/platform/auth/sms"
	"evolyn/internal/platform/httpx"
	"evolyn/internal/platform/iam/model"
	iamservice "evolyn/internal/platform/iam/service"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
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

func TestWriteSessionCookieUsesHttpOnlyJWTWithoutLoginUserPayload(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/token", nil)

	controller := &AuthController{}
	controller.writeSessionCookie(ctx, "signed-jwt", true)

	headers := recorder.Header().Values("Set-Cookie")
	require.Len(t, headers, 2)
	assert.Contains(t, headers[0], "token=signed-jwt")
	assert.Contains(t, headers[0], "HttpOnly")
	assert.Contains(t, headers[0], "Secure")
	assert.Contains(t, headers[0], "SameSite=Lax")
	assert.Contains(t, headers[1], "sessionMode=persistent")
	assert.NotContains(t, strings.Join(headers, "\n"), "loginUser=", "成员资料不得落入浏览器 Cookie")
}

func TestLoginResponsesNeverSerializeJWT(t *testing.T) {
	body, err := json.Marshal(loginResult{})
	require.NoError(t, err)
	assert.NotContains(t, string(body), "token")
	assert.NotContains(t, string(body), "signed-jwt")
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

// mfaLoginGate 仅实现登录流程会触及的 MFA 方法；其余方法由嵌入接口承接，
// 若未来登录路径意外触发它们，nil 嵌入会立即暴露测试遗漏。
type mfaLoginGate struct {
	securityservice.MFAService
	inputs []securityservice.LoginChallengeInput
}

func (m *mfaLoginGate) Enabled(context.Context, uint) (bool, error) { return true, nil }

func (m *mfaLoginGate) CreateLoginChallenge(_ context.Context, input securityservice.LoginChallengeInput) (string, error) {
	m.inputs = append(m.inputs, input)
	return "mfa-login-challenge", nil
}

// loginAccountService 返回固定的已认证身份，并记录认证来源。控制器层测试只
// 关心“认证已通过后是否仍被 MFA 闸门拦住”，不重复覆盖 IAM 的账号解析细节。
type loginAccountService struct {
	iamservice.AccountService
	account    *model.Account
	member     *model.User
	smsCalls   int
	oauthCalls int
}

func (s *loginAccountService) AuthByPhone(context.Context, string, string) (*model.Account, *model.User, error) {
	s.smsCalls++
	return s.account, s.member, nil
}

func (s *loginAccountService) CreateOAuthAccount(context.Context, *model.Account) (*model.Account, *model.User, error) {
	s.oauthCalls++
	return s.account, s.member, nil
}

// loginSMSRedis 是短信 Login 流程所需的最小 Redis 替身。该测试只会调用
// Verify 的 Lua 原子消费；其余方法保留接口实现以防 sms.Service 扩展时失编译。
type loginSMSRedis struct{}

func (loginSMSRedis) Get(context.Context, string) *redis.StringCmd {
	cmd := redis.NewStringCmd(context.Background())
	cmd.SetErr(redis.Nil)
	return cmd
}

func (loginSMSRedis) Set(context.Context, string, interface{}, time.Duration) *redis.StatusCmd {
	return redis.NewStatusCmd(context.Background())
}

func (loginSMSRedis) SetNX(context.Context, string, interface{}, time.Duration) *redis.BoolCmd {
	cmd := redis.NewBoolCmd(context.Background())
	cmd.SetVal(true)
	return cmd
}

func (loginSMSRedis) Del(context.Context, ...string) *redis.IntCmd {
	cmd := redis.NewIntCmd(context.Background())
	cmd.SetVal(0)
	return cmd
}

func (loginSMSRedis) Incr(context.Context, string) *redis.IntCmd {
	cmd := redis.NewIntCmd(context.Background())
	cmd.SetVal(1)
	return cmd
}

func (loginSMSRedis) Expire(context.Context, string, time.Duration) *redis.BoolCmd {
	cmd := redis.NewBoolCmd(context.Background())
	cmd.SetVal(true)
	return cmd
}

func (loginSMSRedis) Eval(context.Context, string, []string, ...interface{}) *redis.Cmd {
	cmd := redis.NewCmd(context.Background())
	cmd.SetVal(int64(1))
	return cmd
}

type loginOAuthProvider struct{}

func (loginOAuthProvider) GetToken(string) (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: "test-oauth-token"}, nil
}

func (loginOAuthProvider) GetUserInfo(*oauth2.Token) (*oauth.UserInfo, error) {
	return &oauth.UserInfo{ID: "oauth-user", AuthType: oauth.GithubAuthType, Username: "oauth-user"}, nil
}

type loginOAuthResolver struct {
	requestedType string
}

func (r *loginOAuthResolver) GetAuthProvider(authType string) (oauth.AuthProvider, error) {
	r.requestedType = authType
	return loginOAuthProvider{}, nil
}

func newMFAGatedLoginController(accountService iamservice.AccountService, smsService *sms.Service, resolver OAuthProviderResolver, mfa securityservice.MFAService, sessions securityservice.SessionService) *AuthController {
	return &AuthController{
		accountService: accountService,
		jwtService:     auth.NewJWTService("test-secret"),
		smsService:     smsService,
		oauthManager:   resolver,
		mfa:            mfa,
		sessions:       sessions,
	}
}

func executeLogin(t *testing.T, controller *AuthController, body string) *httpx.Response {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/token", bytes.NewBufferString(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	controller.Login(ctx)

	response := new(httpx.Response)
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), response))
	assert.Equal(t, http.StatusOK, recorder.Code)
	return response
}

func assertMFALoginChallenge(t *testing.T, response *httpx.Response, gate *mfaLoginGate, expectedMethod string) {
	t.Helper()
	data, ok := response.Data.(map[string]interface{})
	require.True(t, ok)
	assert.True(t, data["mfaRequired"].(bool))
	assert.Equal(t, "mfa-login-challenge", data["mfaChallenge"])
	assert.NotContains(t, data, "token", "MFA 第一阶段绝不能签发 JWT")
	require.Len(t, gate.inputs, 1)
	assert.Equal(t, expectedMethod, gate.inputs[0].AuthMethod)
}

// 短信验证码已通过并不等于可跳过 MFA；必须先返回 challenge，且不得创建
// 设备会话或 JWT。该用例直接覆盖 HTTP 控制器而非只测内部帮助函数。
func TestLoginSMSRequiresMFAChallenge(t *testing.T) {
	accountService := &loginAccountService{
		account: &model.Account{ID: 11, Name: "sms-account"},
		member:  &model.User{ID: 12, TenantBaseModel: kernel.TenantBaseModel{TenantID: 13}},
	}
	gate := new(mfaLoginGate)
	sessions := new(countingSessionService)
	smsService := sms.NewService(loginSMSRedis{}, nil, sms.Options{})
	controller := newMFAGatedLoginController(accountService, smsService, nil, gate, sessions)

	response := executeLogin(t, controller, `{"phone":"13800001111","smsCode":"666666"}`)
	assertMFALoginChallenge(t, response, gate, securitymodel.AuthMethodSMS)
	assert.Equal(t, 1, accountService.smsCalls)
	assert.Zero(t, sessions.issueCalls)
}

// OAuth 第一阶段同样只能获得 MFA challenge。即使第三方 code 已换得用户资料，
// 也不能绕过本地账号启用的第二因子策略。
func TestLoginOAuthRequiresMFAChallenge(t *testing.T) {
	accountService := &loginAccountService{
		account: &model.Account{ID: 21, Name: "oauth-account"},
		member:  &model.User{ID: 22, TenantBaseModel: kernel.TenantBaseModel{TenantID: 23}},
	}
	gate := new(mfaLoginGate)
	sessions := new(countingSessionService)
	resolver := new(loginOAuthResolver)
	controller := newMFAGatedLoginController(accountService, nil, resolver, gate, sessions)

	response := executeLogin(t, controller, `{"authType":"github","authCode":"oauth-code"}`)
	assertMFALoginChallenge(t, response, gate, securitymodel.AuthMethodOAuth)
	assert.Equal(t, oauth.GithubAuthType, resolver.requestedType)
	assert.Equal(t, 1, accountService.oauthCalls)
	assert.Zero(t, sessions.issueCalls)
}
