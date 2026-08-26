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
