package service

import (
	"context"
	"testing"

	"evolyn/internal/platform/iam/model"
	"evolyn/internal/platform/iam/repository"
	tenantmodel "evolyn/internal/platform/tenant/model"
	tenantrepository "evolyn/internal/platform/tenant/repository"
)

// userInfoAccountRepoStub 仅覆盖登录聚合读取路径，其余仓储方法由嵌入接口承接。
type userInfoAccountRepoStub struct {
	repository.AccountRepository
	account *model.Account
}

func (r userInfoAccountRepoStub) GetByID(_ context.Context, _ uint) (*model.Account, error) {
	return r.account, nil
}

type userInfoMemberRepoStub struct {
	repository.UserRepository
	member    *model.User
	requested uint
}

func (r *userInfoMemberRepoStub) GetMemberDetail(_ context.Context, id uint) (*model.User, error) {
	r.requested = id
	return r.member, nil
}

type userInfoTenantRepoStub struct {
	tenantrepository.TenantRepository
	tenant *tenantmodel.Tenant
}

func (r userInfoTenantRepoStub) GetByID(_ context.Context, _ uint) (*tenantmodel.Tenant, error) {
	return r.tenant, nil
}

func TestGetUserInfoLoadsCurrentMemberDepartments(t *testing.T) {
	currentMember := &model.User{
		ID: 12,
		Departments: []model.Department{
			{ID: 3, Name: "产品部"},
		},
	}
	currentMember.TenantID = 7
	memberRepo := &userInfoMemberRepoStub{member: currentMember}
	service := &accountService{
		accountRepo: userInfoAccountRepoStub{account: &model.Account{ID: 9}},
		userRepo:    memberRepo,
		tenantRepo:  userInfoTenantRepoStub{tenant: &tenantmodel.Tenant{ID: 7, Plan: tenantmodel.PlanFree}},
	}

	memberSnapshot := &model.User{ID: 12}
	memberSnapshot.TenantID = 7
	info, err := service.GetUserInfo(context.Background(), 9, memberSnapshot)
	if err != nil {
		t.Fatalf("GetUserInfo() error = %v", err)
	}
	if memberRepo.requested != 12 {
		t.Fatalf("GetMemberDetail() requested member = %d, want 12", memberRepo.requested)
	}
	if got := info.Member.Departments; len(got) != 1 || got[0].ID != 3 || got[0].Name != "产品部" {
		t.Fatalf("userinfo departments = %#v, want current member department", got)
	}
}
