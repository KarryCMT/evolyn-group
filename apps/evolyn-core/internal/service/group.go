package service

import (
	"fmt"
	"strconv"

	"github.com/qingwave/weave/internal/model"
	"github.com/qingwave/weave/internal/repository"
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

func (g *groupService) List() ([]model.Group, error) {
	return g.groupRepository.List()
}

func (g *groupService) Create(user *model.User, group *model.Group) (*model.Group, error) {
	return g.groupRepository.Create(user, group)
}

func (g *groupService) Get(id string) (*model.Group, error) {
	gid, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}
	return g.groupRepository.GetGroupByID(uint(gid))
}

func (g *groupService) Update(id string, group *model.Group) (*model.Group, error) {
	gid, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}
	group.ID = uint(gid)
	return g.groupRepository.Update(group)
}

func (g *groupService) Delete(id string) error {
	gid, err := strconv.Atoi(id)
	if err != nil {
		return err
	}

	return g.groupRepository.Delete(uint(gid))
}

func (g *groupService) GetUsers(id string) (model.Users, error) {
	gid, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}

	return g.groupRepository.GetUsers(&model.Group{ID: uint(gid)})
}

func (g *groupService) AddUser(user *model.User, id string) error {
	var err error
	if user.ID == 0 {
		return fmt.Errorf("invaild user info")
	}

	gid, err := strconv.Atoi(id)
	if err != nil {
		return err
	}

	return g.groupRepository.AddUser(user, &model.Group{ID: uint(gid)})
}

func (g *groupService) DelUser(user *model.User, id string) error {
	var err error
	if user.ID == 0 {
		return fmt.Errorf("invaild user info")
	}

	gid, err := strconv.Atoi(id)
	if err != nil {
		return err
	}

	return g.groupRepository.DelUser(user, &model.Group{ID: uint(gid)})
}

func (g *groupService) AddRole(id, rid string) error {
	gid, err := strconv.Atoi(id)
	if err != nil {
		return err
	}

	roleId, err := strconv.Atoi(rid)
	if err != nil {
		return err
	}

	return g.groupRepository.AddRole(&model.Role{ID: uint(roleId)}, &model.Group{ID: uint(gid)})
}

func (g *groupService) DelRole(id, rid string) error {
	gid, err := strconv.Atoi(id)
	if err != nil {
		return err
	}

	roleId, err := strconv.Atoi(rid)
	if err != nil {
		return err
	}

	return g.groupRepository.DelRole(&model.Role{ID: uint(roleId)}, &model.Group{ID: uint(gid)})
}
