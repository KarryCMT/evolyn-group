package service

import (
	"testing"

	"evolyn/internal/platform/iam/model"

	"github.com/stretchr/testify/assert"
)

// ptr 测试辅助：ParentId 为 *uint（NULL = 根，FIX-015）
func ptr[V any](v V) *V { return &v }

func TestBuildDepartmentTree(t *testing.T) {
	depts := []model.Department{
		{ID: 1, ParentId: nil, Name: "总部", Order: 1},           // NULL = 根
		{ID: 2, ParentId: ptr(uint(1)), Name: "研发部", Order: 2},
		{ID: 3, ParentId: ptr(uint(1)), Name: "销售部", Order: 1},
		{ID: 4, ParentId: ptr(uint(2)), Name: "后端组", Order: 1},
		{ID: 5, ParentId: ptr(uint(99)), Name: "孤儿节点", Order: 1}, // 父不存在 → 挂回根
	}

	tree := BuildDepartmentTree(depts)

	// 根：总部 + 孤儿
	assert.Len(t, tree, 2)
	assert.Equal(t, uint(1), tree[0].ID)

	// 同级按 Order 排序：销售部(1) 在 研发部(2) 前
	children := tree[0].Children
	assert.Len(t, children, 2)
	assert.Equal(t, uint(3), children[0].ID)
	assert.Equal(t, uint(2), children[1].ID)

	// 二级挂载
	assert.Len(t, children[1].Children, 1)
	assert.Equal(t, uint(4), children[1].Children[0].ID)

	// 空表
	assert.Empty(t, BuildDepartmentTree(nil))
}
