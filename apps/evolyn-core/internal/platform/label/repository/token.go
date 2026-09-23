package repository

import (
	"context"
	"time"

	"evolyn/internal/infrastructure"
	kernel "evolyn/internal/model"
	"evolyn/internal/platform/label/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type QRTokenRepository interface {
	Ensure(ctx context.Context, candidate *model.QRToken) (*model.QRToken, error)
	GetByToken(ctx context.Context, token string) (*model.QRToken, error)
	IncrementScan(ctx context.Context, id uint) error
	Migrate() error
}

type qrTokenRepository struct{ db *gorm.DB }

func NewQRTokenRepository(db *gorm.DB) QRTokenRepository { return &qrTokenRepository{db: db} }

func (r *qrTokenRepository) session(ctx context.Context) *gorm.DB {
	return infrastructure.ResolveDB(ctx, r.db)
}

// Ensure 以目标唯一键幂等复用 Token。并发首次打印时冲突更新不改变既有
// 随机 Token，随后按目标键回读同一行。
func (r *qrTokenRepository) Ensure(ctx context.Context, candidate *model.QRToken) (*model.QRToken, error) {
	err := r.session(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "form_id"}, {Name: "record_id"}, {Name: "template_id"}, {Name: "target_type"}},
		DoUpdates: clause.Assignments(map[string]any{"enabled": true, "expire_at": nil, "updated_at": time.Now()}),
	}).Create(candidate).Error
	if err != nil {
		return nil, err
	}
	var token model.QRToken
	err = r.session(ctx).Where(
		"tenant_id = ? AND form_id = ? AND record_id = ? AND template_id = ? AND target_type = ?",
		candidate.TenantID, candidate.FormID, candidate.RecordID, candidate.TemplateID, candidate.TargetType,
	).First(&token).Error
	return &token, err
}

func (r *qrTokenRepository) GetByToken(ctx context.Context, token string) (*model.QRToken, error) {
	var row model.QRToken
	err := r.session(ctx).Where("token = ?", token).First(&row).Error
	return &row, err
}

func (r *qrTokenRepository) IncrementScan(ctx context.Context, id uint) error {
	now := kernel.JSONTime(time.Now())
	result := r.session(ctx).Model(&model.QRToken{}).Where("id = ? AND enabled = TRUE").
		Updates(map[string]any{"scan_count": gorm.Expr("scan_count + 1"), "last_scan_at": &now})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *qrTokenRepository) Migrate() error { return r.db.AutoMigrate(&model.QRToken{}) }
