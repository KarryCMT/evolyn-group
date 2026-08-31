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
	// ExistingFormTargets 返回内部 ID 对应的菜单目标投影（租户过滤由
	// ctx 承载，跨租户 ID 自然不在结果中）。
	ExistingFormTargets(ctx context.Context, ids []uint) (map[uint]FormTargetProjection, error)
}

// FormTargetProjection 是表单域向菜单读模型提供的最小投影；不携带草稿、
// 发布版本等表单内部数据，避免 application 域反向依赖 form 域模型。
type FormTargetProjection struct {
	Code     string
	FormType string
}

// FormPermissionDirectory 表单资产权限裁剪窄端口（表单权限 P1，S5/S8）：
// 由 form 域权限组判定器在装配层适配；application 域不反向依赖 form 域包。
// 入口判定 = view ∨ add（仅录入表单对仅 add 成员可见）；无任何命中（含
// 禁用组收口）的表单节点在成员侧隐藏，空分组随既有 hasVisibleDescendant 裁剪。
type FormPermissionDirectory interface {
	// VisibleFormIDs 返回 ids 中当前成员可入口（view ∨ add）的表单 ID 集；
	// 未命中的表单 ID 不在结果中。管理员旁路（form-data:admin）由端口实现
	// 内部判定（全量可见）。
	VisibleFormIDs(ctx context.Context, memberID uint, formIDs []uint) (map[uint]bool, error)
}

// FormPermissionDirectoryInjector 菜单服务装配期注入能力（可选）。
type FormPermissionDirectoryInjector interface {
	UseFormPermissionDirectory(dir FormPermissionDirectory)
}

// MenuMaintenance 表单资产菜单节点维护窄端口（M2-资产-1）：表单域在创建/
// 改名/删除的事务内调用，节点写入与 menu_revision 递增随之加入同一事务；
// 菜单管理写接口（分组/移动/重排）仍随 M2-菜单-3 落地，本端口只承载
// 资产生命周期驱动的节点维护。
type MenuMaintenance interface {
	// AttachFormEntry 表单创建事务内挂 form 资产节点（target_id 保留内部表单 ID，出网投影 code）；
	// parentEntryCode 为空挂应用根级，非空须为同应用下未软删的分组节点，
	// 否则返回 APP_MENU_PARENT_INVALID（BizError 透传出网）
	AttachFormEntry(ctx context.Context, applicationID, formID uint, name, parentEntryCode string) error
	// SyncFormEntryName 表单改名事务内同步节点展示名
	SyncFormEntryName(ctx context.Context, applicationID, formID uint, name string) error
	// SyncFormEntryAppearance 表单图标/颜色修改事务内同步节点展示属性
	//（ADR-011：资产节点的展示属性以资产域为事实源；空串表示清空，
	// 出网投影为 null）
	SyncFormEntryAppearance(ctx context.Context, applicationID, formID uint, icon, color string) error
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

// SyncFormEntryAppearance 图标/颜色同步：展示属性变更递增修订号
// （节点出网视图随target投影变化）。
func (s *menuMaintenanceService) SyncFormEntryAppearance(ctx context.Context, applicationID, formID uint, icon, color string) error {
	if err := s.repo.UpdateAppearanceByFormTarget(ctx, applicationID, formID, icon, color); err != nil {
		return err
	}
	return s.repo.BumpMenuRevision(ctx, applicationID)
}

func (s *menuMaintenanceService) DetachFormEntry(ctx context.Context, applicationID, formID uint) error {
	// 先清理关联收藏行（ADR-011：个人状态不指向软删节点），再软删节点
	if err := s.repo.DeleteFavoritesByFormTarget(ctx, applicationID, formID); err != nil {
		return err
	}
	if err := s.repo.SoftDeleteByFormTarget(ctx, applicationID, formID); err != nil {
		return err
	}
	return s.repo.BumpMenuRevision(ctx, applicationID)
}
