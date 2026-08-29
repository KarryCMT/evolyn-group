package service

import (
	"context"
	"fmt"
	"strconv"

	auditservice "evolyn/internal/platform/audit/service"
	"evolyn/internal/platform/iam/model"
	"evolyn/internal/platform/iam/repository"
)

// DepartmentNode 部门树节点（平表 → 树，前端组织架构组件直接消费）
type DepartmentNode struct {
	model.Department
	Children []*DepartmentNode `json:"children"`
}

type departmentService struct {
	departmentRepo repository.DepartmentRepository
	userRepo       repository.UserRepository
	audit          auditservice.Recorder
}

func NewDepartmentService(
	departmentRepo repository.DepartmentRepository,
	userRepo repository.UserRepository,
	audit auditservice.Recorder,
) DepartmentService {
	return &departmentService{
		departmentRepo: departmentRepo,
		userRepo:       userRepo,
		audit:          audit,
	}
}

func (s *departmentService) List(ctx context.Context) ([]model.Department, error) {
	return s.departmentRepo.List(ctx)
}

func (s *departmentService) Tree(ctx context.Context) ([]*DepartmentNode, error) {
	depts, err := s.departmentRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	return BuildDepartmentTree(depts), nil
}

// Create 创建部门（FIX-015）：父部门非空时必须存在且与子部门同租户
// （ctx 租户过滤保证），并禁止自引用
func (s *departmentService) Create(ctx context.Context, dept *model.Department) (*model.Department, error) {
	if dept.Name == "" {
		return nil, fmt.Errorf("department name is required")
	}
	if dept.ParentId != nil {
		if err := s.validateParent(ctx, *dept.ParentId, 0); err != nil {
			return nil, err
		}
	}
	if dept.Status == "" {
		dept.Status = model.DeptActive
	}
	if dept.LeaderMemberID != nil {
		if err := s.validateLeader(ctx, *dept.LeaderMemberID); err != nil {
			return nil, err
		}
	}

	created, err := s.departmentRepo.Create(ctx, dept)
	if err == nil && s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "iam", Action: "create", ResourceType: "department",
			ResourceID: strconv.FormatUint(uint64(created.ID), 10),
			After:      map[string]any{"name": created.Name, "parentId": created.ParentId},
		})
	}
	return created, err
}

// Update 更新部门（FIX-015）：父部门须存在、同租户、非自身且不形成环
func (s *departmentService) Update(ctx context.Context, id string, dept *model.Department) (*model.Department, error) {
	did, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}
	// 先加载再更新（FIX-022）：伪造他租部门 ID 时租户过滤使 Update 影响
	// 0 行却返回成功，形成「假成功 + 假审计」
	if _, err := s.departmentRepo.GetByID(ctx, uint(did)); err != nil {
		return nil, err
	}
	if dept.ParentId != nil {
		if err := s.validateParent(ctx, *dept.ParentId, uint(did)); err != nil {
			return nil, err
		}
	}
	if dept.LeaderMemberID != nil {
		if err := s.validateLeader(ctx, *dept.LeaderMemberID); err != nil {
			return nil, err
		}
	}
	dept.ID = uint(did)

	updated, err := s.departmentRepo.Update(ctx, dept)
	if err == nil && s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "iam", Action: "update", ResourceType: "department",
			ResourceID: id,
			After:      map[string]any{"name": dept.Name, "parentId": dept.ParentId},
		})
	}
	return updated, err
}

func (s *departmentService) Delete(ctx context.Context, id string) error {
	did, err := strconv.Atoi(id)
	if err != nil {
		return err
	}
	// 先加载再删除（FIX-022）：跨租户 ID 必须显式拒绝而非静默 0 行成功
	if _, err := s.departmentRepo.GetByID(ctx, uint(did)); err != nil {
		return err
	}

	if err := s.departmentRepo.Delete(ctx, uint(did)); err != nil {
		return err
	}

	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "iam", Action: "delete", ResourceType: "department", ResourceID: id,
		})
	}
	return nil
}

// SetMemberDepartments 整体替换成员的部门归属（FIX-006）：加载成员实体，
// 与每个目标部门显式比对租户一致
func (s *departmentService) SetMemberDepartments(ctx context.Context, memberID string, departmentIDs []uint) error {
	mid, err := strconv.Atoi(memberID)
	if err != nil {
		return err
	}
	member, err := s.userRepo.GetUserByID(ctx, uint(mid))
	if err != nil {
		return err
	}

	// 目标部门必须存在且与成员同租户（租户过滤 + 显式比对双重防御）
	for _, id := range departmentIDs {
		dept, err := s.departmentRepo.GetByID(ctx, id)
		if err != nil {
			return fmt.Errorf("department %d not found", id)
		}
		if err := ensureSameTenant(member.TenantID, dept.TenantID, "member", member.ID, "department", dept.ID); err != nil {
			return err
		}
	}

	if err := s.departmentRepo.SetMemberDepartments(ctx, member, departmentIDs); err != nil {
		return err
	}

	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "iam", Action: "update", ResourceType: "member_department",
			ResourceID: memberID,
			After:      map[string]any{"departmentIds": departmentIDs},
		})
	}
	return nil
}

// validateLeader 部门负责人校验（迁移 000050）：负责人必须是租户内有效成员
// （ctx 租户过滤保证同租户，跨租户成员表现为不存在）；流程引擎
// department_manager 审批人解析依赖该语义（ADR-012 Phase 3）
func (s *departmentService) validateLeader(ctx context.Context, leaderMemberID uint) error {
	leader, err := s.userRepo.GetUserByID(ctx, leaderMemberID)
	if err != nil {
		return fmt.Errorf("leader member %d not found", leaderMemberID)
	}
	if leader.Status != model.MemberStatusActive || leader.ResignedAt != nil {
		return fmt.Errorf("leader member %d is not active", leaderMemberID)
	}
	return nil
}

// validateParent 父部门校验（FIX-015）：存在（租户过滤内，跨租户父表现为
// 不存在）、非自身、沿父链向上无环。selfID=0 表示创建场景无自身可比
func (s *departmentService) validateParent(ctx context.Context, parentID, selfID uint) error {
	if parentID == selfID {
		return fmt.Errorf("department cannot be its own parent")
	}

	// 环检测：从 parent 沿父链向上走，遇自身即成环；
	// 上限迭代次数防脏数据导致的死循环
	current := parentID
	for i := 0; i < 1024; i++ {
		dept, err := s.departmentRepo.GetByID(ctx, current)
		if err != nil {
			return fmt.Errorf("parent department %d not found", current)
		}
		if dept.ID == selfID {
			return fmt.Errorf("department %d cannot be moved under its descendant %d", selfID, parentID)
		}
		if dept.ParentId == nil {
			return nil // 到达根，无环
		}
		current = *dept.ParentId
	}
	return fmt.Errorf("department hierarchy too deep or cyclic at %d", parentID)
}

// BuildDepartmentTree 平表组树：ParentId=NULL 为根；孤儿节点（父被软删）
// 挂回根，保证不丢数据。纯函数便于单测
func BuildDepartmentTree(depts []model.Department) []*DepartmentNode {
	nodes := make(map[uint]*DepartmentNode, len(depts))
	for i := range depts {
		nodes[depts[i].ID] = &DepartmentNode{Department: depts[i]}
	}

	roots := make([]*DepartmentNode, 0)
	for _, n := range nodes {
		if n.ParentId != nil {
			if parent, ok := nodes[*n.ParentId]; ok && parent.ID != n.ID {
				parent.Children = append(parent.Children, n)
				continue
			}
		}
		// 父为 NULL（根）或父不存在（孤儿挂回根）
		roots = append(roots, n)
	}

	// map 遍历乱序：按 Order/ID 稳定排序（与平表排序口径一致）
	sortNodes(roots)
	return roots
}

func sortNodes(nodes []*DepartmentNode) {
	for _, n := range nodes {
		if len(n.Children) > 0 {
			sortNodes(n.Children)
		}
	}
	for i := 1; i < len(nodes); i++ {
		for j := i; j > 0; j-- {
			if nodes[j].Order < nodes[j-1].Order ||
				(nodes[j].Order == nodes[j-1].Order && nodes[j].ID < nodes[j-1].ID) {
				nodes[j], nodes[j-1] = nodes[j-1], nodes[j]
			} else {
				break
			}
		}
	}
}
