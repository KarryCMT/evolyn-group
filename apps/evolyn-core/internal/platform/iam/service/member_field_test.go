package service

import (
	"context"
	"testing"

	"evolyn/internal/platform/iam/model"
	"evolyn/internal/platform/iam/repository"

	"github.com/stretchr/testify/assert"
)

// fieldSettingRepoStub 字段配置服务的最小仓储桩：内存行集模拟
// (tenant_id, field_key) 行为，UpdateWithRevision 按乐观锁语义判定命中
type fieldSettingRepoStub struct {
	repository.MemberFieldSettingRepository
	rows     []model.MemberFieldSetting
	nextID   uint
	updated  map[string]interface{}
	bumped   bool
	conflict bool
}

func newFieldSettingRepoStub() *fieldSettingRepoStub {
	stub := &fieldSettingRepoStub{nextID: 100, updated: map[string]interface{}{}}
	// 以默认配置预置一行可调字段（alias），其余按需由 ensureDefaults 补齐路径覆盖
	for _, def := range model.MemberFieldRegistry() {
		stub.rows = append(stub.rows, model.MemberFieldSetting{
			ID:               stub.nextID,
			FieldKey:         def.Key,
			PersonalVisible:  def.PersonalVisible,
			PersonalEditable: def.PersonalEditable,
			CardVisible:      def.CardVisible,
			Revision:         1,
		})
		stub.nextID++
	}
	return stub
}

func (r *fieldSettingRepoStub) ListByTenant(_ context.Context) ([]model.MemberFieldSetting, error) {
	return r.rows, nil
}

func (r *fieldSettingRepoStub) CreateBatch(_ context.Context, settings []model.MemberFieldSetting) error {
	r.rows = append(r.rows, settings...)
	return nil
}

func (r *fieldSettingRepoStub) UpdateWithRevision(_ context.Context, id uint, revision int64, updates map[string]interface{}) (bool, error) {
	if r.conflict {
		return false, nil
	}
	for i := range r.rows {
		if r.rows[i].ID == id {
			if r.rows[i].Revision != revision {
				return false, nil
			}
			// 模拟真实更新语义：应用布尔开关并推进版本号
			if v, ok := updates["personal_visible"].(bool); ok {
				r.rows[i].PersonalVisible = v
			}
			if v, ok := updates["personal_editable"].(bool); ok {
				r.rows[i].PersonalEditable = v
			}
			if v, ok := updates["card_visible"].(bool); ok {
				r.rows[i].CardVisible = v
			}
			r.rows[i].Revision++
			r.updated["hit"] = true
			return true, nil
		}
	}
	return false, nil
}

func (r *fieldSettingRepoStub) BumpRevision(_ context.Context) error {
	r.bumped = true
	return nil
}

func newMemberFieldServiceForTest(stub *fieldSettingRepoStub) MemberFieldService {
	// passThroughTx 为 addmember_tx_test.go 中的事务透传替身；
	// boolPtr 复用 register_phone_test.go 中的既有定义
	return NewMemberFieldService(passThroughTx{}, stub, nil)
}

func TestMemberFieldGetSnapshotFillsDefaults(t *testing.T) {
	// 空库：ensureDefaults 补齐 15 个预置字段，快照恒完整
	stub := &fieldSettingRepoStub{}
	svc := newMemberFieldServiceForTest(stub)
	ctx := tenantCtx(t, 7)

	snapshot, err := svc.GetSnapshot(ctx)
	assert.NoError(t, err)
	assert.Len(t, snapshot.Fields, 15)
	assert.Equal(t, int64(1), snapshot.Revision)
	// 注册表顺序即返回顺序，首字段为姓名
	assert.Equal(t, model.MemberFieldKeyName, snapshot.Fields[0].Key)
	assert.True(t, snapshot.Fields[0].PersonalVisible)
	assert.False(t, snapshot.Fields[0].PersonalEditable)
	assert.True(t, snapshot.Fields[0].CardLocked)
	assert.Len(t, stub.rows, 15)
}

func TestMemberFieldUpdateFieldValidatesKey(t *testing.T) {
	svc := newMemberFieldServiceForTest(newFieldSettingRepoStub())
	_, err := svc.UpdateField(tenantCtx(t, 7), "not-exists", &MemberFieldSettingUpdateRequest{CardVisible: boolPtr(true), Revision: 1})
	assert.ErrorIs(t, err, ErrMemberFieldNotFound)
}

func TestMemberFieldUpdateFieldRejectsLockedSwitches(t *testing.T) {
	svc := newMemberFieldServiceForTest(newFieldSettingRepoStub())
	ctx := tenantCtx(t, 7)

	// 手机可见性锁定：关闭被拒
	_, err := svc.UpdateField(ctx, model.MemberFieldKeyMobile, &MemberFieldSettingUpdateRequest{PersonalVisible: boolPtr(false), Revision: 1})
	assert.ErrorIs(t, err, ErrMemberFieldLocked)

	// 姓名卡片锁定：关闭卡片被拒
	_, err = svc.UpdateField(ctx, model.MemberFieldKeyName, &MemberFieldSettingUpdateRequest{CardVisible: boolPtr(false), Revision: 1})
	assert.ErrorIs(t, err, ErrMemberFieldLocked)

	// 编号可编辑锁定：开启被拒
	_, err = svc.UpdateField(ctx, model.MemberFieldKeyCode, &MemberFieldSettingUpdateRequest{PersonalEditable: boolPtr(true), Revision: 1})
	assert.ErrorIs(t, err, ErrMemberFieldLocked)
}

func TestMemberFieldUpdateFieldLinkageRules(t *testing.T) {
	svc := newMemberFieldServiceForTest(newFieldSettingRepoStub())
	ctx := tenantCtx(t, 7)

	// 可编辑=true 但可见=false：联动冲突
	_, err := svc.UpdateField(ctx, model.MemberFieldKeyAlias, &MemberFieldSettingUpdateRequest{PersonalEditable: boolPtr(true), Revision: 1})
	assert.ErrorIs(t, err, ErrMemberFieldConfigInvalid)

	// 关闭可见自动联动关闭可编辑：先开启（可见+可编辑）
	snapshot, err := svc.UpdateField(ctx, model.MemberFieldKeyAlias, &MemberFieldSettingUpdateRequest{
		PersonalVisible: boolPtr(true), PersonalEditable: boolPtr(true), Revision: 1,
	})
	assert.NoError(t, err)
	alias := fieldOfSnapshot(snapshot, model.MemberFieldKeyAlias)
	assert.True(t, alias.PersonalVisible)
	assert.True(t, alias.PersonalEditable)
	assert.Equal(t, int64(2), snapshot.Revision)

	// 再仅关闭可见：服务端兜底同步关闭可编辑（文档 3.2）
	snapshot, err = svc.UpdateField(ctx, model.MemberFieldKeyAlias, &MemberFieldSettingUpdateRequest{PersonalVisible: boolPtr(false), Revision: 2})
	assert.NoError(t, err)
	alias = fieldOfSnapshot(snapshot, model.MemberFieldKeyAlias)
	assert.False(t, alias.PersonalVisible)
	assert.False(t, alias.PersonalEditable)
}

func TestMemberFieldUpdateFieldRevisionConflict(t *testing.T) {
	stub := newFieldSettingRepoStub()
	stub.conflict = true // 模拟其他管理员已先行修改
	svc := newMemberFieldServiceForTest(stub)

	_, err := svc.UpdateField(tenantCtx(t, 7), model.MemberFieldKeyAlias, &MemberFieldSettingUpdateRequest{CardVisible: boolPtr(true), Revision: 1})
	assert.ErrorIs(t, err, ErrMemberFieldConfigConflict)
}

func TestMemberFieldUpdateFieldBumpsTenantRevision(t *testing.T) {
	stub := newFieldSettingRepoStub()
	svc := newMemberFieldServiceForTest(stub)

	snapshot, err := svc.UpdateField(tenantCtx(t, 7), model.MemberFieldKeyEducation, &MemberFieldSettingUpdateRequest{CardVisible: boolPtr(true), Revision: 1})
	assert.NoError(t, err)
	assert.Equal(t, int64(2), snapshot.Revision)
	assert.True(t, stub.bumped, "单字段更新后应推进整页版本号")
}

func fieldOfSnapshot(snapshot *model.MemberFieldConfigSnapshot, key string) model.MemberFieldSettingView {
	for _, field := range snapshot.Fields {
		if field.Key == key {
			return field
		}
	}
	return model.MemberFieldSettingView{}
}
