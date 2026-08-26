package repository

import (
	"context"

	"evolyn/internal/contextx"
	"evolyn/internal/infrastructure"
	"evolyn/internal/platform/iam/model"

	"gorm.io/gorm"
)

type memberInvitationRepository struct{ db *gorm.DB }

func newMemberInvitationRepository(db *gorm.DB) MemberInvitationRepository {
	return &memberInvitationRepository{db: db}
}

func (r *memberInvitationRepository) withContext(ctx context.Context) *gorm.DB {
	return infrastructure.ResolveDB(ctx, r.db)
}

func (r *memberInvitationRepository) Create(ctx context.Context, invitation *model.MemberInvitation) (*model.MemberInvitation, error) {
	if err := r.withContext(ctx).Create(invitation).Error; err != nil {
		return nil, err
	}
	return invitation, nil
}

func (r *memberInvitationRepository) CreateBatch(ctx context.Context, invitations []model.MemberInvitation) error {
	if len(invitations) == 0 {
		return nil
	}
	return r.withContext(ctx).Create(&invitations).Error
}

// GetByToken 按单人邀请 token 查询：剥离调用链租户（接受邀请时尚未进入
// 目标租户），token 全局唯一（部分唯一索引兜底）
func (r *memberInvitationRepository) GetByToken(ctx context.Context, token string) (*model.MemberInvitation, error) {
	invitation := new(model.MemberInvitation)
	if err := r.withContext(contextx.DetachTenant(ctx)).Where("invite_token = ?", token).First(invitation).Error; err != nil {
		return nil, err
	}
	return invitation, nil
}

// MarkAccepted 状态流转 accepted：按 ID 白名单更新 status，软删与租户过滤
// 由 Callback 保证（跨租户 ID 更新 0 行时返回错误，禁止静默成功）
func (r *memberInvitationRepository) MarkAccepted(ctx context.Context, invitation *model.MemberInvitation) error {
	result := r.withContext(ctx).Model(&model.MemberInvitation{}).
		Where("id = ? AND status = ?", invitation.ID, model.MemberInvitationPending).
		Update("status", model.MemberInvitationAccepted)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *memberInvitationRepository) GetPublicLink(ctx context.Context) (*model.TenantPublicInvitationLink, error) {
	link := new(model.TenantPublicInvitationLink)
	if err := r.withContext(ctx).First(link).Error; err != nil {
		return nil, err
	}
	return link, nil
}

func (r *memberInvitationRepository) GetPublicLinkByToken(ctx context.Context, token string) (*model.TenantPublicInvitationLink, error) {
	link := new(model.TenantPublicInvitationLink)
	if err := r.withContext(contextx.DetachTenant(ctx)).Where("token = ?", token).First(link).Error; err != nil {
		return nil, err
	}
	return link, nil
}

func (r *memberInvitationRepository) CreatePublicLink(ctx context.Context, link *model.TenantPublicInvitationLink) (*model.TenantPublicInvitationLink, error) {
	if err := r.withContext(ctx).Create(link).Error; err != nil {
		return nil, err
	}
	return link, nil
}

func (r *memberInvitationRepository) UpdatePublicLink(ctx context.Context, link *model.TenantPublicInvitationLink) (*model.TenantPublicInvitationLink, error) {
	if err := r.withContext(ctx).Model(&model.TenantPublicInvitationLink{}).Where("id = ?", link.ID).
		Select("enabled", "creator_member_id").Updates(link).Error; err != nil {
		return nil, err
	}
	return link, nil
}

func (r *memberInvitationRepository) Migrate() error {
	return r.db.AutoMigrate(&model.MemberInvitation{}, &model.TenantPublicInvitationLink{})
}
