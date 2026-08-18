package service

import (
	"context"
	"fmt"
	"strconv"

	"evolyn/internal/platform/iam/model"
	"evolyn/internal/platform/iam/repository"
)

// userService 成员服务（租户内语义）：登录身份相关见 AccountService（ADR-006）
type userService struct {
	userRepository repository.UserRepository
}

func NewUserService(userRepository repository.UserRepository) UserService {
	return &userService{
		userRepository: userRepository,
	}
}

func (u *userService) List(ctx context.Context) (model.Users, error) {
	return u.userRepository.List(ctx)
}

func (u *userService) Get(ctx context.Context, id string) (*model.User, error) {
	return u.getUserByID(ctx, id)
}

// Update 成员资料更新（租户内昵称）；账号资料走 accounts 域
func (u *userService) Update(ctx context.Context, id string, member *model.User) (*model.User, error) {
	old, err := u.getUserByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if member.ID != 0 && old.ID != member.ID {
		return nil, fmt.Errorf("update member %s not match", id)
	}
	member.ID = old.ID

	return u.userRepository.Update(ctx, member)
}

func (u *userService) Delete(ctx context.Context, id string) error {
	uid, err := strconv.Atoi(id)
	if err != nil {
		return err
	}
	return u.userRepository.Delete(ctx, &model.User{ID: uint(uid)})
}

func (u *userService) GetGroups(ctx context.Context, id string) ([]model.Group, error) {
	member, err := u.getUserByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return u.userRepository.GetGroups(ctx, member)
}

func (u *userService) AddRole(ctx context.Context, id, rid string) error {
	uid, err := strconv.Atoi(id)
	if err != nil {
		return err
	}

	roleId, err := strconv.Atoi(rid)
	if err != nil {
		return err
	}

	return u.userRepository.AddRole(ctx, &model.Role{ID: uint(roleId)}, &model.User{ID: uint(uid)})
}

func (u *userService) DelRole(ctx context.Context, id, rid string) error {
	uid, err := strconv.Atoi(id)
	if err != nil {
		return err
	}

	roleId, err := strconv.Atoi(rid)
	if err != nil {
		return err
	}

	return u.userRepository.DelRole(ctx, &model.Role{ID: uint(roleId)}, &model.User{ID: uint(uid)})
}

func (u *userService) getUserByID(ctx context.Context, id string) (*model.User, error) {
	uid, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}
	return u.userRepository.GetUserByID(ctx, uint(uid))
}
