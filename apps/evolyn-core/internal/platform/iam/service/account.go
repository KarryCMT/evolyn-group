package service

import (
	"context"
	"errors"
	"fmt"

	"evolyn/internal/contextx"
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
		// 账号不存在与密码错误统一文案：既不泄露 gorm 内部错误，
		// 也不区分「账号是否存在」
		return nil, nil, fmt.Errorf("invalid account or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(auser.Password)); err != nil {
		return nil, nil, fmt.Errorf("invalid account or password")
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
		return nil, nil, fmt.Errorf("account not found")
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
	return nil, fmt.Errorf("account %s is not a member of tenant %s", account.Name, tenantCode)
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
		return nil, nil, fmt.Errorf("account is not a member of tenant %d", tenantID)
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

// ChangePassword 账号自助：校验旧密码后重置
func (s *accountService) ChangePassword(ctx context.Context, accountID uint, oldPassword, newPassword string) error {
	if len(newPassword) < MinPasswordLength {
		return fmt.Errorf("password length must great than %d", MinPasswordLength)
	}

	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(oldPassword)); err != nil {
		return fmt.Errorf("old password mismatch")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.accountRepo.UpdatePassword(ctx, accountID, string(hashed))
}
