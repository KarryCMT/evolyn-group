package repository

import (
	"context"
	"fmt"

	"evolyn/internal/infrastructure"
	"evolyn/internal/platform/iam/model"

	"gorm.io/gorm"
)

type accountRepository struct {
	db  *gorm.DB
	rdb *infrastructure.RedisDB
}

func newAccountRepository(db *gorm.DB, rdb *infrastructure.RedisDB) AccountRepository {
	return &accountRepository{
		db:  db,
		rdb: rdb,
	}
}

// withContext 以请求 ctx 打开新会话；accounts 为平台级表（无 tenant_id 列），
// 租户 Callback 对其无副作用。ctx 携带事务 session 时加入外层事务
// （FIX-020：开通租户时新建 owner 账号须随全流程回滚）
func (a *accountRepository) withContext(ctx context.Context) *gorm.DB {
	return infrastructure.ResolveDB(ctx, a.db)
}

func (a *accountRepository) GetByID(ctx context.Context, id uint) (*model.Account, error) {
	account := new(model.Account)
	if err := a.withContext(ctx).Preload(model.UserAuthInfoAssociation).First(account, id).Error; err != nil {
		return nil, err
	}
	return account, nil
}

func (a *accountRepository) GetByName(ctx context.Context, name string) (*model.Account, error) {
	account := new(model.Account)
	if err := a.withContext(ctx).Preload(model.UserAuthInfoAssociation).Where("name = ?", name).First(account).Error; err != nil {
		return nil, err
	}
	return account, nil
}

func (a *accountRepository) GetByPhone(ctx context.Context, phone string) (*model.Account, error) {
	account := new(model.Account)
	if err := a.withContext(ctx).Preload(model.UserAuthInfoAssociation).Where("phone = ?", phone).First(account).Error; err != nil {
		return nil, err
	}
	return account, nil
}

// GetByAuthID 按第三方凭证定位账号（OAuth 登录链路；登录前无租户上下文）
func (a *accountRepository) GetByAuthID(ctx context.Context, authType, authID string) (*model.Account, error) {
	authInfo := new(model.AuthInfo)
	if err := a.withContext(ctx).Where("auth_type = ? and auth_id = ?", authType, authID).First(authInfo).Error; err != nil {
		return nil, err
	}

	return a.GetByID(ctx, authInfo.AccountId)
}

func (a *accountRepository) List(ctx context.Context) ([]model.Account, error) {
	accounts := make([]model.Account, 0)
	if err := a.withContext(ctx).Order("id").Find(&accounts).Error; err != nil {
		return nil, err
	}
	return accounts, nil
}

func (a *accountRepository) Create(ctx context.Context, account *model.Account) (*model.Account, error) {
	if err := a.withContext(ctx).Create(account).Error; err != nil {
		return nil, err
	}
	return account, nil
}

// Update 更新账号资料（昵称/邮箱/头像/注册画像等），不含密码（密码走专用重置链路）。
// 部分更新语义：只写非空字段——显式 Select 会让 GORM 强制落库零值，
// 曾把请求未携带的 phone/email 清空（验证码登录随之 account not found）
func (a *accountRepository) Update(ctx context.Context, account *model.Account) (*model.Account, error) {
	cols := make([]string, 0, 5)
	if account.Nickname != "" {
		cols = append(cols, "nickname")
	}
	if account.Phone != "" {
		cols = append(cols, "phone")
	}
	if account.Email != "" {
		cols = append(cols, "email")
	}
	if account.Avatar != "" {
		cols = append(cols, "avatar")
	}
	if account.Onboarding != (model.AccountOnboarding{}) {
		cols = append(cols, "onboarding")
	}
	if len(cols) == 0 {
		return account, nil
	}
	if err := a.withContext(ctx).Model(&model.Account{}).Where("id = ?", account.ID).
		Select(cols).Updates(account).Error; err != nil {
		return nil, err
	}
	return account, nil
}

// UpdatePassword 直接写散列后的密码，并同步落「密码是否由用户设置」标记。
// session_version 在同一条 UPDATE 内递增，避免密码更新成功但旧 JWT 未失效。
// 服务层负责 bcrypt 与旧密码校验；首设成功由服务层传 true。
func (a *accountRepository) UpdatePassword(ctx context.Context, id uint, hashed string, initialized bool) error {
	return a.withContext(ctx).Model(&model.Account{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"password":             hashed,
			"password_initialized": initialized,
			"session_version":      gorm.Expr("session_version + 1"),
		}).Error
}

// Purge 只由平台账号删除流程在外层事务内调用。成员关系先由 UserRepository
// 清理；pf_auth_infos 不设级联外键需显式删除，账号安全表由数据库 CASCADE 清理。
func (a *accountRepository) Purge(ctx context.Context, accountID uint) error {
	db := a.withContext(ctx)
	if err := db.Where("account_id = ?", accountID).Delete(&model.AuthInfo{}).Error; err != nil {
		return err
	}
	return db.Unscoped().Delete(&model.Account{}, accountID).Error
}

func (a *accountRepository) AddAuthInfo(ctx context.Context, authInfo *model.AuthInfo) error {
	if authInfo == nil {
		return nil
	}
	if authInfo.AccountId == 0 {
		return fmt.Errorf("empty account id")
	}
	return a.withContext(ctx).Create(authInfo).Error
}

func (a *accountRepository) DelAuthInfo(ctx context.Context, authInfo *model.AuthInfo) error {
	if authInfo == nil {
		return nil
	}
	return a.withContext(ctx).Delete(authInfo).Error
}

func (a *accountRepository) Migrate() error {
	return a.db.AutoMigrate(&model.Account{}, &model.AuthInfo{})
}
