package service

import (
	"context"
	"testing"

	auditservice "evolyn/internal/platform/audit/service"
	iammodel "evolyn/internal/platform/iam/model"
	iamrepository "evolyn/internal/platform/iam/repository"
	tenantrepository "evolyn/internal/platform/tenant/repository"

	"github.com/stretchr/testify/assert"
)

type platformAccountTx struct{}

func (platformAccountTx) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type platformAccountRepo struct {
	iamrepository.AccountRepository
	account *iammodel.Account
	purged  bool
}

func (r *platformAccountRepo) GetByID(context.Context, uint) (*iammodel.Account, error) {
	return r.account, nil
}
func (r *platformAccountRepo) Purge(context.Context, uint) error {
	r.purged = true
	return nil
}

type platformUserRepo struct {
	iamrepository.UserRepository
	purged bool
}

func (r *platformUserRepo) PurgeByAccount(context.Context, uint) error {
	r.purged = true
	return nil
}

type platformTenantRepo struct {
	tenantrepository.TenantRepository
	owned int64
}

func (r *platformTenantRepo) CountOwnedByAccount(context.Context, uint) (int64, error) {
	return r.owned, nil
}

type platformAudit struct{ entries int }

func (a *platformAudit) Record(context.Context, auditservice.Entry) { a.entries++ }

func TestPlatformAccountDeleteRejectsTenantOwner(t *testing.T) {
	accounts := &platformAccountRepo{account: &iammodel.Account{ID: 9}}
	users := &platformUserRepo{}
	audit := &platformAudit{}
	svc := NewAccountDeletionService(platformAccountTx{}, accounts, users, &platformTenantRepo{owned: 1}, audit)

	err := svc.Delete(context.Background(), 9)
	assert.ErrorIs(t, err, ErrAccountOwnsTenant)
	assert.False(t, users.purged)
	assert.False(t, accounts.purged)
	assert.Zero(t, audit.entries)
}

func TestPlatformAccountDeletePurgesRelationsBeforeAccount(t *testing.T) {
	accounts := &platformAccountRepo{account: &iammodel.Account{ID: 9}}
	users := &platformUserRepo{}
	audit := &platformAudit{}
	svc := NewAccountDeletionService(platformAccountTx{}, accounts, users, &platformTenantRepo{}, audit)

	assert.NoError(t, svc.Delete(context.Background(), 9))
	assert.True(t, users.purged)
	assert.True(t, accounts.purged)
	assert.Equal(t, 1, audit.entries)
}
