package service

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"evolyn/internal/platform/iam/model"
	"evolyn/internal/platform/iam/repository"
	tenantmodel "evolyn/internal/platform/tenant/model"
	tenantrepository "evolyn/internal/platform/tenant/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// ---- 短信免密注册测试桩：自包含实现本组用例触及的仓储方法 ----

type phoneAccountRepo struct {
	repository.AccountRepository
	accounts  map[uint]*model.Account
	nextID    uint
	updatedPW map[uint]phonePWUpdate
}

type phonePWUpdate struct {
	hashed      string
	initialized bool
}

// boolPtr 帮助构造 *bool 桩值
func boolPtr(v bool) *bool { return &v }

func newPhoneAccountRepo(seed ...*model.Account) *phoneAccountRepo {
	repo := &phoneAccountRepo{
		accounts:  map[uint]*model.Account{},
		nextID:    100,
		updatedPW: map[uint]phonePWUpdate{},
	}
	for _, a := range seed {
		repo.accounts[a.ID] = a
	}
	return repo
}

func (f *phoneAccountRepo) GetByPhone(_ context.Context, phone string) (*model.Account, error) {
	for _, a := range f.accounts {
		if a.Phone == phone {
			return a, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *phoneAccountRepo) GetByName(_ context.Context, name string) (*model.Account, error) {
	for _, a := range f.accounts {
		if a.Name == name {
			return a, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *phoneAccountRepo) GetByID(_ context.Context, id uint) (*model.Account, error) {
	if a, ok := f.accounts[id]; ok {
		return a, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *phoneAccountRepo) Create(_ context.Context, account *model.Account) (*model.Account, error) {
	f.nextID++
	account.ID = f.nextID
	f.accounts[account.ID] = account
	return account, nil
}

func (f *phoneAccountRepo) UpdatePassword(_ context.Context, id uint, hashed string, initialized bool) error {
	if _, ok := f.accounts[id]; !ok {
		return gorm.ErrRecordNotFound
	}
	f.updatedPW[id] = phonePWUpdate{hashed: hashed, initialized: initialized}
	return nil
}

// Update 部分更新桩（换绑手机号用例）：只回放非空字段，与真实仓储口径一致
func (f *phoneAccountRepo) Update(_ context.Context, account *model.Account) (*model.Account, error) {
	cur, ok := f.accounts[account.ID]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	if account.Phone != "" {
		cur.Phone = account.Phone
	}
	if account.Nickname != "" {
		cur.Nickname = account.Nickname
	}
	return cur, nil
}

type phoneUserRepo struct {
	repository.UserRepository
	members map[uint]*model.User
	nextID  uint
}

func (f *phoneUserRepo) ListByAccount(_ context.Context, accountID uint) (model.Users, error) {
	users := make(model.Users, 0)
	for _, u := range f.members {
		if u.AccountId == accountID {
			users = append(users, *u)
		}
	}
	return users, nil
}

func (f *phoneUserRepo) Create(_ context.Context, member *model.User) (*model.User, error) {
	f.nextID++
	member.ID = f.nextID
	f.members[member.ID] = member
	return member, nil
}

// phoneTenantRepo 空实现租户仓储：登录成员优选（P1-2）按租户概要判断
// 自有/默认租户，查不到租户信息时优选逻辑回落第一个成员（桩数据无租户）
type phoneTenantRepo struct {
	tenantrepository.TenantRepository
}

func (f *phoneTenantRepo) GetByIDs(_ context.Context, _ []uint) ([]tenantmodel.Tenant, error) {
	return nil, nil
}

func newPhoneSvc(accounts *phoneAccountRepo, users *phoneUserRepo) AccountService {
	return NewAccountService(passThroughTx{}, accounts, users, &phoneTenantRepo{}, fakeQuota{}, nil)
}

// ---- RegisterByPhone：新手机号免密建号 ----

func TestRegisterByPhoneCreatesPasswordlessAccount(t *testing.T) {
	accounts, users := newPhoneAccountRepo(), &phoneUserRepo{members: map[uint]*model.User{}}
	svc := newPhoneSvc(accounts, users)

	account, member, created, err := svc.RegisterByPhone(context.Background(), "13800001111")
	assert.NoError(t, err)
	assert.True(t, created)

	// 随机登录名 u- 前缀，昵称为脱敏手机号，免密标记未初始化
	assert.Regexp(t, regexp.MustCompile(`^u-[0-9a-f]{8}$`), account.Name)
	assert.Equal(t, "138****1111", account.Nickname)
	require.NotNil(t, account.PasswordInitialized)
	assert.False(t, *account.PasswordInitialized)
	// 密码为服务端随机值的 bcrypt 散列（用户不可知，非明文）
	assert.True(t, strings.HasPrefix(account.Password, "$2"), "password should be a bcrypt hash")

	// 默认租户成员同事务建立，昵称取账号昵称
	assert.Equal(t, account.ID, member.AccountId)
	assert.Equal(t, "138****1111", member.Nickname)
	assert.NotEmpty(t, users.members)
}

// ---- RegisterByPhone：已注册手机号等价短信登录（幂等重试） ----

func TestRegisterByPhoneExistingPhoneLogsIn(t *testing.T) {
	seed := &model.Account{ID: 10, Name: "existing", Phone: "13800001111", PasswordInitialized: boolPtr(true)}
	accounts, users := newPhoneAccountRepo(seed), &phoneUserRepo{
		members: map[uint]*model.User{
			7: {ID: 7, AccountId: 10, Nickname: "member"},
		},
	}
	svc := newPhoneSvc(accounts, users)

	account, member, created, err := svc.RegisterByPhone(context.Background(), "13800001111")
	assert.NoError(t, err)
	assert.False(t, created)
	assert.Equal(t, uint(10), account.ID)
	assert.Equal(t, uint(7), member.ID)
	// 未新建账号/成员
	assert.Len(t, accounts.accounts, 1)
	assert.Len(t, users.members, 1)
}

// ---- RegisterByPhone：手机号格式校验 ----

func TestRegisterByPhoneInvalidPhone(t *testing.T) {
	svc := newPhoneSvc(newPhoneAccountRepo(), &phoneUserRepo{members: map[uint]*model.User{}})

	_, _, _, err := svc.RegisterByPhone(context.Background(), "12345")
	assert.Error(t, err)
}

// ---- ChangePassword：免密账号首设免旧密码，之后恢复常规校验 ----

func TestChangePasswordFirstSetSkipsOldPassword(t *testing.T) {
	accounts := newPhoneAccountRepo(&model.Account{
		ID: 10, Name: "sms-user", Phone: "13800001111",
		Password: "$2a$10$fakehash", PasswordInitialized: boolPtr(false),
	})
	svc := newPhoneSvc(accounts, &phoneUserRepo{members: map[uint]*model.User{}})

	// 免密注册账号：不传旧密码也可首次设置
	assert.NoError(t, svc.ChangePassword(context.Background(), 10, "", "newpass123"))
	update := accounts.updatedPW[10]
	assert.True(t, update.initialized)
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(update.hashed), []byte("newpass123")))
}

func TestChangePasswordRequiresOldPasswordWhenInitialized(t *testing.T) {
	hashed, _ := bcrypt.GenerateFromPassword([]byte("right-old"), bcrypt.DefaultCost)
	accounts := newPhoneAccountRepo(&model.Account{
		ID: 10, Name: "pwd-user", Password: string(hashed), PasswordInitialized: boolPtr(true),
	})
	svc := newPhoneSvc(accounts, &phoneUserRepo{members: map[uint]*model.User{}})

	// 已设置过密码：错误/缺失旧密码均拒绝
	assert.Error(t, svc.ChangePassword(context.Background(), 10, "wrong-old", "newpass123"))
	assert.Error(t, svc.ChangePassword(context.Background(), 10, "", "newpass123"))

	// 正确旧密码通过
	assert.NoError(t, svc.ChangePassword(context.Background(), 10, "right-old", "newpass123"))
	assert.True(t, accounts.updatedPW[10].initialized)
}
