package service

import (
	"context"
	"fmt"
	"net/http"

	auditservice "evolyn/internal/platform/audit/service"
	"evolyn/internal/platform/httpx"
	"evolyn/internal/platform/iam/repository"
	tenantrepository "evolyn/internal/platform/tenant/repository"
)

// AccountDeletionService 供账号自助与平台运营域共用的账号物理删除能力。账号
// 仍是任一租户创建人时拒绝删除，强制先完成创建人转移或注销租户，避免遗留无主租户。
type AccountDeletionService interface {
	Delete(ctx context.Context, accountID uint) error
}

type platformAccountService struct {
	tx       TxManager
	accounts repository.AccountRepository
	users    repository.UserRepository
	tenants  tenantrepository.TenantRepository
	audit    auditservice.Recorder
}

func NewAccountDeletionService(
	tx TxManager,
	accounts repository.AccountRepository,
	users repository.UserRepository,
	tenants tenantrepository.TenantRepository,
	audit auditservice.Recorder,
) AccountDeletionService {
	return &platformAccountService{tx: tx, accounts: accounts, users: users, tenants: tenants, audit: audit}
}

var ErrAccountOwnsTenant = httpx.NewBiz("ACCOUNT_OWNS_TENANT", "账号仍是租户创建人，请先转移创建人或注销租户", http.StatusConflict)

// Delete 在单一事务内删除账号的全部成员身份、第三方凭证与账号本体。账号安全
// 表通过外键级联清理；登录与审计日志作为历史流水保留，不依赖账号外键。
func (s *platformAccountService) Delete(ctx context.Context, accountID uint) error {
	if accountID == 0 {
		return httpx.NewBiz(httpx.CodeValidation, "账号不能为空", http.StatusBadRequest)
	}

	err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		if _, err := s.accounts.GetByID(tctx, accountID); err != nil {
			return err
		}
		owned, err := s.tenants.CountOwnedByAccount(tctx, accountID)
		if err != nil {
			return err
		}
		if owned > 0 {
			return ErrAccountOwnsTenant
		}
		if err := s.users.PurgeByAccount(tctx, accountID); err != nil {
			return err
		}
		return s.accounts.Purge(tctx, accountID)
	})
	if err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "iam", Action: "delete", ResourceType: "account",
			ResourceID: fmt.Sprint(accountID), AccountID: accountID,
		})
	}
	return nil
}
