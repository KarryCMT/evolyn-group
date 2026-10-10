package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	apperrors "evolyn/internal/platform/app"
	"evolyn/internal/platform/app/model"
	"evolyn/internal/platform/app/repository"
	"evolyn/internal/platform/httpx"
)

// FormDirectory 表单目录窄端口（M2-资产-1，菜单读侧资产可见性与 target
// 投影）：由表单域仓储在装配层适配；app 域不反向依赖 form 域包。
type FormDirectory interface {
	// ExistingFormTargets 返回内部 ID 对应的菜单目标投影（租户过滤由
	// ctx 承载，跨租户 ID 自然不在结果中）。
	ExistingFormTargets(ctx context.Context, ids []uint) (map[uint]FormTargetProjection, error)
}

// FormTargetProjection 是表单域向菜单读模型提供的最小投影；不携带草稿、
// 发布版本等表单内部数据，避免 app 域反向依赖 form 域模型。
type FormTargetProjection struct {
	Code     string
	FormType string
}

// DashboardDirectory 仪表盘目录窄端口（仪表盘资产 Stage 2）：菜单读侧只
// 消费内部 ID 到公开编码的最小投影，使编辑、改名、移动、删除动作能够以
// 稳定 code 调用资产接口；发布可见性仍由后续 DashboardDirectory 扩展负责。
type DashboardDirectory interface {
	ExistingDashboardTargets(ctx context.Context, ids []uint) (map[uint]string, error)
}

// DashboardDirectoryInjector 菜单服务装配期注入能力（可选）。
type DashboardDirectoryInjector interface {
	UseDashboardDirectory(dir DashboardDirectory)
}

// FormPermissionDirectory 表单资产权限裁剪窄端口（表单权限 P1，S5/S8）：
// 由 form 域权限组判定器在装配层适配；app 域不反向依赖 form 域包。
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
	// AttachFormNode 表单创建事务内挂 form 资产节点（target_id 保留内部表单 ID，出网投影 code）；
	// parentMenuCode 为空挂应用根级，非空须为同应用下未软删的分组节点，
	// 否则返回 APP_MENU_PARENT_INVALID（BizError 透传出网）
	AttachFormNode(ctx context.Context, appID, formID uint, name, parentMenuCode string) error
	// SyncFormNodeName 表单改名事务内同步节点展示名
	SyncFormNodeName(ctx context.Context, appID, formID uint, name string) error
	// SyncFormNodeAppearance 表单图标/颜色修改事务内同步节点展示属性
	//（ADR-011：资产节点的展示属性以资产域为事实源；空串表示清空，
	// 出网投影为 null）
	SyncFormNodeAppearance(ctx context.Context, appID, formID uint, icon, color string) error
	// DetachFormNode 表单删除事务内软删节点
	DetachFormNode(ctx context.Context, appID, formID uint) error
}

// menuMaintenanceService 端口实现：每次维护同事务写节点并递增修订号。
type menuMaintenanceService struct {
	repo repository.MenuRepository
}

func (s *menuMaintenanceService) dashboardRepo() (repository.DashboardMenuRepository, error) {
	repo, ok := s.repo.(repository.DashboardMenuRepository)
	if !ok {
		return nil, fmt.Errorf("dashboard menu repository is not configured")
	}
	return repo, nil
}

// NewMenuMaintenanceService 构造菜单维护端口实现（server 装配注入表单域）。
func NewMenuMaintenanceService(repo repository.MenuRepository) *menuMaintenanceService {
	return &menuMaintenanceService{repo: repo}
}

// AttachFormNode 挂 form 资产节点：parentMenuCode 为空挂应用根级，非空
// 先定位分组节点并校验（存在 + 分组类型，跨应用/跨租户编码定位不到，
// 统一按 APP_MENU_PARENT_INVALID 拒绝），sortOrder 取同父最大值 + 1024。
func (s *menuMaintenanceService) AttachFormNode(ctx context.Context, appID, formID uint, name, parentMenuCode string) error {
	var parentMenuID *uint
	if parentMenuCode != "" {
		parent, err := s.repo.FindByCode(ctx, appID, parentMenuCode)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return httpx.Wrap(apperrors.ErrMenuParentInvalid,
					fmt.Errorf("parent node %q not found in app %d", parentMenuCode, appID))
			}
			return err
		}
		if parent.MenuType != model.MenuTypeGroup {
			return httpx.Wrap(apperrors.ErrMenuParentInvalid,
				fmt.Errorf("parent node %q is %s, not group", parentMenuCode, parent.MenuType))
		}
		parentMenuID = &parent.ID
	}

	sortOrder, err := s.repo.MaxSortOrder(ctx, appID, parentMenuID)
	if err != nil {
		return err
	}
	targetType := model.MenuTypeForm
	node := &model.MenuNode{
		AppID:        appID,
		ParentMenuID: parentMenuID,
		MenuType:     model.MenuTypeForm,
		Name:         name,
		TargetType:   &targetType,
		TargetID:     &formID,
		SortOrder:    sortOrder + 1024,
	}
	if _, err := s.repo.CreateFormNode(ctx, node); err != nil {
		return err
	}
	return s.repo.BumpMenuRevision(ctx, appID)
}

func (s *menuMaintenanceService) SyncFormNodeName(ctx context.Context, appID, formID uint, name string) error {
	if err := s.repo.UpdateNameByFormTarget(ctx, appID, formID, name); err != nil {
		return err
	}
	return s.repo.BumpMenuRevision(ctx, appID)
}

// SyncFormNodeAppearance 图标/颜色同步：展示属性变更递增修订号
// （节点出网视图随target投影变化）。
func (s *menuMaintenanceService) SyncFormNodeAppearance(ctx context.Context, appID, formID uint, icon, color string) error {
	if err := s.repo.UpdateAppearanceByFormTarget(ctx, appID, formID, icon, color); err != nil {
		return err
	}
	return s.repo.BumpMenuRevision(ctx, appID)
}

func (s *menuMaintenanceService) DetachFormNode(ctx context.Context, appID, formID uint) error {
	// 先清理关联收藏行（ADR-011：个人状态不指向软删节点），再软删节点
	if err := s.repo.DeleteFavoritesByFormTarget(ctx, appID, formID); err != nil {
		return err
	}
	if err := s.repo.SoftDeleteByFormTarget(ctx, appID, formID); err != nil {
		return err
	}
	return s.repo.BumpMenuRevision(ctx, appID)
}

// AttachDashboardNode 在仪表盘预创建事务内挂载节点；父分组校验、排序和
// revision 推进与表单节点同口径。
func (s *menuMaintenanceService) AttachDashboardNode(ctx context.Context, appID, dashboardID uint, name, icon, color, parentMenuCode string) error {
	dashboardRepo, err := s.dashboardRepo()
	if err != nil {
		return err
	}
	parentMenuID, err := s.resolveDashboardParent(ctx, appID, parentMenuCode)
	if err != nil {
		return err
	}
	sortOrder, err := s.repo.MaxSortOrder(ctx, appID, parentMenuID)
	if err != nil {
		return err
	}
	targetType := model.MenuTypeDashboard
	_, err = dashboardRepo.CreateDashboardNode(ctx, &model.MenuNode{
		AppID: appID, ParentMenuID: parentMenuID, MenuType: model.MenuTypeDashboard,
		Name: name, Icon: icon, Color: color, TargetType: &targetType,
		TargetID: &dashboardID, SortOrder: sortOrder + 1024,
	})
	if err != nil {
		return err
	}
	return s.repo.BumpMenuRevision(ctx, appID)
}

// AttachDashboardCopyNode 将副本放在源仪表盘所在分组的末尾。父节点从服务端
// 菜单事实源读取，避免客户端提交过期或越权的父分组编码。
func (s *menuMaintenanceService) AttachDashboardCopyNode(ctx context.Context, appID, sourceDashboardID, dashboardID uint, name, icon, color string) error {
	dashboardRepo, err := s.dashboardRepo()
	if err != nil {
		return err
	}
	source, err := dashboardRepo.FindByAssetTarget(ctx, appID, model.MenuTypeDashboard, sourceDashboardID)
	if err != nil {
		return err
	}
	sortOrder, err := s.repo.MaxSortOrder(ctx, appID, source.ParentMenuID)
	if err != nil {
		return err
	}
	targetType := model.MenuTypeDashboard
	if _, err := dashboardRepo.CreateDashboardNode(ctx, &model.MenuNode{
		AppID: appID, ParentMenuID: source.ParentMenuID, MenuType: model.MenuTypeDashboard,
		Name: name, Icon: icon, Color: color, TargetType: &targetType,
		TargetID: &dashboardID, SortOrder: sortOrder + 1024,
	}); err != nil {
		return err
	}
	return s.repo.BumpMenuRevision(ctx, appID)
}

// SyncDashboardNode 把展示信息与可选移动合并为一次节点 UPDATE，并且无论组合
// 了多少字段都只推进一次 menu_revision。
func (s *menuMaintenanceService) SyncDashboardNode(ctx context.Context, appID, dashboardID uint, name, icon, color *string, parentMenuCode *string) error {
	dashboardRepo, err := s.dashboardRepo()
	if err != nil {
		return err
	}
	fields := map[string]interface{}{}
	if name != nil {
		fields["name"] = *name
	}
	if icon != nil {
		fields["icon"] = *icon
	}
	if color != nil {
		fields["color"] = *color
	}
	if parentMenuCode != nil {
		parentID, err := s.resolveDashboardParent(ctx, appID, *parentMenuCode)
		if err != nil {
			return err
		}
		sortOrder, err := s.repo.MaxSortOrder(ctx, appID, parentID)
		if err != nil {
			return err
		}
		fields["parent_menu_id"] = parentID
		fields["sort_order"] = sortOrder + 1024
	}
	if len(fields) == 0 {
		return nil
	}
	if _, err := dashboardRepo.FindByAssetTarget(ctx, appID, model.MenuTypeDashboard, dashboardID); err != nil {
		return err
	}
	if err := dashboardRepo.UpdateDashboardTargetFields(ctx, appID, dashboardID, fields); err != nil {
		return err
	}
	return s.repo.BumpMenuRevision(ctx, appID)
}

func (s *menuMaintenanceService) DetachDashboardNode(ctx context.Context, appID, dashboardID uint) error {
	dashboardRepo, err := s.dashboardRepo()
	if err != nil {
		return err
	}
	if err := dashboardRepo.DeleteFavoritesByDashboardTarget(ctx, appID, dashboardID); err != nil {
		return err
	}
	if err := dashboardRepo.SoftDeleteByDashboardTarget(ctx, appID, dashboardID); err != nil {
		return err
	}
	return s.repo.BumpMenuRevision(ctx, appID)
}

func (s *menuMaintenanceService) resolveDashboardParent(ctx context.Context, appID uint, parentMenuCode string) (*uint, error) {
	parentMenuCode = strings.TrimSpace(parentMenuCode)
	if parentMenuCode == "" {
		return nil, nil
	}
	parent, err := s.repo.FindByCode(ctx, appID, parentMenuCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.Wrap(apperrors.ErrMenuParentInvalid, fmt.Errorf("parent node %q not found in app %d", parentMenuCode, appID))
		}
		return nil, err
	}
	if parent.MenuType != model.MenuTypeGroup {
		return nil, httpx.Wrap(apperrors.ErrMenuParentInvalid, fmt.Errorf("parent node %q is not group", parentMenuCode))
	}
	return &parent.ID, nil
}
