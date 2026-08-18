package service

import (
	"context"
	"errors"
	"fmt"

	"evolyn/internal/platform/iam/model"
	"evolyn/internal/platform/iam/repository"
	tenantmodel "evolyn/internal/platform/tenant/model"
	tenantrepository "evolyn/internal/platform/tenant/repository"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	MinPasswordLength = 6
)

type accountService struct {
	accountRepo repository.AccountRepository
	userRepo    repository.UserRepository
	tenantRepo  tenantrepository.TenantRepository
}

// NewAccountService 账号服务：登录身份校验与账号生命周期；
// 依赖成员仓储（注册即建默认租户成员）与租户仓储（登录指定 TenantCode 时解析）
func NewAccountService(accountRepo repository.AccountRepository, userRepo repository.UserRepository, tenantRepo tenantrepository.TenantRepository) AccountService {
	return &accountService{
		accountRepo: accountRepo,
		userRepo:    userRepo,
		tenantRepo:  tenantRepo,
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
		return nil, nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(auser.Password)); err != nil {
		return nil, nil, err
	}

	members, err := s.userRepo.ListByAccount(ctx, account.ID)
	if err != nil {
		return nil, nil, err
	}
	if len(members) == 0 {
		return nil, nil, fmt.Errorf("account %s has no tenant membership", account.Name)
	}

	member := &members[0]
	if auser.TenantCode != "" {
		tenant, err := s.tenantRepo.GetByCode(ctx, auser.TenantCode)
		if err != nil {
			return nil, nil, err
		}
		member = nil
		for i := range members {
			if members[i].TenantID == tenant.ID {
				member = &members[i]
				break
			}
		}
		if member == nil {
			return nil, nil, fmt.Errorf("account %s is not a member of tenant %s", account.Name, auser.TenantCode)
		}
	}

	return account, member, nil
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

	account, err = s.accountRepo.Create(ctx, account)
	if err != nil {
		return nil, nil, err
	}

	member, err := s.createDefaultMember(ctx, account)
	if err != nil {
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
			account, err = s.accountRepo.Create(ctx, account)
			if err != nil {
				return nil, nil, err
			}
			member, err := s.createDefaultMember(ctx, account)
			if err != nil {
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
		return nil, nil, fmt.Errorf("account %s has no tenant membership", old.Name)
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
// 显式指定 TenantID，避免依赖列默认值
func (s *accountService) createDefaultMember(ctx context.Context, account *model.Account) (*model.User, error) {
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
