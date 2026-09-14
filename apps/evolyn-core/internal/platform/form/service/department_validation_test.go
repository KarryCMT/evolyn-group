package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

type fakeDepartmentDirectory struct {
	valid map[string]bool
}

func (d fakeDepartmentDirectory) ResolveActiveDepartmentIDs(_ context.Context, references []string) (map[string]bool, error) {
	resolved := make(map[string]bool, len(references))
	for _, reference := range references {
		if d.valid[reference] {
			resolved[reference] = true
		}
	}
	return resolved, nil
}

// 部门 ID 的字符串形状通过并不代表该部门可用；目录终审必须收口不存在、停用和
// 跨租户引用。后两者在生产适配器中均不会进入当前租户的 active 目录。
func TestValidateDepartmentReferences(t *testing.T) {
	content := snapshot(snapItem("dept", "_widget_department", "所属部门", nil))
	service := &formService{departments: fakeDepartmentDirectory{valid: map[string]bool{"12": true}}}

	fieldErrors, err := service.validateDepartmentReferences(context.Background(), content, map[string]any{
		"_widget_department": "12",
	})
	assert.NoError(t, err)
	assert.Empty(t, fieldErrors)

	fieldErrors, err = service.validateDepartmentReferences(context.Background(), content, map[string]any{
		"_widget_department": "999",
	})
	assert.NoError(t, err)
	assert.Equal(t, RecordFieldErrors{
		"_widget_department": {"所属部门不存在、已停用或无权选择"},
	}, fieldErrors)
}

func TestValidateDepartmentReferencesFailsClosedWithoutDirectory(t *testing.T) {
	content := snapshot(snapItem("dept", "_widget_department", "所属部门", nil))
	service := &formService{}

	_, err := service.validateDepartmentReferences(context.Background(), content, map[string]any{
		"_widget_department": "12",
	})
	assert.EqualError(t, err, "department directory is not configured")
}
