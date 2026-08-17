package service

import "evolyn/internal/model"

type UserService interface {
	List() (model.Users, error)
	Create(*model.User) (*model.User, error)
	Get(string) (*model.User, error)
	CreateOAuthUser(user *model.User) (*model.User, error)
	Update(string, *model.User) (*model.User, error)
	Delete(string) error
	Validate(*model.User) error
	Auth(*model.AuthUser) (*model.User, error)
	Default(*model.User)
	GetGroups(string) ([]model.Group, error)
	AddRole(id, rid string) error
	DelRole(id, rid string) error
}

type GroupService interface {
	List() ([]model.Group, error)
	Create(*model.User, *model.Group) (*model.Group, error)
	Get(string) (*model.Group, error)
	Update(string, *model.Group) (*model.Group, error)
	Delete(string) error
	GetUsers(gid string) (model.Users, error)
	AddUser(user *model.User, gid string) error
	DelUser(user *model.User, gid string) error
	AddRole(id, rid string) error
	DelRole(id, rid string) error
}

type RBACService interface {
	List() ([]model.Role, error)
	Create(role *model.Role) (*model.Role, error)
	Get(id string) (*model.Role, error)
	Update(id string, role *model.Role) (*model.Role, error)
	Delete(id string) error
	ListResources() ([]model.Resource, error)
	ListOperations() ([]model.Operation, error)
}
