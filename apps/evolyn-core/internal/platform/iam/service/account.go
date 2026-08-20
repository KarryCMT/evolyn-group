package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"regexp"

	"evolyn/internal/contextx"
	"evolyn/internal/platform/httpx"
	tenantmodel "evolyn/internal/platform/tenant/model"
	tenantrepository "evolyn/internal/platform/tenant/repository"
	tenantservice "evolyn/internal/platform/tenant/service"

	"evolyn/internal/platform/iam/model"
	"evolyn/internal/platform/iam/repository"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	MinPasswordLength = 6
)

// 手机号格式（与 sms 域同口径；iam 不反向依赖认证域，就地复制）
var accountPhonePattern = regexp.MustCompile(`^1[3-9]\d{9}$`)

type accountService struct {
	tx          TxManager
	accountRepo repository.AccountRepository
	userRepo    repository.UserRepository
	tenantRepo  tenantrepository.TenantRepository
	quota       tenantservice.QuotaService
}

// NewAccountService 账号服务：登录身份校验与账号生命周期；
// 依赖成员仓储（注册即建默认租户成员）、租户仓储（登录指定 TenantCode 时解析）、
// 配额服务与事务管理器（账号创建与默认成员同事务提交，不留半注册账号）
func NewAccountService(
	tx TxManager,
	accountRepo repository.AccountRepository,
	userRepo repository.UserRepository,
	tenantRepo tenantrepository.TenantRepository,
	quota tenantservice.QuotaService,
) AccountService {
	return &accountService{
		tx:          tx,
		accountRepo: accountRepo,
		userRepo:    userRepo,
		tenantRepo:  tenantRepo,
		quota:       quota,
	}
}

// Auth 账号密码校验：登录名或手机号定位账号 → bcrypt 比对 → 解析登录成员。
// TenantCode 非空时精确匹配该租户成员，否则取第一个成员关系（默认租户体验）
func (s *accountService) Auth(ctx context.Context, auser *model.AuthUser) (*model.Account, *model.User, error) {
	if auser == nil || (auser.Name == "" && auser.Phone == "") || auser.Password == "" {
		return nil, nil, fmt.Errorf("name/phone or password is empty")
	}

	var (
		account *model.Account
		err     error
	)
	if auser.Name != "" {
		account, err = s.accountRepo.GetByName(ctx, auser.Name)
	} else {
		account, err = s.accountRepo.GetByPhone(ctx, auser.Phone)
	}
	if err != nil {
		// 账号不存在与密码错误统一稳定码（ADR-008）：既不泄露 gorm 内部错误，
		// 也不区分「账号是否存在」
		return nil, nil, ErrCredentialsInvalid
	}

	if err := bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(auser.Password)); err != nil {
		return nil, nil, ErrCredentialsInvalid
	}

	member, err := s.resolveLoginMember(ctx, account, auser.TenantCode)
	if err != nil {
		return nil, nil, err
	}

	return account, member, nil
}

// AuthByPhone 验证码登录的账号解析（验证码已由调用方经 sms 域校验）：
// 手机号定位账号后复用登录成员解析（孤儿自愈 + TenantCode 语义）
func (s *accountService) AuthByPhone(ctx context.Context, phone, tenantCode string) (*model.Account, *model.User, error) {
	account, err := s.accountRepo.GetByPhone(ctx, phone)
	if err != nil {
		// 验证码已通过，未注册是独立语义（ADR-008）：可引导走注册
		return nil, nil, ErrAccountNotFound
	}

	member, err := s.resolveLoginMember(ctx, account, tenantCode)
	if err != nil {
		return nil, nil, err
	}

	return account, member, nil
}

// resolveLoginMember 登录成员解析：孤儿账号自愈补建默认成员；
// TenantCode 非空时精确匹配该租户成员，缺省取第一个成员关系（默认租户体验）。
// 全程剥离租户上下文（账号级跨租户查询）：登录请求可能携带其他账号的
// 旧 token，其租户上下文会污染成员列表导致误判孤儿
func (s *accountService) resolveLoginMember(ctx context.Context, account *model.Account, tenantCode string) (*model.User, error) {
	bctx := contextx.DetachTenant(ctx)

	members, err := s.userRepo.ListByAccount(bctx, account.ID)
	if err != nil {
		return nil, err
	}
	if len(members) == 0 {
		// 历史孤儿账号自愈：旧版注册无事务，成员创建失败会留下「有账号无身份」
		// 的半注册账号（登录/注册两头堵死）。登录凭证已验证，此处补建默认成员
		return s.createDefaultMember(bctx, account)
	}

	if tenantCode == "" {
		return &members[0], nil
	}

	tenant, err := s.tenantRepo.GetByCode(bctx, tenantCode)
	if err != nil {
		return nil, err
	}
	for i := range members {
		if members[i].TenantID == tenant.ID {
			return &members[i], nil
		}
	}
	return nil, httpx.Wrap(ErrNotMember, fmt.Errorf("account %s is not a member of tenant %s", account.Name, tenantCode))
}

// Register 注册：创建账号 + 默认租户成员（保持单租户默认体验，ADR-006）
func (s *accountService) Register(ctx context.Context, account *model.Account) (*model.Account, *model.User, error) {
	if err := s.Validate(account); err != nil {
		return nil, nil, err
	}

	password, err := bcrypt.GenerateFromPassword([]byte(account.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil, err
	}
	account.Password = string(password)

	// 账号 + 默认成员同事务（FIX-020 约定）：否则成员创建半程失败会留下
	// 「有账号无身份」的孤儿账号，登录（无成员）与重试注册（名字唯一键
	// 冲突）双双失败，用户被永久卡死
	var member *model.User
	if err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		created, err := s.accountRepo.Create(tctx, account)
		if err != nil {
			return err
		}
		account = created

		member, err = s.createDefaultMember(tctx, account)
		return err
	}); err != nil {
		return nil, nil, err
	}

	return account, member, nil
}

// RegisterByPhone 短信免密注册（注册向导第 1 步「手机号+验证码」）：
// 验证码已由调用方经 sms 域校验（scene=register，即手机号持有证明）。
// 手机号已注册则直接解析登录成员返回 created=false——与短信登录等价，
// 注册向导后续步骤失败重试时不会因「账号已存在」卡死（幂等）；
// 未注册则服务端生成随机登录名与随机密码（用户不可知，仅兜底满足
// accounts.name/password 约束；PasswordInitialized=false 标记免密状态），
// 账号 + 默认成员同事务提交（与 Register 同口径，防半注册孤儿）
func (s *accountService) RegisterByPhone(ctx context.Context, phone string) (*model.Account, *model.User, bool, error) {
	if !accountPhonePattern.MatchString(phone) {
		return nil, nil, false, errors.New("invalid phone number")
	}

	if _, err := s.accountRepo.GetByPhone(ctx, phone); err == nil {
		// 已注册：等价短信登录，直接返回登录身份
		account, member, err := s.AuthByPhone(ctx, phone, "")
		return account, member, false, err
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, false, err
	}

	password, err := generateRandomSecret(16)
	if err != nil {
		return nil, nil, false, err
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil, false, err
	}

	// 显式取 false 指针：零值 bool 会被 GORM 的 default 标签省略、错走列默认 true
	uninitialized := false
	account := &model.Account{
		Phone:               phone,
		Nickname:            maskPhone(phone),
		Password:            string(hashed),
		PasswordInitialized: &uninitialized,
	}

	var member *model.User
	if err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		name, err := s.uniqueLoginName(tctx)
		if err != nil {
			return err
		}
		account.Name = name

		created, err := s.accountRepo.Create(tctx, account)
		if err != nil {
			return err
		}
		account = created

		member, err = s.createDefaultMember(tctx, account)
		return err
	}); err != nil {
		return nil, nil, false, err
	}

	return account, member, true, nil
}

// uniqueLoginName 生成未占用的随机登录名：u- 前缀 + 8 位 hex（4 字节
// crypto/rand）。仅满足 name 唯一约束的机器标识，用户后续可经资料入口
// 自定义；碰撞时查库重试，连续失败即上抛（事务回滚，重试注册安全）
func (s *accountService) uniqueLoginName(ctx context.Context) (string, error) {
	for i := 0; i < 3; i++ {
		name, err := generateRandomSecret(4)
		if err != nil {
			return "", err
		}
		candidate := fmt.Sprintf("u-%s", name)
		if _, err := s.accountRepo.GetByName(ctx, candidate); errors.Is(err, gorm.ErrRecordNotFound) {
			return candidate, nil
		} else if err != nil {
			return "", err
		}
	}
	return "", errors.New("failed to generate unique login name")
}

// maskPhone 手机号脱敏展示（如 138****1234）：免密注册账号的默认昵称
func maskPhone(phone string) string {
	if len(phone) != 11 {
		return phone
	}
	return phone[:3] + "****" + phone[7:]
}

// passwordInitialized 读侧判定：nil 视同 true——历史账号与其他未显式
// 落库的创建路径（OAuth 首登等）密码均为用户链路写入
func passwordInitialized(account *model.Account) bool {
	return account.PasswordInitialized == nil || *account.PasswordInitialized
}

// generateRandomSecret n 字节 crypto/rand 的 hex 串（2n 个字符）
func generateRandomSecret(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate random secret: %w", err)
	}
	return fmt.Sprintf("%x", buf), nil
}

// CreateOAuthAccount OAuth 登录链路：凭证已存在则复用账号并取默认成员；
// 首登则创建账号（含第三方凭证）+ 默认租户成员
func (s *accountService) CreateOAuthAccount(ctx context.Context, account *model.Account) (*model.Account, *model.User, error) {
	if len(account.AuthInfos) == 0 {
		return nil, nil, fmt.Errorf("empty auth info")
	}
	authInfo := account.AuthInfos[0]

	old, err := s.accountRepo.GetByAuthID(ctx, authInfo.AuthType, authInfo.AuthId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 与密码注册同口径：账号 + 默认成员同事务，防半注册孤儿
			var member *model.User
			if err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
				created, err := s.accountRepo.Create(tctx, account)
				if err != nil {
					return err
				}
				account = created

				member, err = s.createDefaultMember(tctx, account)
				return err
			}); err != nil {
				return nil, nil, err
			}
			return account, member, nil
		}
		return nil, nil, err
	}

	members, err := s.userRepo.ListByAccount(ctx, old.ID)
	if err != nil {
		return nil, nil, err
	}
	if len(members) == 0 {
		// 同 Auth 的孤儿自愈：OAuth 凭证已验证，补建默认成员
		member, err := s.createDefaultMember(ctx, old)
		if err != nil {
			return nil, nil, err
		}
		return old, member, nil
	}

	return old, &members[0], nil
}

func (s *accountService) Validate(account *model.Account) error {
	if account == nil {
		return errors.New("account is empty")
	}
	if account.Name == "" {
		return errors.New("account name is empty")
	}
	if len(account.Password) < MinPasswordLength {
		return fmt.Errorf("password length must great than %d", MinPasswordLength)
	}
	return nil
}

func (s *accountService) Default(account *model.Account) {
	if account == nil || account.Name == "" {
		return
	}
	if account.Email == "" {
		account.Email = fmt.Sprintf("%s@qinng.io", account.Name)
	}
	if account.Nickname == "" {
		account.Nickname = account.Name
	}
}

// createDefaultMember 注册/OAuth 首登后在默认租户建立成员关系；
// 显式指定 TenantID，避免依赖列默认值。建成员前执行 members 配额校验（FIX-011）
func (s *accountService) createDefaultMember(ctx context.Context, account *model.Account) (*model.User, error) {
	// 默认租户是全平台注册账号的共享落脚点（非客户租户），不受成员配额约束：
	// 免费版 5 人上限若在此生效，等于封死全平台新用户注册（QUOTA_EXCEEDED 500）。
	// 真实租户的配额校验走开通（tenantService.Open）/加入链路
	nickname := account.Nickname
	if nickname == "" {
		nickname = account.Name
	}
	member := &model.User{
		AccountId: account.ID,
		Nickname:  nickname,
	}
	member.TenantID = tenantmodel.DefaultTenantID
	return s.userRepo.Create(ctx, member)
}

// TenantMembership 账号的租户成员关系（对齐简道云 owned_tenant_list 形态）
type TenantMembership struct {
	TenantID uint   `json:"tenantId"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	MemberID uint   `json:"memberId"`
	IsOwner  bool   `json:"isOwner"`
}

// ListTenants 账号的全部成员关系及租户概要（含 owner 标记）。
// 账号级跨租户查询：剥离请求携带的租户上下文，否则租户 Callback
// 追加 "tenant_id = 当前租户" 会漏掉其他租户的成员关系
func (s *accountService) ListTenants(ctx context.Context, accountID uint) ([]TenantMembership, error) {
	bctx := contextx.DetachTenant(ctx)
	members, err := s.userRepo.ListByAccount(bctx, accountID)
	if err != nil {
		return nil, err
	}

	tenantIDs := make([]uint, 0, len(members))
	for _, m := range members {
		tenantIDs = append(tenantIDs, m.TenantID)
	}
	tenants, err := s.tenantRepo.GetByIDs(ctx, tenantIDs)
	if err != nil {
		return nil, err
	}
	tenantByID := make(map[uint]*tenantmodel.Tenant, len(tenants))
	for i := range tenants {
		tenantByID[tenants[i].ID] = &tenants[i]
	}

	result := make([]TenantMembership, 0, len(members))
	for _, m := range members {
		item := TenantMembership{TenantID: m.TenantID, MemberID: m.ID}
		if t, ok := tenantByID[m.TenantID]; ok {
			item.Code = t.Code
			item.Name = t.Name
			// FIX-016：Owner 为可空引用，NULL = 暂无 Owner
			item.IsOwner = t.OwnerAccountId != nil && *t.OwnerAccountId == accountID
		}
		result = append(result, item)
	}
	return result, nil
}

// SwitchTenant 切换租户：校验账号在该租户的成员关系，返回重签所需的账号+成员
func (s *accountService) SwitchTenant(ctx context.Context, accountID, tenantID uint) (*model.Account, *model.User, error) {
	// 目标租户≠当前会话租户：剥离租户上下文查询目标成员身份，
	// 否则租户 Callback 追加当前租户过滤导致跨租户切换恒失败
	bctx := contextx.DetachTenant(ctx)
	member, err := s.userRepo.GetByAccountAndTenant(bctx, accountID, tenantID)
	if err != nil {
		// 与既有口径一致：成员关系缺失/查询失败均拒绝切换，细节经 Wrap 只入日志
		return nil, nil, httpx.Wrap(ErrNotMember, err)
	}

	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, nil, err
	}

	return account, member, nil
}

// UserInfoResult 登录聚合信息（对齐简道云 get_login_user 引导形态）：
// 账号资料 + 当前成员身份 + 当前租户（配置/套餐/生效配额）
type UserInfoResult struct {
	Account         *model.Account      `json:"account"`
	Member          *model.User         `json:"member"`
	Tenant          *tenantmodel.Tenant `json:"tenant"`
	EffectiveQuotas map[string]int64    `json:"effectiveQuotas"`
}

// GetUserInfo 聚合账号资料、成员身份与租户配置/套餐（member 由调用方从会话提供）
func (s *accountService) GetUserInfo(ctx context.Context, accountID uint, member *model.User) (*UserInfoResult, error) {
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}

	tenant, err := s.tenantRepo.GetByID(ctx, member.TenantID)
	if err != nil {
		return nil, err
	}

	// 生效配额：覆盖值优先，缺键回落套餐默认（键集见 tenant model 常量）
	quotas := make(map[string]int64)
	for _, key := range []string{
		tenantmodel.QuotaApps,
		tenantmodel.QuotaForms,
		tenantmodel.QuotaMembers,
		tenantmodel.QuotaStorageGB,
		tenantmodel.QuotaWorkflowRunsMonth,
	} {
		quotas[key] = tenant.Quotas.Get(tenant.Plan, key, 0)
	}

	return &UserInfoResult{
		Account:         account,
		Member:          member,
		Tenant:          tenant,
		EffectiveQuotas: quotas,
	}, nil
}

// GetProfile 账号自助：取本人账号资料
func (s *accountService) GetProfile(ctx context.Context, accountID uint) (*model.Account, error) {
	return s.accountRepo.GetByID(ctx, accountID)
}

// UpdateProfile 账号自助：更新昵称/邮箱/头像/手机号与注册引导画像（onboarding）。
// 昵称非空时同步刷新当前成员（users）昵称——注册向导第 3 步「怎么称呼你」
// 语义是租户内称呼（ADR-006：账号是登录身份，成员是租户内身份），两表写走同一事务
func (s *accountService) UpdateProfile(ctx context.Context, account *model.Account) (*model.Account, error) {
	if account == nil || account.ID == 0 {
		return nil, fmt.Errorf("empty account")
	}

	err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		if _, err := s.accountRepo.Update(tctx, account); err != nil {
			return err
		}
		if account.Nickname == "" {
			return nil
		}
		actor, ok := contextx.ActorFromContext(tctx)
		if !ok || actor.MemberID == 0 {
			// 无成员上下文（如平台侧维护路径）：只改账号昵称，成员留待各自入口
			return nil
		}
		_, err := s.userRepo.Update(tctx, &model.User{ID: actor.MemberID, Nickname: account.Nickname})
		return err
	})
	if err != nil {
		return nil, err
	}
	return account, nil
}

// ChangePassword 账号自助：校验旧密码后重置。短信免密注册的账号
// （PasswordInitialized=false，密码为服务端随机值）首次设置免旧密码，
// 设置成功即置位，此后恢复常规旧密码校验
func (s *accountService) ChangePassword(ctx context.Context, accountID uint, oldPassword, newPassword string) error {
	if len(newPassword) < MinPasswordLength {
		return fmt.Errorf("password length must great than %d", MinPasswordLength)
	}

	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return err
	}
	if passwordInitialized(account) {
		if err := bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(oldPassword)); err != nil {
			return fmt.Errorf("old password mismatch")
		}
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.accountRepo.UpdatePassword(ctx, accountID, string(hashed), true)
}
