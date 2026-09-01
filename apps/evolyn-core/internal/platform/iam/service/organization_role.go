package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	auditservice "evolyn/internal/platform/audit/service"
	"evolyn/internal/platform/iam/model"
	"evolyn/internal/platform/iam/repository"

	"gorm.io/gorm"
)

// OrganizationRoleTree 是内部组织页“角色”页签的完整树形读模型。
type OrganizationRoleTree struct {
	Groups []OrganizationRoleGroup `json:"groups"`
}

type OrganizationRoleGroup struct {
	ID    uint               `json:"id"`
	Name  string             `json:"name"`
	Roles []OrganizationRole `json:"roles"`
}

type OrganizationRole struct {
	ID      uint   `json:"id"`
	Name    string `json:"name"`
	GroupID uint   `json:"groupId"`
}

type CreateOrganizationRoleGroupRequest struct {
	Name string `json:"name"`
}

type CreateOrganizationRoleRequest struct {
	Name    string `json:"name"`
	GroupID uint   `json:"groupId"`
}

type OrganizationRoleMemberRequest struct {
	MemberIDs []uint `json:"memberIds"`
}

// ReplaceMemberRolesRequest 表达成员的完整直接角色集合；空数组表示解除全部直接角色。
// 角色组只用于展示，不是可绑定的角色主体。
type ReplaceMemberRolesRequest struct {
	RoleIDs []uint `json:"roleIds"`
}

type ReorderOrganizationRoleGroupRequest struct {
	GroupIDs []uint `json:"groupIds"`
}

type ReorderOrganizationRoleRequest struct {
	RoleIDs []uint `json:"roleIds"`
}

// OrganizationRoleService 收敛内部组织页的角色树、角色成员及角色分组写操作。
// 角色展示分组独立于权限分组 Group，二者不得混用。
type OrganizationRoleService interface {
	Tree(ctx context.Context) (*OrganizationRoleTree, error)
	CreateGroup(ctx context.Context, creatorID uint, req CreateOrganizationRoleGroupRequest) (*model.RoleGroup, error)
	RenameGroup(ctx context.Context, groupID, name string) (*model.RoleGroup, error)
	DeleteGroup(ctx context.Context, groupID string) error
	ReorderGroups(ctx context.Context, groupIDs []uint) error
	CreateRole(ctx context.Context, req CreateOrganizationRoleRequest) (*model.Role, error)
	RenameRole(ctx context.Context, roleID, name string) (*model.Role, error)
	MoveRole(ctx context.Context, roleID, groupID string) (*model.Role, error)
	ReorderRoles(ctx context.Context, groupID string, roleIDs []uint) error
	ListMembers(ctx context.Context, roleID string, query model.MemberListQuery) (*model.MemberPage, error)
	AddMembers(ctx context.Context, roleID string, memberIDs []uint) error
	RemoveMember(ctx context.Context, roleID, memberID string) error
	// ReplaceMemberRoles 原子替换成员的全部直接角色，用于成员详情中的角色列表。
	ReplaceMemberRoles(ctx context.Context, memberID string, roleIDs []uint) error
}

type organizationRoleService struct {
	tx         TxManager
	roles      repository.RBACRepository
	roleGroups repository.RoleGroupRepository
	users      repository.UserRepository
	userSvc    UserService
	audit      auditservice.Recorder
}

func NewOrganizationRoleService(tx TxManager, roles repository.RBACRepository, roleGroups repository.RoleGroupRepository, users repository.UserRepository, userSvc UserService, audit auditservice.Recorder) OrganizationRoleService {
	return &organizationRoleService{tx: tx, roles: roles, roleGroups: roleGroups, users: users, userSvc: userSvc, audit: audit}
}

func (s *organizationRoleService) Tree(ctx context.Context) (*OrganizationRoleTree, error) {
	groups, err := s.roleGroups.List(ctx)
	if err != nil {
		return nil, err
	}
	defaultGroup, err := s.defaultGroup(ctx, groups)
	if err != nil {
		return nil, err
	}
	hasDefaultGroup := false
	for i := range groups {
		if groups[i].ID == defaultGroup.ID {
			hasDefaultGroup = true
			break
		}
	}
	if !hasDefaultGroup {
		groups = append([]model.RoleGroup{*defaultGroup}, groups...)
	}
	roles, err := s.roles.List(ctx)
	if err != nil {
		return nil, err
	}
	tree := &OrganizationRoleTree{Groups: make([]OrganizationRoleGroup, 0, len(groups))}
	byID := make(map[uint]int, len(groups))
	for _, group := range groups {
		byID[group.ID] = len(tree.Groups)
		tree.Groups = append(tree.Groups, OrganizationRoleGroup{ID: group.ID, Name: group.Name, Roles: make([]OrganizationRole, 0)})
	}
	for _, role := range roles {
		groupID := defaultGroup.ID
		if role.RoleGroupID != nil {
			groupID = *role.RoleGroupID
		}
		if index, ok := byID[groupID]; ok {
			tree.Groups[index].Roles = append(tree.Groups[index].Roles, OrganizationRole{ID: role.ID, Name: role.Name, GroupID: groupID})
		}
	}
	return tree, nil
}

func (s *organizationRoleService) CreateGroup(ctx context.Context, creatorID uint, req CreateOrganizationRoleGroupRequest) (*model.RoleGroup, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, ErrOrganizationRoleGroupNameInvalid
	}
	if _, err := s.roleGroups.GetByName(ctx, name); err == nil {
		return nil, ErrDuplicateName
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	groups, err := s.roleGroups.List(ctx)
	if err != nil {
		return nil, err
	}
	group, err := s.roleGroups.Create(ctx, &model.RoleGroup{Name: name, CreatorMemberID: creatorID, Sort: len(groups)})
	if err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{Module: "iam", Action: "create", ResourceType: "role_group", ResourceID: strconv.FormatUint(uint64(group.ID), 10), After: map[string]string{"name": group.Name}})
	}
	return group, nil
}

func (s *organizationRoleService) RenameGroup(ctx context.Context, groupID, name string) (*model.RoleGroup, error) {
	id, err := strconv.Atoi(groupID)
	if err != nil || id < 1 {
		return nil, ErrOrganizationRoleRequestInvalid
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrOrganizationRoleGroupNameInvalid
	}
	group, err := s.roleGroups.GetByID(ctx, uint(id))
	if err != nil {
		return nil, err
	}
	if existing, err := s.roleGroups.GetByName(ctx, name); err == nil && existing.ID != group.ID {
		return nil, ErrDuplicateName
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	group.Name = name
	updated, err := s.roleGroups.Update(ctx, group)
	if err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{Module: "iam", Action: "update", ResourceType: "role_group", ResourceID: groupID, After: map[string]string{"name": updated.Name}})
	}
	return updated, nil
}

// DeleteGroup 只删除展示分组；组内角色会回到默认角色组显示，既有角色权限和成员绑定不受影响。
func (s *organizationRoleService) DeleteGroup(ctx context.Context, groupID string) error {
	id, err := strconv.Atoi(groupID)
	if err != nil || id < 1 {
		return ErrOrganizationRoleRequestInvalid
	}
	group, err := s.roleGroups.GetByID(ctx, uint(id))
	if err != nil {
		return err
	}
	if s.tx == nil {
		return fmt.Errorf("organization role transaction manager is required")
	}
	if err := s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.roles.ClearRoleGroup(txCtx, group.ID); err != nil {
			return err
		}
		return s.roleGroups.Delete(txCtx, group)
	}); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{Module: "iam", Action: "delete", ResourceType: "role_group", ResourceID: groupID})
	}
	return nil
}

// ReorderGroups 使用完整排序快照提交，避免拖动后只更新局部节点而出现重复顺序。
func (s *organizationRoleService) ReorderGroups(ctx context.Context, groupIDs []uint) error {
	groups, err := s.roleGroups.List(ctx)
	if err != nil {
		return err
	}
	if len(groupIDs) != len(groups) {
		return ErrOrganizationRoleRequestInvalid
	}
	existing := make(map[uint]struct{}, len(groups))
	for _, group := range groups {
		existing[group.ID] = struct{}{}
	}
	seen := make(map[uint]struct{}, len(groupIDs))
	for _, id := range groupIDs {
		if id == 0 {
			return ErrOrganizationRoleRequestInvalid
		}
		if _, ok := existing[id]; !ok {
			return ErrOrganizationRoleRequestInvalid
		}
		if _, duplicate := seen[id]; duplicate {
			return ErrOrganizationRoleRequestInvalid
		}
		seen[id] = struct{}{}
	}
	if s.tx == nil {
		return fmt.Errorf("organization role transaction manager is required")
	}
	byID := make(map[uint]*model.RoleGroup, len(groups))
	for index := range groups {
		byID[groups[index].ID] = &groups[index]
	}
	if err := s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		for sort, id := range groupIDs {
			group := byID[id]
			group.Sort = sort
			if _, err := s.roleGroups.Update(txCtx, group); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{Module: "iam", Action: "reorder", ResourceType: "role_group", After: map[string]any{"groupIds": groupIDs}})
	}
	return nil
}

func (s *organizationRoleService) CreateRole(ctx context.Context, req CreateOrganizationRoleRequest) (*model.Role, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" || req.GroupID == 0 {
		return nil, ErrOrganizationRoleRequestInvalid
	}
	if _, err := s.roleGroups.GetByID(ctx, req.GroupID); err != nil {
		return nil, err
	}
	if _, err := s.roles.GetRoleByName(ctx, name); err == nil {
		return nil, ErrDuplicateName
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	roles, err := s.roles.List(ctx)
	if err != nil {
		return nil, err
	}
	sort := 0
	for _, current := range roles {
		if current.RoleGroupID != nil && *current.RoleGroupID == req.GroupID {
			sort++
		}
	}
	role, err := s.roles.Create(ctx, &model.Role{Name: name, RoleGroupID: &req.GroupID, Sort: sort})
	if err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{Module: "iam", Action: "create", ResourceType: "role", ResourceID: strconv.FormatUint(uint64(role.ID), 10), After: map[string]any{"name": role.Name, "roleGroupId": req.GroupID}})
	}
	return role, nil
}

// RenameRole 仅更新角色名称。先加载完整角色再写回，避免组织页只上传 name 时
// 覆盖既有 rules，导致角色权限被意外清空。
func (s *organizationRoleService) RenameRole(ctx context.Context, roleID, name string) (*model.Role, error) {
	rid, err := strconv.Atoi(roleID)
	if err != nil || rid < 1 {
		return nil, ErrOrganizationRoleRequestInvalid
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrOrganizationRoleRequestInvalid
	}
	role, err := s.roles.GetRoleByID(ctx, rid)
	if err != nil {
		return nil, err
	}
	if existing, err := s.roles.GetRoleByName(ctx, name); err == nil && existing.ID != role.ID {
		return nil, ErrDuplicateName
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	role.Name = name
	updated, err := s.roles.Update(ctx, role)
	if err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{Module: "iam", Action: "update", ResourceType: "role", ResourceID: roleID, After: map[string]string{"name": updated.Name}})
	}
	return updated, nil
}

func (s *organizationRoleService) MoveRole(ctx context.Context, roleID, groupID string) (*model.Role, error) {
	rid, err := strconv.Atoi(roleID)
	if err != nil {
		return nil, ErrOrganizationRoleRequestInvalid
	}
	gid, err := strconv.Atoi(groupID)
	if err != nil || gid < 1 {
		return nil, ErrOrganizationRoleRequestInvalid
	}
	role, err := s.roles.GetRoleByID(ctx, rid)
	if err != nil {
		return nil, err
	}
	if _, err := s.roleGroups.GetByID(ctx, uint(gid)); err != nil {
		return nil, err
	}
	roles, err := s.roles.List(ctx)
	if err != nil {
		return nil, err
	}
	sort := 0
	for _, current := range roles {
		if current.RoleGroupID != nil && *current.RoleGroupID == uint(gid) && current.ID != role.ID {
			sort++
		}
	}
	role.RoleGroupID = ptrUint(uint(gid))
	role.Sort = sort
	updated, err := s.roles.Update(ctx, role)
	if err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{Module: "iam", Action: "update", ResourceType: "role_group", ResourceID: roleID, After: map[string]uint{"roleGroupId": uint(gid)}})
	}
	return updated, nil
}

// ReorderRoles 保存单个角色组内的完整角色排序；跨角色组移动由调整分组接口承担。
func (s *organizationRoleService) ReorderRoles(ctx context.Context, groupID string, roleIDs []uint) error {
	gid, err := strconv.Atoi(groupID)
	if err != nil || gid < 1 {
		return ErrOrganizationRoleRequestInvalid
	}
	groups, err := s.roleGroups.List(ctx)
	if err != nil {
		return err
	}
	defaultGroup, err := s.defaultGroup(ctx, groups)
	if err != nil {
		return err
	}
	if _, err := s.roleGroups.GetByID(ctx, uint(gid)); err != nil {
		return err
	}
	roles, err := s.roles.List(ctx)
	if err != nil {
		return err
	}
	groupRoles := make([]*model.Role, 0)
	for index := range roles {
		role := &roles[index]
		roleGroupID := defaultGroup.ID
		if role.RoleGroupID != nil {
			roleGroupID = *role.RoleGroupID
		}
		if roleGroupID == uint(gid) {
			groupRoles = append(groupRoles, role)
		}
	}
	if len(roleIDs) != len(groupRoles) {
		return ErrOrganizationRoleRequestInvalid
	}
	byID := make(map[uint]*model.Role, len(groupRoles))
	for _, role := range groupRoles {
		byID[role.ID] = role
	}
	seen := make(map[uint]struct{}, len(roleIDs))
	for _, id := range roleIDs {
		if _, ok := byID[id]; !ok {
			return ErrOrganizationRoleRequestInvalid
		}
		if _, duplicate := seen[id]; duplicate {
			return ErrOrganizationRoleRequestInvalid
		}
		seen[id] = struct{}{}
	}
	if s.tx == nil {
		return fmt.Errorf("organization role transaction manager is required")
	}
	if err := s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		for sort, id := range roleIDs {
			role := byID[id]
			role.Sort = sort
			if _, err := s.roles.Update(txCtx, role); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{Module: "iam", Action: "reorder", ResourceType: "role", ResourceID: groupID, After: map[string]any{"roleIds": roleIDs}})
	}
	return nil
}

func (s *organizationRoleService) ListMembers(ctx context.Context, roleID string, query model.MemberListQuery) (*model.MemberPage, error) {
	rid, err := strconv.Atoi(roleID)
	if err != nil || rid < 1 {
		return nil, ErrOrganizationRoleRequestInvalid
	}
	if _, err := s.roles.GetRoleByID(ctx, rid); err != nil {
		return nil, err
	}
	query.RoleID = uint(rid)
	return s.userSvc.ListPage(ctx, query)
}

func (s *organizationRoleService) AddMembers(ctx context.Context, roleID string, memberIDs []uint) error {
	return s.changeMembers(ctx, roleID, memberIDs, true)
}

func (s *organizationRoleService) RemoveMember(ctx context.Context, roleID, memberID string) error {
	mID, err := strconv.Atoi(memberID)
	if err != nil || mID < 1 {
		return ErrOrganizationRoleRequestInvalid
	}
	return s.changeMembers(ctx, roleID, []uint{uint(mID)}, false)
}

// ReplaceMemberRoles 在一个事务内完成差量解绑和绑定，避免前端逐角色提交导致成员
// 角色集合只更新一部分。传入的角色及成员均通过租户过滤读取，跨租户引用直接失败。
func (s *organizationRoleService) ReplaceMemberRoles(ctx context.Context, memberID string, roleIDs []uint) error {
	mID, err := strconv.Atoi(memberID)
	if err != nil || mID < 1 {
		return ErrOrganizationRoleRequestInvalid
	}
	if s.tx == nil {
		return fmt.Errorf("organization role transaction manager is required")
	}

	member, err := s.users.GetUserByID(ctx, uint(mID))
	if err != nil {
		return err
	}
	targetRoles := make(map[uint]*model.Role, len(roleIDs))
	for _, roleID := range roleIDs {
		if roleID == 0 {
			return ErrOrganizationRoleRequestInvalid
		}
		if _, duplicated := targetRoles[roleID]; duplicated {
			return ErrOrganizationRoleRequestInvalid
		}
		role, err := s.roles.GetRoleByID(ctx, int(roleID))
		if err != nil {
			return err
		}
		if err := ensureSameTenant(member.TenantID, role.TenantID, "member", member.ID, "role", role.ID); err != nil {
			return err
		}
		targetRoles[roleID] = role
	}

	currentRoles := make(map[uint]*model.Role, len(member.Roles))
	for index := range member.Roles {
		role := &member.Roles[index]
		currentRoles[role.ID] = role
	}
	if err := s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		for roleID, role := range currentRoles {
			if _, retained := targetRoles[roleID]; retained {
				continue
			}
			if err := s.users.DelRole(txCtx, role, member); err != nil {
				return err
			}
		}
		for roleID, role := range targetRoles {
			if _, existed := currentRoles[roleID]; existed {
				continue
			}
			if err := s.users.AddRole(txCtx, role, member); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "iam", Action: "replace", ResourceType: "member_role",
			ResourceID: memberID, After: map[string]any{"roleIds": roleIDs},
		})
	}
	return nil
}

func (s *organizationRoleService) changeMembers(ctx context.Context, roleID string, memberIDs []uint, add bool) error {
	rid, err := strconv.Atoi(roleID)
	if err != nil || rid < 1 || len(memberIDs) == 0 {
		return ErrOrganizationRoleRequestInvalid
	}
	role, err := s.roles.GetRoleByID(ctx, rid)
	if err != nil {
		return err
	}
	if s.tx == nil {
		return fmt.Errorf("organization role transaction manager is required")
	}
	if err := s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		for _, memberID := range memberIDs {
			member, err := s.users.GetUserByID(txCtx, memberID)
			if err != nil {
				return err
			}
			if err := ensureSameTenant(member.TenantID, role.TenantID, "member", member.ID, "role", role.ID); err != nil {
				return err
			}
			if add {
				err = s.users.AddRole(txCtx, role, member)
			} else {
				err = s.users.DelRole(txCtx, role, member)
			}
			if err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	if s.audit != nil {
		action := "unbind"
		if add {
			action = "bind"
		}
		s.audit.Record(ctx, auditservice.Entry{Module: "iam", Action: action, ResourceType: "role_member", ResourceID: roleID, After: map[string]any{"memberIds": memberIDs}})
	}
	return nil
}

func (s *organizationRoleService) defaultGroup(ctx context.Context, groups []model.RoleGroup) (*model.RoleGroup, error) {
	for i := range groups {
		if groups[i].Name == model.DefaultRoleGroupName {
			return &groups[i], nil
		}
	}
	group, err := s.roleGroups.Create(ctx, &model.RoleGroup{Name: model.DefaultRoleGroupName})
	if err == nil {
		return group, nil
	}
	// 并发首次进入组织页时可能同时创建默认组；冲突后读取赢家即可。
	if existing, getErr := s.roleGroups.GetByName(ctx, model.DefaultRoleGroupName); getErr == nil {
		return existing, nil
	}
	return nil, err
}

func ptrUint(value uint) *uint { return &value }
