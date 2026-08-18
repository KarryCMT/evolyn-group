package service

import (
	"context"
	"fmt"
	"strconv"

	"evolyn/internal/platform/iam/model"
	"evolyn/internal/platform/iam/repository"
)

type groupService struct {
	userRepository  repository.UserRepository
	groupRepository repository.GroupRepository
	rbacRepository  repository.RBACRepository
}

func NewGroupService(groupRepository repository.GroupRepository, userRepository repository.UserRepository) GroupService {
	return &groupService{
		groupRepository: groupRepository,
		userRepository:  userRepository,
	}
}

func (g *groupService) List(ctx context.Context) ([]model.Group, error) {
	return g.groupRepository.List(ctx)
}

func (g *groupService) Create(ctx context.Context, user *model.User, group *model.Group) (*model.Group, error) {
	return g.groupRepository.Create(ctx, user, group)
}

func (g *groupService) Get(ctx context.Context, id string) (*model.Group, error) {
	gid, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}
	return g.groupRepository.GetGroupByID(ctx, uint(gid))
}

func (g *groupService) Update(ctx context.Context, id string, group *model.Group) (*model.Group, error) {
	gid, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}
	group.ID = uint(gid)
	return g.groupRepository.Update(ctx, group)
}

func (g *groupService) Delete(ctx context.Context, id string) error {
	gid, err := strconv.Atoi(id)
	if err != nil {
		return err
	}

	return g.groupRepository.Delete(ctx, uint(gid))
}

func (g *groupService) GetUsers(ctx context.Context, id string) (model.Users, error) {
	gid, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}

	return g.groupRepository.GetUsers(ctx, &model.Group{ID: uint(gid)})
}

func (g *groupService) AddUser(ctx context.Context, user *model.User, id string) error {
	var err error
	if user.ID == 0 {
		return fmt.Errorf("invaild user info")
	}

	gid, err := strconv.Atoi(id)
	if err != nil {
		return err
	}

	return g.groupRepository.AddUser(ctx, user, &model.Group{ID: uint(gid)})
}

func (g *groupService) DelUser(ctx context.Context, user *model.User, id string) error {
	var err error
	if user.ID == 0 {
		return fmt.Errorf("invaild user info")
	}

	gid, err := strconv.Atoi(id)
	if err != nil {
		return err
	}

	return g.groupRepository.DelUser(ctx, user, &model.Group{ID: uint(gid)})
}

func (g *groupService) AddRole(ctx context.Context, id, rid string) error {
	gid, err := strconv.Atoi(id)
	if err != nil {
		return err
	}

	roleId, err := strconv.Atoi(rid)
	if err != nil {
		return err
	}

	return g.groupRepository.AddRole(ctx, &model.Role{ID: uint(roleId)}, &model.Group{ID: uint(gid)})
}

func (g *groupService) DelRole(ctx context.Context, id, rid string) error {
	gid, err := strconv.Atoi(id)
	if err != nil {
		return err
	}

	roleId, err := strconv.Atoi(rid)
	if err != nil {
		return err
	}

	return g.groupRepository.DelRole(ctx, &model.Role{ID: uint(roleId)}, &model.Group{ID: uint(gid)})
}
