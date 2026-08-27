package service

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	apperrors "evolyn/internal/platform/application"
	"evolyn/internal/platform/application/model"
	"evolyn/internal/platform/application/repository"
	"evolyn/internal/platform/httpx"
)

// FormDirectory 表单目录窄端口（M2-资产-1，菜单读侧资产可见性与 target
// 投影）：由表单域仓储在装配层适配；application 域不反向依赖 form 域包。
type FormDirectory interface {
	// ExistingFormIDs 返回 ids 中存在且未软删的表单 ID 集合（租户过滤由
	// ctx 承载，跨租户 ID 自然不在结果中）
	ExistingFormIDs(ctx context.Context, ids []uint) (map[uint]bool, error)
}

// MenuMaintenance 表单资产菜单节点维护窄端口（M2-资产-1）：表单域在创建/
// 改名/删除的事务内调用，节点写入与 menu_revision 递增随之加入同一事务；
// 菜单管理写接口（分组/移动/重排）仍随 M2-菜单-3 落地，本端口只承载
// 资产生命周期驱动的节点维护。
type MenuMaintenance interface {
	// AttachFormEntry 表单创建事务内挂 form 资产节点（target 指向表单 ID）；
	// parentEntryCode 为空挂应用根级，非空须为同应用下未软删的分组节点，
	// 否则返回 APP_MENU_PARENT_INVALID（BizError 透传出网）
	AttachFormEntry(ctx context.Context, applicationID, formID uint, name, parentEntryCode string) error
	// SyncFormEntryName 表单改名事务内同步节点展示名
	SyncFormEntryName(ctx context.Context, applicationID, formID uint, name string) error
	// DetachFormEntry 表单删除事务内软删节点
	DetachFormEntry(ctx context.Context, applicationID, formID uint) error
}

// menuMaintenanceService 端口实现：每次维护同事务写节点并递增修订号。
type menuMaintenanceService struct {
	repo repository.MenuRepository
}

// NewMenuMaintenanceService 构造菜单维护端口实现（server 装配注入表单域）。
func NewMenuMaintenanceService(repo repository.MenuRepository) MenuMaintenance {
	return &menuMaintenanceService{repo: repo}
}

// AttachFormEntry 挂 form 资产节点：parentEntryCode 为空挂应用根级，非空
// 先定位分组节点并校验（存在 + 分组类型，跨应用/跨租户编码定位不到，
// 统一按 APP_MENU_PARENT_INVALID 拒绝），sortOrder 取同父最大值 + 1024。
func (s *menuMaintenanceService) AttachFormEntry(ctx context.Context, applicationID, formID uint, name, parentEntryCode string) error {
	var parentEntryID *uint
	if parentEntryCode != "" {
		parent, err := s.repo.FindByCode(ctx, applicationID, parentEntryCode)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return httpx.Wrap(apperrors.ErrMenuParentInvalid,
					fmt.Errorf("parent entry %q not found in application %d", parentEntryCode, applicationID))
			}
			return err
		}
		if parent.EntryType != model.MenuEntryTypeGroup {
			return httpx.Wrap(apperrors.ErrMenuParentInvalid,
				fmt.Errorf("parent entry %q is %s, not group", parentEntryCode, parent.EntryType))
		}
		parentEntryID = &parent.ID
	}

	sortOrder, err := s.repo.MaxSortOrder(ctx, applicationID, parentEntryID)
	if err != nil {
		return err
	}
	targetType := model.MenuEntryTypeForm
	entry := &model.MenuEntry{
		ApplicationID: applicationID,
		ParentEntryID: parentEntryID,
		EntryType:     model.MenuEntryTypeForm,
		Name:          name,
		TargetType:    &targetType,
		TargetID:      &formID,
		SortOrder:     sortOrder + 1024,
	}
	if _, err := s.repo.CreateFormEntry(ctx, entry); err != nil {
		return err
	}
	return s.repo.BumpMenuRevision(ctx, applicationID)
}

func (s *menuMaintenanceService) SyncFormEntryName(ctx context.Context, applicationID, formID uint, name string) error {
	if err := s.repo.UpdateNameByFormTarget(ctx, applicationID, formID, name); err != nil {
		return err
	}
	return s.repo.BumpMenuRevision(ctx, applicationID)
}

func (s *menuMaintenanceService) DetachFormEntry(ctx context.Context, applicationID, formID uint) error {
	if err := s.repo.SoftDeleteByFormTarget(ctx, applicationID, formID); err != nil {
		return err
	}
	return s.repo.BumpMenuRevision(ctx, applicationID)
}
