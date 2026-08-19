package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	auditservice "evolyn/internal/platform/audit/service"
	"evolyn/internal/platform/iam/model"
	"evolyn/internal/platform/iam/repository"

	"gorm.io/gorm"
)

type groupService struct {
	userRepository  repository.UserRepository
	groupRepository repository.GroupRepository
	rbacRepository  repository.RBACRepository
	audit           auditservice.Recorder
}

func NewGroupService(
	groupRepository repository.GroupRepository,
	userRepository repository.UserRepository,
	rbacRepository repository.RBACRepository,
	audit auditservice.Recorder,
) GroupService {
	return &groupService{
		groupRepository: groupRepository,
		userRepository:  userRepository,
		rbacRepository:  rbacRepository,
		audit:           audit,
	}
}

func (g *groupService) List(ctx context.Context) ([]model.Group, error) {
	return g.groupRepository.List(ctx)
}

// Create 创建用户组（FIX-003）：名称租户内唯一，服务层预检给出友好错误，
// 数据库部分唯一索引兜底并发窗口
func (g *groupService) Create(ctx context.Context, user *model.User, group *model.Group) (*model.Group, error) {
	if err := g.ensureNameAvailable(ctx, group.Name, 0); err != nil {
		return nil, err
	}

	created, err := g.groupRepository.Create(ctx, user, group)
	if err == nil && g.audit != nil {
		g.audit.Record(ctx, auditservice.Entry{
			Module: "iam", Action: "create", ResourceType: "group",
			ResourceID: strconv.FormatUint(uint64(created.ID), 10),
			After:      map[string]string{"name": created.Name},
		})
	}
	return created, err
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
	// 先加载再更新（FIX-022）：伪造他租分组 ID 时租户过滤使 Update 影响
	// 0 行却返回成功，形成「假成功 + 假审计」
	if _, err := g.groupRepository.GetGroupByID(ctx, uint(gid)); err != nil {
		return nil, err
	}
	// 改名时校验租户内唯一（排除自身）
	if err := g.ensureNameAvailable(ctx, group.Name, uint(gid)); err != nil {
		return nil, err
	}

	group.ID = uint(gid)
	updated, err := g.groupRepository.Update(ctx, group)
	if err == nil && g.audit != nil {
		g.audit.Record(ctx, auditservice.Entry{
			Module: "iam", Action: "update", ResourceType: "group",
			ResourceID: id,
			After:      map[string]any{"name": group.Name, "describe": group.Describe},
		})
	}
	return updated, err
}

func (g *groupService) Delete(ctx context.Context, id string) error {
	gid, err := strconv.Atoi(id)
	if err != nil {
		return err
	}
	// 先加载再删除（FIX-022）：跨租户 ID 必须显式拒绝而非静默 0 行成功
	if _, err := g.groupRepository.GetGroupByID(ctx, uint(gid)); err != nil {
		return err
	}

	if err := g.groupRepository.Delete(ctx, uint(gid)); err != nil {
		return err
	}

	if g.audit != nil {
		g.audit.Record(ctx, auditservice.Entry{
			Module: "iam", Action: "delete", ResourceType: "group", ResourceID: id,
		})
	}
	return nil
}

func (g *groupService) GetUsers(ctx context.Context, id string) (model.Users, error) {
	gid, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}

	return g.groupRepository.GetUsers(ctx, &model.Group{ID: uint(gid)})
}

// AddUser 成员加入分组（FIX-006）：加载两端实体并校验同租户
func (g *groupService) AddUser(ctx context.Context, user *model.User, id string) error {
	if user == nil || user.ID == 0 {
		return fmt.Errorf("invaild user info")
	}

	gid, err := strconv.Atoi(id)
	if err != nil {
		return err
	}

	group, member, err := g.loadGroupAndMember(ctx, uint(gid), user.ID)
	if err != nil {
		return err
	}
	if err := ensureSameTenant(member.TenantID, group.TenantID, "member", member.ID, "group", group.ID); err != nil {
		return err
	}

	if err := g.groupRepository.AddUser(ctx, member, group); err != nil {
		return err
	}

	if g.audit != nil {
		g.audit.Record(ctx, auditservice.Entry{
			Module: "iam", Action: "bind", ResourceType: "group_member",
			ResourceID: fmt.Sprintf("%d:%d", group.ID, member.ID),
		})
	}
	return nil
}

func (g *groupService) DelUser(ctx context.Context, user *model.User, id string) error {
	if user == nil || user.ID == 0 {
		return fmt.Errorf("invaild user info")
	}

	gid, err := strconv.Atoi(id)
	if err != nil {
		return err
	}

	group, member, err := g.loadGroupAndMember(ctx, uint(gid), user.ID)
	if err != nil {
		return err
	}
	if err := ensureSameTenant(member.TenantID, group.TenantID, "member", member.ID, "group", group.ID); err != nil {
		return err
	}

	if err := g.groupRepository.DelUser(ctx, member, group); err != nil {
		return err
	}

	if g.audit != nil {
		g.audit.Record(ctx, auditservice.Entry{
			Module: "iam", Action: "unbind", ResourceType: "group_member",
			ResourceID: fmt.Sprintf("%d:%d", group.ID, member.ID),
		})
	}
	return nil
}

// AddRole 分组绑定角色（FIX-006）：加载两端实体并校验同租户
func (g *groupService) AddRole(ctx context.Context, id, rid string) error {
	group, role, err := g.loadGroupAndRole(ctx, id, rid)
	if err != nil {
		return err
	}
	if err := ensureSameTenant(group.TenantID, role.TenantID, "group", group.ID, "role", role.ID); err != nil {
		return err
	}

	if err := g.groupRepository.AddRole(ctx, role, group); err != nil {
		return err
	}

	if g.audit != nil {
		g.audit.Record(ctx, auditservice.Entry{
			Module: "iam", Action: "bind", ResourceType: "group_role",
			ResourceID: fmt.Sprintf("%d:%d", group.ID, role.ID),
		})
	}
	return nil
}

func (g *groupService) DelRole(ctx context.Context, id, rid string) error {
	group, role, err := g.loadGroupAndRole(ctx, id, rid)
	if err != nil {
		return err
	}
	if err := ensureSameTenant(group.TenantID, role.TenantID, "group", group.ID, "role", role.ID); err != nil {
		return err
	}

	if err := g.groupRepository.DelRole(ctx, role, group); err != nil {
		return err
	}

	if g.audit != nil {
		g.audit.Record(ctx, auditservice.Entry{
			Module: "iam", Action: "unbind", ResourceType: "group_role",
			ResourceID: fmt.Sprintf("%d:%d", group.ID, role.ID),
		})
	}
	return nil
}

// ensureNameAvailable 租户内重名校验（FIX-003）：excludeSelf 用于更新场景排除自身
func (g *groupService) ensureNameAvailable(ctx context.Context, name string, excludeSelf uint) error {
	if name == "" {
		return fmt.Errorf("group name is required")
	}
	existing, err := g.groupRepository.GetGroupByName(ctx, name)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if existing != nil && existing.ID != excludeSelf {
		return fmt.Errorf("%w: group name %s already exists in tenant", ErrDuplicateName, name)
	}
	return nil
}

// loadGroupAndMember 加载分组与成员实体（ctx 携带租户时 Callback 已过滤，
// 跨租户 ID 直接 NotFound；此处再显式比对兜底）
func (g *groupService) loadGroupAndMember(ctx context.Context, groupID, memberID uint) (*model.Group, *model.User, error) {
	group, err := g.groupRepository.GetGroupByID(ctx, groupID)
	if err != nil {
		return nil, nil, err
	}
	member, err := g.userRepository.GetUserByID(ctx, memberID)
	if err != nil {
		return nil, nil, err
	}
	return group, member, nil
}

func (g *groupService) loadGroupAndRole(ctx context.Context, id, rid string) (*model.Group, *model.Role, error) {
	gid, err := strconv.Atoi(id)
	if err != nil {
		return nil, nil, err
	}
	group, err := g.groupRepository.GetGroupByID(ctx, uint(gid))
	if err != nil {
		return nil, nil, err
	}

	roleId, err := strconv.Atoi(rid)
	if err != nil {
		return nil, nil, err
	}
	role, err := g.rbacRepository.GetRoleByID(ctx, roleId)
	if err != nil {
		return nil, nil, err
	}
	return group, role, nil
}
