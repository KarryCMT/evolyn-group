package service

import (
	"context"
	"testing"

	"evolyn/internal/infrastructure"
	auditrepository "evolyn/internal/platform/audit/repository"
	auditservice "evolyn/internal/platform/audit/service"
	iammodel "evolyn/internal/platform/iam/model"

	"github.com/stretchr/testify/assert"
)

// ---- 成员信息管理真库集成测试（成员信息管理开发文档第 9 章验收）----
//
// 复用 FIX-022 双租户环境（alpha/beta），叠加字段配置/成员档案/邀请接受
// 三条服务链路。覆盖：读取兜底补齐、租户隔离、锁定与联动校验、乐观锁冲突、
// 本人裁剪与写入拦截、管理员视图与编号唯一、邀请接受事务与重复消费。

// memberInfoEnv 在 secEnv 之上装配成员信息管理服务链
type memberInfoEnv struct {
	*secEnv
	fieldSvc   MemberFieldService
	profileSvc MemberProfileService
	inviteSvc  MemberInvitationService
	txManager  *infrastructure.TxManager
}

func newMemberInfoEnv(t *testing.T) *memberInfoEnv {
	t.Helper()
	env := newSecEnv(t)
	auditSvc := auditservice.NewService(auditrepository.NewRepository(env.db))
	txManager := infrastructure.NewTxManager(env.db)
	fieldSvc := NewMemberFieldService(txManager, env.iamRepo.MemberFieldSetting(), auditSvc)
	profileSvc := NewMemberProfileService(txManager, env.iamRepo.MemberProfile(), env.iamRepo.MemberFieldSetting(), env.iamRepo.User(), auditSvc)
	inviteSvc := NewMemberInvitationService(txManager, env.iamRepo.Invitation(), env.iamRepo.Department(), env.iamRepo.Account(), env.userSvc, env.iamRepo.MemberProfile(), auditSvc)
	return &memberInfoEnv{secEnv: env, fieldSvc: fieldSvc, profileSvc: profileSvc, inviteSvc: inviteSvc, txManager: txManager}
}

// MEMBER-FIELD-INT-1：空租户首次读取幂等补齐 15 字段；两租户配置互不影响
func TestMemberFieldINTSnapshotAndIsolation(t *testing.T) {
	env := newMemberInfoEnv(t)

	// 未注入 fieldSeeder 的租户：读取侧兜底补齐
	alphaSnapshot, err := env.fieldSvc.GetSnapshot(itCtx(env.alpha.ID))
	assert.NoError(t, err)
	assert.Len(t, alphaSnapshot.Fields, 15)
	assert.EqualValues(t, 15, env.rawCount(t, "SELECT COUNT(*) FROM tenant_member_field_settings WHERE tenant_id = ?", env.alpha.ID))

	// alpha 开启 alias（可见+可编辑+卡片），beta 不受影响
	_, err = env.fieldSvc.UpdateField(itCtx(env.alpha.ID), iammodel.MemberFieldKeyAlias, &MemberFieldSettingUpdateRequest{
		PersonalVisible: boolPtr(true), PersonalEditable: boolPtr(true), CardVisible: boolPtr(true), Revision: alphaSnapshot.Revision,
	})
	assert.NoError(t, err)

	betaSnapshot, err := env.fieldSvc.GetSnapshot(itCtx(env.beta.ID))
	assert.NoError(t, err)
	betaAlias := fieldOfSnapshot(betaSnapshot, iammodel.MemberFieldKeyAlias)
	assert.False(t, betaAlias.PersonalVisible, "租户间字段配置互不影响")
	// beta 仅注册表默认可见字段（name/mobile/email）为 true，未被 alpha 变更波及
	assert.EqualValues(t, 3, env.rawCount(t,
		"SELECT COUNT(*) FROM tenant_member_field_settings WHERE tenant_id = ? AND personal_visible", env.beta.ID))

	// 二次读取走库内行（不再补齐），数值保持
	again, err := env.fieldSvc.GetSnapshot(itCtx(env.alpha.ID))
	assert.NoError(t, err)
	assert.True(t, fieldOfSnapshot(again, iammodel.MemberFieldKeyAlias).PersonalEditable)
}

// MEMBER-FIELD-INT-2：乐观锁——过期 revision 的并发写被拒绝且不落库
func TestMemberFieldINTRevisionConflict(t *testing.T) {
	env := newMemberInfoEnv(t)
	ctx := itCtx(env.alpha.ID)

	snapshot, err := env.fieldSvc.GetSnapshot(ctx)
	assert.NoError(t, err)

	// 第一次更新推进版本到 2
	updated, err := env.fieldSvc.UpdateField(ctx, iammodel.MemberFieldKeyEducation, &MemberFieldSettingUpdateRequest{
		CardVisible: boolPtr(true), Revision: snapshot.Revision,
	})
	assert.NoError(t, err)
	assert.Equal(t, snapshot.Revision+1, updated.Revision)

	// 携带旧 revision 再写：冲突拒绝，库内值不变
	_, err = env.fieldSvc.UpdateField(ctx, iammodel.MemberFieldKeyEducation, &MemberFieldSettingUpdateRequest{
		CardVisible: boolPtr(false), Revision: snapshot.Revision,
	})
	assert.ErrorIs(t, err, ErrMemberFieldConfigConflict)
	latest, err := env.fieldSvc.GetSnapshot(ctx)
	assert.NoError(t, err)
	assert.True(t, fieldOfSnapshot(latest, iammodel.MemberFieldKeyEducation).CardVisible)
}

// MEMBER-PROFILE-INT-1：本人视图按可见裁剪；仅可编辑扩展字段可写；
// 手机号等账号安全字段经通用资料接口提交被拒绝
func TestMemberProfileINTMyProfileRules(t *testing.T) {
	env := newMemberInfoEnv(t)
	ctx := itCtx(env.alpha.ID)

	// 开启 alias（可见+可编辑），mobile 保持默认可见可编辑（锁定项）
	snapshot, err := env.fieldSvc.GetSnapshot(ctx)
	assert.NoError(t, err)
	_, err = env.fieldSvc.UpdateField(ctx, iammodel.MemberFieldKeyAlias, &MemberFieldSettingUpdateRequest{
		PersonalVisible: boolPtr(true), PersonalEditable: boolPtr(true), Revision: snapshot.Revision,
	})
	assert.NoError(t, err)

	// 本人写入已开启可编辑的扩展字段（alias）；hireDate 等未开启编辑的字段
	// 同样被拒绝，仅断言 alias 生效
	_, err = env.profileSvc.UpdateMyProfile(ctx, env.alphaMember.ID, map[string]string{
		iammodel.MemberFieldKeyHireDate: "2024-02-01",
	})
	assert.ErrorIs(t, err, ErrMemberProfileInvalid)
	view, err := env.profileSvc.UpdateMyProfile(ctx, env.alphaMember.ID, map[string]string{
		iammodel.MemberFieldKeyAlias: "帆小云",
	})
	assert.NoError(t, err)
	assert.Equal(t, "帆小云", view.Values[iammodel.MemberFieldKeyAlias])
	assert.Contains(t, view.EditableKeys, iammodel.MemberFieldKeyAlias)

	// 姓名（锁定不可编辑）与手机号（账号安全字段）提交均拒绝
	_, err = env.profileSvc.UpdateMyProfile(ctx, env.alphaMember.ID, map[string]string{iammodel.MemberFieldKeyName: "改名"})
	assert.ErrorIs(t, err, ErrMemberProfileInvalid)
	_, err = env.profileSvc.UpdateMyProfile(ctx, env.alphaMember.ID, map[string]string{iammodel.MemberFieldKeyMobile: "13900000000"})
	assert.ErrorIs(t, err, ErrMemberProfileInvalid)

	// 关闭 alias 可见后：读取裁剪 + 提交拒绝
	snapshot, err = env.fieldSvc.GetSnapshot(ctx)
	assert.NoError(t, err)
	_, err = env.fieldSvc.UpdateField(ctx, iammodel.MemberFieldKeyAlias, &MemberFieldSettingUpdateRequest{
		PersonalVisible: boolPtr(false), Revision: snapshot.Revision,
	})
	assert.NoError(t, err)
	view, err = env.profileSvc.GetMyProfile(ctx, env.alphaMember.ID)
	assert.NoError(t, err)
	assert.NotContains(t, view.Values, iammodel.MemberFieldKeyAlias, "个人设置中被隐藏的字段不下发")
	_, err = env.profileSvc.UpdateMyProfile(ctx, env.alphaMember.ID, map[string]string{iammodel.MemberFieldKeyAlias: "再改"})
	assert.ErrorIs(t, err, ErrMemberProfileInvalid)
}

// MEMBER-PROFILE-INT-2：管理员全量视图 + 卡片按配置裁剪 + 编号租户内唯一 +
// 日期格式与长度校验
func TestMemberProfileINTAdminViewAndValidation(t *testing.T) {
	env := newMemberInfoEnv(t)
	ctx := itCtx(env.alpha.ID)

	identifier := "A-0001"
	adminView, err := env.profileSvc.UpdateMemberProfile(ctx, env.alphaMember.ID, &MemberProfileUpdateRequest{
		Identifier: &identifier,
		Values:     map[string]string{iammodel.MemberFieldKeyPosition: "销售经理"},
	})
	assert.NoError(t, err)
	assert.Equal(t, "A-0001", adminView.Values[iammodel.MemberFieldKeyCode])
	// 卡片默认不含 position（cardVisible=false）：服务端裁剪生效
	assert.NotContains(t, adminView.CardValues, iammodel.MemberFieldKeyPosition)
	assert.Contains(t, adminView.CardValues, iammodel.MemberFieldKeyName, "姓名为卡片固定信息")

	// 编号租户内唯一：跨租户允许同编号（beta 成员可复用 alpha 的编号），
	// 同租户其他成员占用同编号被拒
	betaIdentifier := "A-0001"
	_, err = env.profileSvc.UpdateMemberProfile(itCtx(env.beta.ID), env.betaMember.ID, &MemberProfileUpdateRequest{Identifier: &betaIdentifier})
	assert.NoError(t, err, "编号唯一性仅限租户内，跨租户可复用")

	// victim 加入 alpha 后占用同编号：被拒且不落库
	second, err := env.userSvc.AddMember(ctx, &AddMemberRequest{AccountID: env.victim.ID})
	assert.NoError(t, err)
	_, err = env.profileSvc.UpdateMemberProfile(ctx, second.ID, &MemberProfileUpdateRequest{Identifier: &identifier})
	assert.ErrorIs(t, err, ErrMemberProfileInvalid)
	var dupCount int64
	assert.NoError(t, env.db.Raw(
		"SELECT COUNT(*) FROM member_profiles WHERE tenant_id = ? AND identifier = 'A-0001'",
		env.alpha.ID).Scan(&dupCount).Error)
	assert.EqualValues(t, 1, dupCount, "同租户同编号仍只有一条有效档案")
	// 同成员重写自身编号不受自身占用影响（excludeMemberID 语义）
	sameIdentifier := "A-0001"
	_, err = env.profileSvc.UpdateMemberProfile(ctx, env.alphaMember.ID, &MemberProfileUpdateRequest{Identifier: &sameIdentifier})
	assert.NoError(t, err)

	// 日期格式与长度校验
	_, err = env.profileSvc.UpdateMemberProfile(ctx, env.alphaMember.ID, &MemberProfileUpdateRequest{
		Values: map[string]string{iammodel.MemberFieldKeyBirthday: "1994/07/21"},
	})
	assert.ErrorIs(t, err, ErrMemberProfileInvalid)
	_, err = env.profileSvc.UpdateMemberProfile(ctx, env.alphaMember.ID, &MemberProfileUpdateRequest{
		Values: map[string]string{iammodel.MemberFieldKeyWorkplace: string(make([]rune, 51))},
	})
	assert.ErrorIs(t, err, ErrMemberProfileInvalid)
}

// INVITE-ACCEPT-INT-1：单人邀请接受事务——建成员/迁档案/绑部门/置 accepted；
// 重复消费与身份不匹配被拒；失败不落半写状态
func TestInviteINTAcceptPersonalInviteTx(t *testing.T) {
	env := newMemberInfoEnv(t)
	bg := context.Background()

	// 受邀账号：手机号与邀请一致
	invitee, err := env.iamRepo.Account().Create(bg, &iammodel.Account{Name: "invitee-1", Nickname: "受邀人", Phone: "13811112222"})
	assert.NoError(t, err)

	// 邀请档案：部门 + 扩展字段（含邀请侧命名 employeeNo/title/hiredAt）
	created, err := env.inviteSvc.Create(itCtx(env.alpha.ID), env.alphaMember.ID, MemberInvitationRequest{
		Name:       "受邀人",
		Phone:      "13811112222",
		Email:      "invitee@example.com",
		Alias:      "帆小云",
		EmployeeNo: "LYY-0009",
		Title:      "销售经理",
		HiredAt:    "2024-03-01",
	})
	assert.NoError(t, err)

	member, err := env.inviteSvc.AcceptPersonalInvite(bg, invitee.ID, created.InviteToken)
	assert.NoError(t, err)
	assert.Equal(t, env.alpha.ID, member.TenantID)

	// 档案已迁入统一 key；编号来自邀请 identifier
	profile, err := env.iamRepo.MemberProfile().GetByMember(itCtx(env.alpha.ID), member.ID)
	assert.NoError(t, err)
	assert.Equal(t, "帆小云", profile.Attributes[iammodel.MemberFieldKeyAlias])
	assert.Equal(t, "LYY-0009", profile.Attributes[iammodel.MemberFieldKeyEmployeeId])
	assert.Equal(t, "销售经理", profile.Attributes[iammodel.MemberFieldKeyPosition])
	assert.Equal(t, "2024-03-01", profile.Attributes[iammodel.MemberFieldKeyHireDate])

	// 邀请状态 accepted（重复消费按无效拒绝）
	assert.EqualValues(t, 1, env.rawCount(t,
		"SELECT COUNT(*) FROM member_invitations WHERE id = ? AND status = 'accepted'", created.ID))
	_, err = env.inviteSvc.AcceptPersonalInvite(bg, invitee.ID, created.InviteToken)
	assert.ErrorIs(t, err, ErrMemberInvitationAcceptInvalid)

	// 身份不匹配：手机号不同的账号消费同一邀请被拒（新邀请，pending）
	other, err := env.iamRepo.Account().Create(bg, &iammodel.Account{Name: "invitee-2", Nickname: "无关人", Phone: "13833334444"})
	assert.NoError(t, err)
	created2, err := env.inviteSvc.Create(itCtx(env.alpha.ID), env.alphaMember.ID, MemberInvitationRequest{
		Name: "另一位", Phone: "13855556666",
	})
	assert.NoError(t, err)
	_, err = env.inviteSvc.AcceptPersonalInvite(bg, other.ID, created2.InviteToken)
	assert.ErrorIs(t, err, ErrMemberInvitationAcceptInvalid)
	// 未接受的邀请保持 pending，且该账号未产生成员（无半写状态）
	assert.EqualValues(t, 1, env.rawCount(t,
		"SELECT COUNT(*) FROM member_invitations WHERE id = ? AND status = 'pending'", created2.ID))
	var otherMemberCount int64
	assert.NoError(t, env.db.Raw("SELECT COUNT(*) FROM users WHERE account_id = ?", other.ID).Scan(&otherMemberCount).Error)
	assert.EqualValues(t, 0, otherMemberCount)
}

// INVITE-ACCEPT-INT-2：非法日期的邀请档案在接受时被拦截，整体不落库
func TestInviteINTAcceptRejectsInvalidProfile(t *testing.T) {
	env := newMemberInfoEnv(t)
	bg := context.Background()

	invitee, err := env.iamRepo.Account().Create(bg, &iammodel.Account{Name: "invitee-3", Nickname: "受邀人", Phone: "13877778888"})
	assert.NoError(t, err)
	created, err := env.inviteSvc.Create(itCtx(env.alpha.ID), env.alphaMember.ID, MemberInvitationRequest{
		Name:    "受邀人",
		Phone:   "13877778888",
		HiredAt: "2024/03/05", // 非法日期格式
	})
	assert.NoError(t, err)

	_, err = env.inviteSvc.AcceptPersonalInvite(bg, invitee.ID, created.InviteToken)
	assert.ErrorIs(t, err, ErrMemberProfileInvalid)
	// 无成员、无档案、邀请保持 pending（事务回滚）
	var memberCount, profileCount int64
	assert.NoError(t, env.db.Raw("SELECT COUNT(*) FROM users WHERE tenant_id = ? AND account_id = ?", env.alpha.ID, invitee.ID).Scan(&memberCount).Error)
	assert.EqualValues(t, 0, memberCount)
	assert.NoError(t, env.db.Raw("SELECT COUNT(*) FROM member_profiles WHERE tenant_id = ?", env.alpha.ID).Scan(&profileCount).Error)
	assert.EqualValues(t, 0, profileCount)
	assert.EqualValues(t, 1, env.rawCount(t,
		"SELECT COUNT(*) FROM member_invitations WHERE id = ? AND status = 'pending'", created.ID))
}

// MEMBER-FIELD-INT-3：字段配置更新在真库的锁定项拒绝（数据库行已存在路径）
func TestMemberFieldINTLockedRejected(t *testing.T) {
	env := newMemberInfoEnv(t)
	ctx := itCtx(env.alpha.ID)

	snapshot, err := env.fieldSvc.GetSnapshot(ctx)
	assert.NoError(t, err)
	_, err = env.fieldSvc.UpdateField(ctx, iammodel.MemberFieldKeyMobile, &MemberFieldSettingUpdateRequest{
		PersonalVisible: boolPtr(false), Revision: snapshot.Revision,
	})
	assert.ErrorIs(t, err, ErrMemberFieldLocked)

	// 库内锁定字段配置保持默认
	var visible bool
	assert.NoError(t, env.db.Raw(
		"SELECT personal_visible FROM tenant_member_field_settings WHERE tenant_id = ? AND field_key = 'mobile'",
		env.alpha.ID).Scan(&visible).Error)
	assert.True(t, visible)
}
