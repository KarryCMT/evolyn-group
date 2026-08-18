package service

import (
	"context"
	"strconv"

	"evolyn/internal/platform/iam/model"
	"evolyn/internal/platform/iam/repository"
	"evolyn/pkg/utils/request"
)

type rbacService struct {
	rbacRepository repository.RBACRepository
}

func NewRBACService(rbacRepository repository.RBACRepository) RBACService {
	return &rbacService{
		rbacRepository: rbacRepository,
	}
}

func (rbac *rbacService) List(ctx context.Context) ([]model.Role, error) {
	return rbac.rbacRepository.List(ctx)
}

func (rbac *rbacService) Create(ctx context.Context, role *model.Role) (*model.Role, error) {
	return rbac.rbacRepository.Create(ctx, role)
}

func (rbac *rbacService) Get(ctx context.Context, id string) (*model.Role, error) {
	rid, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}
	return rbac.rbacRepository.GetRoleByID(ctx, rid)
}

func (rbac *rbacService) Update(ctx context.Context, id string, role *model.Role) (*model.Role, error) {
	rid, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}
	role.ID = uint(rid)
	return rbac.rbacRepository.Update(ctx, role)
}

func (rbac *rbacService) Delete(ctx context.Context, id string) error {
	rid, err := strconv.Atoi(id)
	if err != nil {
		return err
	}

	return rbac.rbacRepository.Delete(ctx, uint(rid))
}

func (rbac *rbacService) ListResources(ctx context.Context) ([]model.Resource, error) {
	return rbac.rbacRepository.ListResources(ctx)
}

func (rbac *rbacService) ListOperations() ([]model.Operation, error) {
	return []model.Operation{
		model.AllOperation,
		model.EditOperation,
		model.ViewOperation,
		request.CreateOperation,
		request.PatchOperation,
		request.UpdateOperation,
		request.GetOperation,
		request.ListOperation,
		request.DeleteOperation,
		"log",
		"exec",
		"proxy",
	}, nil
}
