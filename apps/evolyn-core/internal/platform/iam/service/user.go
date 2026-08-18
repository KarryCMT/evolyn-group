package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"evolyn/internal/platform/iam/model"
	"evolyn/internal/platform/iam/repository"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	MinPasswordLength = 6
)

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

func (u *userService) Create(ctx context.Context, user *model.User) (*model.User, error) {
	password, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user.Password = string(password)
	return u.userRepository.Create(ctx, user)
}

func (u *userService) Get(ctx context.Context, id string) (*model.User, error) {
	return u.getUserByID(ctx, id)
}

func (u *userService) Update(ctx context.Context, id string, new *model.User) (*model.User, error) {
	old, err := u.getUserByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if new.ID != 0 && old.ID != new.ID {
		return nil, fmt.Errorf("update user %s not match", id)
	}
	new.ID = old.ID

	if len(new.Password) > 0 {
		password, err := bcrypt.GenerateFromPassword([]byte(new.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		new.Password = string(password)
	}

	return u.userRepository.Update(ctx, new)
}

func (u *userService) Delete(ctx context.Context, id string) error {
	user, err := u.getUser(id)
	if err != nil {
		return err
	}
	return u.userRepository.Delete(ctx, user)
}

func (u *userService) Validate(user *model.User) error {
	if user == nil {
		return errors.New("user is empty")
	}
	if user.Name == "" {
		return errors.New("user name is empty")
	}
	if len(user.Password) < MinPasswordLength {
		return fmt.Errorf("password length must great than %d", MinPasswordLength)
	}
	return nil
}

func (u *userService) Default(user *model.User) {
	if user == nil || user.Name == "" {
		return
	}
	if user.Email == "" {
		user.Email = fmt.Sprintf("%s@qinng.io", user.Name)
	}
}

// Auth 登录链路在认证前无租户上下文，按用户名全局查找；
// 多租户登录入口（租户识别）随 P1 IAM 改造补充
func (u *userService) Auth(ctx context.Context, auser *model.AuthUser) (*model.User, error) {
	if auser == nil || auser.Name == "" || auser.Password == "" {
		return nil, fmt.Errorf("name or password is empty")
	}

	user, err := u.userRepository.GetUserByName(ctx, auser.Name)
	if err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(auser.Password)); err != nil {
		return nil, err
	}

	user.Password = ""

	return user, nil
}

func (u *userService) CreateOAuthUser(ctx context.Context, user *model.User) (*model.User, error) {
	if len(user.AuthInfos) == 0 {
		return nil, fmt.Errorf("empty auth info")
	}
	authInfo := user.AuthInfos[0]
	old, err := u.userRepository.GetUserByAuthID(ctx, authInfo.AuthType, authInfo.AuthId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return u.userRepository.Create(ctx, user)
		}
		return nil, err
	}
	return old, nil
}

func (u *userService) GetGroups(ctx context.Context, id string) ([]model.Group, error) {
	user, err := u.getUserByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return u.userRepository.GetGroups(ctx, user)
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

func (u *userService) getUser(id string) (*model.User, error) {
	uid, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}
	return &model.User{ID: uint(uid)}, nil
}
