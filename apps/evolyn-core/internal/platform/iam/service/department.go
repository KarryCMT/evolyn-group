package service

import (
	"context"
	"fmt"
	"strconv"

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
}

func NewDepartmentService(departmentRepo repository.DepartmentRepository, userRepo repository.UserRepository) DepartmentService {
	return &departmentService{
		departmentRepo: departmentRepo,
		userRepo:       userRepo,
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

func (s *departmentService) Create(ctx context.Context, dept *model.Department) (*model.Department, error) {
	if dept.Name == "" {
		return nil, fmt.Errorf("department name is required")
	}
	// 父部门校验：ParentId 非零必须存在（且属当前租户，由租户过滤保证）
	if dept.ParentId != 0 {
		if _, err := s.departmentRepo.GetByID(ctx, dept.ParentId); err != nil {
			return nil, fmt.Errorf("parent department %d not found", dept.ParentId)
		}
	}
	if dept.Status == "" {
		dept.Status = model.DeptActive
	}
	return s.departmentRepo.Create(ctx, dept)
}

func (s *departmentService) Update(ctx context.Context, id string, dept *model.Department) (*model.Department, error) {
	did, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}
	if dept.ParentId != 0 && dept.ParentId == uint(did) {
		return nil, fmt.Errorf("department cannot be its own parent")
	}
	dept.ID = uint(did)
	return s.departmentRepo.Update(ctx, dept)
}

func (s *departmentService) Delete(ctx context.Context, id string) error {
	did, err := strconv.Atoi(id)
	if err != nil {
		return err
	}
	return s.departmentRepo.Delete(ctx, uint(did))
}

// SetMemberDepartments 整体替换成员的部门归属
func (s *departmentService) SetMemberDepartments(ctx context.Context, memberID string, departmentIDs []uint) error {
	mid, err := strconv.Atoi(memberID)
	if err != nil {
		return err
	}
	// 部门归属校验：目标部门必须存在（租户过滤内）
	for _, id := range departmentIDs {
		if _, err := s.departmentRepo.GetByID(ctx, id); err != nil {
			return fmt.Errorf("department %d not found", id)
		}
	}
	return s.departmentRepo.SetMemberDepartments(ctx, &model.User{ID: uint(mid)}, departmentIDs)
}

// BuildDepartmentTree 平表组树：ParentId=0 为根；孤儿节点（父被软删）挂回根，
// 保证不丢数据。纯函数便于单测
func BuildDepartmentTree(depts []model.Department) []*DepartmentNode {
	nodes := make(map[uint]*DepartmentNode, len(depts))
	for i := range depts {
		nodes[depts[i].ID] = &DepartmentNode{Department: depts[i]}
	}

	roots := make([]*DepartmentNode, 0)
	for _, n := range nodes {
		parent, ok := nodes[n.ParentId]
		if ok && parent.ID != n.ID {
			parent.Children = append(parent.Children, n)
		} else {
			roots = append(roots, n)
		}
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
