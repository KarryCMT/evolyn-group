package service

import (
	"context"
	"errors"
	"fmt"

	"evolyn/internal/platform/httpx"
	apperrors "evolyn/internal/platform/workbench"
	"evolyn/internal/platform/workbench/model"
	"evolyn/internal/platform/workbench/repository"

	"gorm.io/gorm"
)

type workbenchService struct {
	tx   TxManager
	repo Repository
}

// NewWorkbenchService 自定义工作台域服务工厂（ADR-007 域模块化）
func NewWorkbenchService(tx TxManager, repo repository.Repository) WorkbenchService {
	return &workbenchService{tx: tx, repo: repo}
}

func (s *workbenchService) Get(ctx context.Context, tenantID uint) (*model.WorkbenchView, error) {
	record, err := s.repo.FindByTenant(ctx, tenantID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 行缺失：前端以 data=null 回退本地默认布局（种子兜底外的异常路径）
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &model.WorkbenchView{Revision: record.Revision, Document: jsonRaw(record.Content)}, nil
}

func (s *workbenchService) Save(
	ctx context.Context, tenantID uint, req *model.SaveWorkbenchRequest,
) (*model.WorkbenchView, error) {
	// 服务端终审：结构校验 + 规范化（剔除未知字段）后再落库
	normalized, issue := ValidateWorkbenchDocument(req.Document)
	if issue != nil {
		return nil, httpx.Wrap(apperrors.ErrDocumentInvalid, issue)
	}
	if req.Revision < 0 {
		return nil, httpx.Wrap(apperrors.ErrDocumentInvalid,
			fmt.Errorf("workbench revision must be non-negative, got %d", req.Revision))
	}

	view := new(model.WorkbenchView)
	err := s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		record, err := s.repo.FindByTenant(ctx, tenantID)
		switch {
		case err == nil:
			// 已有记录：revision 口令不匹配即并发覆盖，拒绝
			if req.Revision != record.Revision {
				return apperrors.ErrRevisionConflict
			}
			// 先固定新版本号再更新，避免仓储实现原地修改记录影响返回值
			newRevision := record.Revision + 1
			updated, err := s.repo.UpdateContentWithRevision(ctx, record.ID, record.Revision, model.Content(normalized))
			if err != nil {
				return err
			}
			if !updated {
				// 事务内重读后仍失配：并发窗口内的版本过期
				return apperrors.ErrRevisionConflict
			}
			view.Revision = newRevision
			view.Document = normalized
			return nil
		case errors.Is(err, gorm.ErrRecordNotFound):
			// 种子缺失/被清理的兜底首存：口令必须为 0；携带非 0 视为过期
			if req.Revision != 0 {
				return apperrors.ErrRevisionConflict
			}
			record = &model.TenantWorkbench{
				TenantID: tenantID,
				Content:  model.Content(normalized),
				Revision: 1,
			}
			if err := s.repo.Create(ctx, record); err != nil {
				// 并发写入：另一请求已插入同租户行，映射为乐观锁冲突
				//（提示刷新后重试）而不是让唯一约束裸错误出网
				if isUniqueViolation(err) {
					return apperrors.ErrRevisionConflict
				}
				return err
			}
			view.Revision = 1
			view.Document = normalized
			return nil
		default:
			return err
		}
	})
	if err != nil {
		return nil, err
	}
	return view, nil
}

// SeedDefaults 租户开通事务内种子默认工作台：幂等（已有行直接成功），
// 默认文档合法性由 schema_test 保证（DefaultWorkbenchDocument 必过校验器）
func (s *workbenchService) SeedDefaults(ctx context.Context, tenantID uint) error {
	if _, err := s.repo.FindByTenant(ctx, tenantID); err == nil {
		return nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	err := s.repo.Create(ctx, &model.TenantWorkbench{
		TenantID: tenantID,
		Content:  model.Content(model.DefaultWorkbenchDocument),
		Revision: 1,
	})
	if err != nil && isUniqueViolation(err) {
		// 并发种子竞态（同租户重复开通重试）：幂等成功
		return nil
	}
	return err
}

// jsonRaw Content → json.RawMessage 视图转换（零拷贝语义一致，仅类型收窄）
func jsonRaw(content model.Content) []byte {
	return []byte(content)
}

// isUniqueViolation 判定 PostgreSQL 唯一约束冲突（口径同 form 域 storage.go）
func isUniqueViolation(err error) bool {
	var pgErr interface{ GetSQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.GetSQLState() == "23505"
	}
	return false
}
