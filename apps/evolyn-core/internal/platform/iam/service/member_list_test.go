package service

import (
	"context"
	"errors"
	"testing"

	"evolyn/internal/contextx"
	kernel "evolyn/internal/model"
	"evolyn/internal/platform/iam/model"
	"evolyn/internal/platform/iam/repository"
	tenantmodel "evolyn/internal/platform/tenant/model"

	"github.com/stretchr/testify/assert"
)

// memberListRepo 是成员列表服务的最小仓储桩；未覆写方法由嵌入接口兜底，
// 测试只覆盖查询参数规范化、读模型组装与状态流转。
type memberListRepo struct {
	repository.UserRepository
	users       model.Users
	total       int64
	listQuery   model.MemberListQuery
	updatedUser *model.User
}

func (r *memberListRepo) ListPage(_ context.Context, query model.MemberListQuery) (model.Users, int64, error) {
	r.listQuery = query
	return r.users, r.total, nil
}

func (r *memberListRepo) GetUserByID(_ context.Context, id uint) (*model.User, error) {
	for i := range r.users {
		if r.users[i].ID == id {
			return &r.users[i], nil
		}
	}
	return nil, errors.New("not found")
}

func (r *memberListRepo) UpdateStatus(_ context.Context, user *model.User) (*model.User, error) {
	r.updatedUser = user
	return user, nil
}

// memberTenantOwnerReaderStub 仅模拟成员状态变更所需的租户创建人读取。
type memberTenantOwnerReaderStub struct {
	ownerAccountID *uint
}

func (r memberTenantOwnerReaderStub) GetByID(_ context.Context, id uint) (*tenantmodel.Tenant, error) {
	return &tenantmodel.Tenant{ID: id, OwnerAccountId: r.ownerAccountID}, nil
}

func TestMemberListPageNormalizesQueryAndBuildsReadModel(t *testing.T) {
	repo := &memberListRepo{
		users: model.Users{
			{
				ID:        8,
				AccountId: 18,
				Nickname:  "租户昵称",
				Status:    model.MemberStatusActive,
				Account:   &model.Account{Nickname: "账号昵称", Phone: "13800000000", Email: "member@example.com"},
				Departments: []model.Department{
					{ID: 2, Name: "研发部"},
				},
				Roles: []model.Role{{ID: 3, Name: "租户管理员"}},
			},
		},
		total: 1,
	}
	svc := NewUserService(nil, repo, nil, nil, nil, nil, nil)

	page, err := svc.ListPage(context.Background(), model.MemberListQuery{Keyword: " 研发 ", PageSize: 500})

	assert.NoError(t, err)
	assert.Equal(t, 1, repo.listQuery.Page)
	assert.Equal(t, 100, repo.listQuery.PageSize)
	assert.Equal(t, "研发", repo.listQuery.Keyword)
	assert.Len(t, page.Items, 1)
	assert.Equal(t, "租户昵称", page.Items[0].Name)
	assert.Equal(t, "13800000000", page.Items[0].Phone)
	assert.Equal(t, []model.MemberDepartment{{ID: 2, Name: "研发部"}}, page.Items[0].Departments)
	assert.Equal(t, []model.MemberRole{{ID: 3, Name: "租户管理员"}}, page.Items[0].Roles)
}

func TestMemberListPageRejectsUnknownStatus(t *testing.T) {
	svc := NewUserService(nil, &memberListRepo{}, nil, nil, nil, nil, nil)

	_, err := svc.ListPage(context.Background(), model.MemberListQuery{Status: "unknown"})

	assert.ErrorIs(t, err, ErrMemberStatusInvalid)
}

func TestUpdateMemberStatusKeepsResignationHistory(t *testing.T) {
	repo := &memberListRepo{users: model.Users{{
		ID: 8, Status: model.MemberStatusActive,
		TenantBaseModel: kernel.TenantBaseModel{TenantID: 1},
	}}}
	svc := NewUserService(nil, repo, nil, nil, nil, nil, nil)

	resigned, err := svc.UpdateStatus(context.Background(), "8", model.MemberStatusResigned)

	assert.NoError(t, err)
	assert.Equal(t, model.MemberStatusResigned, resigned.Status)
	assert.NotNil(t, resigned.ResignedAt)
	assert.Same(t, resigned, repo.updatedUser)

	active, err := svc.UpdateStatus(context.Background(), "8", model.MemberStatusActive)
	assert.NoError(t, err)
	assert.Equal(t, model.MemberStatusActive, active.Status)
	assert.Nil(t, active.ResignedAt)
}

func TestUpdateMemberStatusRejectsTenantCreatorResignation(t *testing.T) {
	ownerAccountID := uint(18)
	repo := &memberListRepo{users: model.Users{{
		ID:              8,
		AccountId:       ownerAccountID,
		Status:          model.MemberStatusActive,
		TenantBaseModel: kernel.TenantBaseModel{TenantID: 1},
	}}}
	svc := NewUserService(
		nil,
		repo,
		nil,
		nil,
		nil,
		nil,
		nil,
		memberTenantOwnerReaderStub{ownerAccountID: &ownerAccountID},
	)

	_, err := svc.UpdateStatus(
		contextx.NewTenantContext(context.Background(), 1),
		"8",
		model.MemberStatusResigned,
	)

	assert.ErrorIs(t, err, ErrTenantCreatorStatusImmutable)
	assert.Nil(t, repo.updatedUser)
	assert.Equal(t, model.MemberStatusActive, repo.users[0].Status)
}
