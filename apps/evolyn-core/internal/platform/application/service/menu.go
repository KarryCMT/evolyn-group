package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"evolyn/internal/contextx"
	apperrors "evolyn/internal/platform/application"
	"evolyn/internal/platform/application/model"
	"evolyn/internal/platform/application/repository"
	auditservice "evolyn/internal/platform/audit/service"
	"evolyn/internal/platform/httpx"
	"evolyn/internal/platform/iam/authorization"
	iammodel "evolyn/internal/platform/iam/model"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// menuService 应用菜单服务：读取快照 → 树完整性校验 → 可见性裁剪 →
// capabilities 派生；分组创建通过统一事务和 menuRevision 条件更新串行化。
type menuService struct {
	tx      TxManager
	repo    repository.MenuRepository
	audit   auditservice.Recorder
	access  ApplicationAccessEvaluator
	formDir FormDirectory
}

// NewMenuService 构造菜单服务；访问判定与鉴权中间件同源（复用应用域
// ApplicationAccessEvaluator，§6.1：Service 复核不能仅依赖中间件）
func NewMenuService(tx TxManager, repo repository.MenuRepository, audit auditservice.Recorder, access ApplicationAccessEvaluator) ApplicationMenuService {
	return &menuService{tx: tx, repo: repo, audit: audit, access: access}
}

// UseFormDirectory 注入表单目录窄端口（装配期一次性调用，M2-资产-1）：
// 菜单读侧据此裁剪已软删表单节点并投影 target 公开编码；未注入时保持
// 旧行为（节点按存在性出网、target 不投影），存量测试桩无需调整。
func (s *menuService) UseFormDirectory(dir FormDirectory) {
	s.formDir = dir
}

// MenuFormDirectoryInjector 装配期注入能力（可选）。
type MenuFormDirectoryInjector interface {
	UseFormDirectory(dir FormDirectory)
}

// GetMenu 按应用编码读取当前成员可见的菜单快照（方案 §6）：
// 无读取权限与应用不存在统一 APP_NOT_FOUND，避免泄露应用存在性；
// 返回空树（rootEntryIds 空）是合法结果，触发前端空应用引导
func (s *menuService) GetMenu(ctx context.Context, member *iammodel.User, code string) (*model.MenuSnapshot, error) {
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("tenant context required")
	}

	// 与路由层 GET /applications/code/:code/menu（verb=get）同口径复核；
	// 无 applications:get 时按「应用不存在」出网（§6.1 统一口径），
	// 细节经 Wrap 只入日志
	perms := s.access.Permissions(ctx, member)
	if !perms["applications:get"] {
		return nil, httpx.Wrap(apperrors.ErrNotFound,
			fmt.Errorf("member %d (tenant %d) cannot read menu of application %s", memberID(member), tenantID, code))
	}

	snap, err := s.repo.GetSnapshot(ctx, tenantID, code)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.Wrap(apperrors.ErrNotFound, err)
		}
		return nil, err
	}

	// 个人收藏状态（ADR-011）：读时并查当前成员在本应用内的收藏集合，
	// 仅用于投影 Favorited，不参与可见性与修订号
	favorites, err := s.repo.FavoriteEntryIDs(ctx, tenantID, member.ID, snap.ApplicationID)
	if err != nil {
		return nil, err
	}

	// M2-资产-1：表单目标投影批量查询。existingFormTargets 以 nil 表达「目录
	// 未接入」（旧行为：不裁剪、不投影）；目录接入且确无表单时为空 map，
	// form 节点按不存在裁剪——两种空态语义必须区分。
	var existingFormTargets map[uint]FormTargetProjection
	formIDs := make([]uint, 0)
	for i := range snap.Entries {
		if snap.Entries[i].EntryType == model.MenuEntryTypeForm && snap.Entries[i].TargetID != nil {
			formIDs = append(formIDs, *snap.Entries[i].TargetID)
		}
	}
	if s.formDir != nil && len(formIDs) > 0 {
		existingFormTargets, err = s.formDir.ExistingFormTargets(ctx, formIDs)
		if err != nil {
			return nil, err
		}
	}

	return buildMenuSnapshot(perms, snap, existingFormTargets, favorites)
}

// CreateGroup 创建根分组或二级子分组。条件递增修订号是事务内第一项菜单
// 写操作：它既拒绝陈旧客户端，也锁住应用行，使后续父节点与排序校验基于
// 稳定菜单版本；任一步失败都会连同修订号递增一起回滚。
func (s *menuService) CreateGroup(ctx context.Context, member *iammodel.User, code string, req *model.CreateMenuGroupRequest) (*model.MenuGroupMutation, error) {
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("tenant context required")
	}
	if req == nil {
		return nil, httpx.Wrap(apperrors.ErrMenuNameInvalid, errors.New("create menu group request is nil"))
	}
	if member == nil || member.ID == 0 || member.TenantID != tenantID {
		return nil, httpx.Wrap(apperrors.ErrMemberInvalid,
			fmt.Errorf("member %d (tenant %d) not in tenant %d", memberID(member), memberTenant(member), tenantID))
	}
	perms := s.access.Permissions(ctx, member)
	if !perms["applications:create"] {
		return nil, httpx.Wrap(apperrors.ErrForbidden,
			fmt.Errorf("member %d cannot create menu group in application %s", member.ID, code))
	}

	name := strings.TrimSpace(req.Name)
	if name == "" || utf8.RuneCountInString(name) > 128 {
		return nil, httpx.Wrap(apperrors.ErrMenuNameInvalid,
			fmt.Errorf("menu group name rune count %d", utf8.RuneCountInString(name)))
	}
	var parentCode *string
	if req.ParentEntryID != nil {
		trimmed := strings.TrimSpace(*req.ParentEntryID)
		if trimmed == "" {
			return nil, httpx.Wrap(apperrors.ErrMenuParentInvalid, errors.New("parent entry id is blank"))
		}
		parentCode = &trimmed
	}

	var created *model.MenuEntry
	if s.tx == nil {
		return nil, errors.New("application menu transaction manager is required")
	}
	err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		snap, err := s.repo.GetSnapshot(tctx, tenantID, code)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return httpx.Wrap(apperrors.ErrNotFound, err)
			}
			return err
		}
		if provisionStatus(snap.ProvisionStatus) {
			return httpx.Wrap(apperrors.ErrProvisioning,
				fmt.Errorf("application %d provisioning (%s)", snap.ApplicationID, snap.ProvisionStatus))
		}
		if snap.Status != model.ApplicationStatusActive {
			return httpx.Wrap(apperrors.ErrStatusInvalid,
				fmt.Errorf("application %d status %s", snap.ApplicationID, snap.Status))
		}

		advanced, err := s.repo.BumpMenuRevisionFrom(tctx, snap.ApplicationID, req.BaseMenuRevision)
		if err != nil {
			return err
		}
		if !advanced {
			return httpx.Wrap(apperrors.ErrMenuVersionConflict,
				fmt.Errorf("application %d menu revision changed from base %d", snap.ApplicationID, req.BaseMenuRevision))
		}

		var parentEntryID *uint
		if parentCode != nil {
			parent, err := menuEntryByCode(snap.Entries, *parentCode)
			if err != nil || parent.EntryType != model.MenuEntryTypeGroup {
				return httpx.Wrap(apperrors.ErrMenuParentInvalid,
					fmt.Errorf("parent entry %q is missing or not a group", *parentCode))
			}
			// 根分组为第一级；其子分组为第二级。第二级下不再允许创建分组。
			if parent.ParentEntryID != nil {
				return httpx.Wrap(apperrors.ErrMenuDepthExceeded,
					fmt.Errorf("parent entry %q is already a nested group", *parentCode))
			}
			parentEntryID = &parent.ID
		}

		sortOrder, err := s.repo.MaxSortOrder(tctx, snap.ApplicationID, parentEntryID)
		if err != nil {
			return err
		}
		created, err = s.repo.CreateGroupEntry(tctx, &model.MenuEntry{
			TenantID:      tenantID,
			ApplicationID: snap.ApplicationID,
			ParentEntryID: parentEntryID,
			EntryType:     model.MenuEntryTypeGroup,
			Name:          name,
			SortOrder:     sortOrder + 1024,
		})
		return err
	})
	if err != nil {
		return nil, err
	}

	result := &model.MenuGroupMutation{
		EntryID:       created.Code,
		ParentEntryID: parentCode,
		Name:          name,
		MenuRevision:  req.BaseMenuRevision + 1,
	}
	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "application", Action: "create", ResourceType: "application_menu_entry",
			ResourceID: created.Code,
			After: map[string]any{
				"applicationCode": code,
				"entryCode":       created.Code,
				"entryType":       model.MenuEntryTypeGroup,
				"parentEntryId":   parentCode,
				"name":            name,
			},
		})
	}
	return result, nil
}

// menuEntryByCode 在已与 menuRevision 同快照读取的节点集合中定位父节点，
// 避免条件递增前后再发起一次可能读到不同版本的查询。
func menuEntryByCode(entries []model.MenuEntry, code string) (*model.MenuEntry, error) {
	for i := range entries {
		if entries[i].Code == code {
			return &entries[i], nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

// UpdateEntry 菜单节点管理更新（ADR-011）：分组改名 / 资产节点对成员隐藏 /
// 移动节点，统一走 menuRevision 乐观锁串行化。资产节点名称以资产域为事实
// 源，本接口拒绝改名（经对应资产接口修改后同事务同步回节点）。
func (s *menuService) UpdateEntry(ctx context.Context, member *iammodel.User, code, entryCode string, req *model.UpdateMenuEntryRequest) (*model.MenuEntryMutation, error) {
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("tenant context required")
	}
	if req == nil {
		return nil, httpx.Wrap(apperrors.ErrMenuNameInvalid, errors.New("update menu entry request is nil"))
	}
	if member == nil || member.ID == 0 || member.TenantID != tenantID {
		return nil, httpx.Wrap(apperrors.ErrMemberInvalid,
			fmt.Errorf("member %d (tenant %d) not in tenant %d", memberID(member), memberTenant(member), tenantID))
	}
	// 与路由层 PATCH /applications/...（verb=patch）同口径复核
	perms := s.access.Permissions(ctx, member)
	if !perms["applications:patch"] {
		return nil, httpx.Wrap(apperrors.ErrForbidden,
			fmt.Errorf("member %d cannot update menu entry in application %s", member.ID, code))
	}
	// 隐藏开关是独立动作授权键（form-actions:hide），不随菜单管理权限放大
	if req.Hidden != nil && !perms["form-actions:hide"] {
		return nil, httpx.Wrap(apperrors.ErrForbidden,
			fmt.Errorf("member %d lacks form-actions:hide", member.ID))
	}

	// 改名校验（仅分组；资产节点改名拒绝）
	var newName *string
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" || utf8.RuneCountInString(name) > 128 {
			return nil, httpx.Wrap(apperrors.ErrMenuNameInvalid,
				fmt.Errorf("menu entry name rune count %d", utf8.RuneCountInString(name)))
		}
		newName = &name
	}
	// 移动目标：nil 指针=未提交移动；非 nil 空串=移动到根级；非空=分组编码
	var newParentCode *string
	if req.ParentEntryCode != nil {
		trimmed := strings.TrimSpace(*req.ParentEntryCode)
		newParentCode = &trimmed
	}
	if newName == nil && req.Hidden == nil && newParentCode == nil {
		return nil, httpx.Wrap(apperrors.ErrMenuNameInvalid, errors.New("no updatable field provided"))
	}

	var updated *model.MenuEntry
	var newRevision int64
	if s.tx == nil {
		return nil, errors.New("application menu transaction manager is required")
	}
	err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		snap, err := s.repo.GetSnapshot(tctx, tenantID, code)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return httpx.Wrap(apperrors.ErrNotFound, err)
			}
			return err
		}
		if provisionStatus(snap.ProvisionStatus) {
			return httpx.Wrap(apperrors.ErrProvisioning,
				fmt.Errorf("application %d provisioning (%s)", snap.ApplicationID, snap.ProvisionStatus))
		}
		if snap.Status != model.ApplicationStatusActive {
			return httpx.Wrap(apperrors.ErrStatusInvalid,
				fmt.Errorf("application %d status %s", snap.ApplicationID, snap.Status))
		}

		advanced, err := s.repo.BumpMenuRevisionFrom(tctx, snap.ApplicationID, req.BaseMenuRevision)
		if err != nil {
			return err
		}
		if !advanced {
			return httpx.Wrap(apperrors.ErrMenuVersionConflict,
				fmt.Errorf("application %d menu revision changed from base %d", snap.ApplicationID, req.BaseMenuRevision))
		}

		entry, err := menuEntryByCode(snap.Entries, entryCode)
		if err != nil {
			return httpx.Wrap(apperrors.ErrMenuNotFound,
				fmt.Errorf("entry %q not found in application %s", entryCode, code))
		}

		fields := make(map[string]interface{}, 3)
		if newName != nil {
			// 资产节点的展示名由资产域事实源管理，拒绝旁路改名
			if entry.EntryType != model.MenuEntryTypeGroup {
				return httpx.Wrap(apperrors.ErrMenuEntryRenameForbidden,
					fmt.Errorf("entry %q is an asset node of type %s", entryCode, entry.EntryType))
			}
			fields["name"] = *newName
		}
		if req.Hidden != nil {
			// 分组可见性由后代派生，「对成员隐藏」仅对资产节点成立
			if entry.EntryType == model.MenuEntryTypeGroup {
				return httpx.Wrap(apperrors.ErrMenuHiddenInvalid,
					fmt.Errorf("entry %q is a group node", entryCode))
			}
			fields["hidden"] = *req.Hidden
		}
		if newParentCode != nil {
			parentEntryID, err := resolveMoveParent(snap.Entries, entry, *newParentCode)
			if err != nil {
				return err
			}
			sortOrder, err := s.repo.MaxSortOrder(tctx, snap.ApplicationID, parentEntryID)
			if err != nil {
				return err
			}
			fields["parent_entry_id"] = parentEntryID
			// 移动节点追加到目标父节点末位（服务端排序，不信任客户端排序值）
			fields["sort_order"] = sortOrder + 1024
		}
		if err := s.repo.UpdateEntryFields(tctx, snap.ApplicationID, entry.ID, fields); err != nil {
			return err
		}
		updated = entry
		newRevision = req.BaseMenuRevision + 1
		return nil
	})
	if err != nil {
		return nil, err
	}

	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "application", Action: "update", ResourceType: "application_menu_entry",
			ResourceID: updated.Code,
			After: map[string]any{
				"applicationCode": code,
				"entryCode":       updated.Code,
				"name":            newName,
				"hidden":          req.Hidden,
				"parentEntryCode": newParentCode,
			},
		})
	}
	return &model.MenuEntryMutation{EntryID: updated.Code, MenuRevision: newRevision}, nil
}

// resolveMoveParent 校验移动目标父节点：空串移动到根级；非空须为同应用
// 未软删分组，且不是被移动节点自身或其后代（防自环），分组移动须落到根
// 分组下（层级上限与创建分组一致：根分组+子分组两层）。
func resolveMoveParent(entries []model.MenuEntry, entry *model.MenuEntry, parentCode string) (*uint, error) {
	if parentCode == "" {
		return nil, nil // 根级
	}
	parent, err := menuEntryByCode(entries, parentCode)
	if err != nil || parent.EntryType != model.MenuEntryTypeGroup {
		return nil, httpx.Wrap(apperrors.ErrMenuParentInvalid,
			fmt.Errorf("move parent %q is missing or not a group", parentCode))
	}
	if parent.ID == entry.ID {
		return nil, httpx.Wrap(apperrors.ErrMenuMoveInvalid,
			fmt.Errorf("cannot move entry %q under itself", entry.Code))
	}
	// 沿父链上行：目标父节点处于被移动节点子树内即形成环
	byID := make(map[uint]*model.MenuEntry, len(entries))
	for i := range entries {
		byID[entries[i].ID] = &entries[i]
	}
	for cur := parent; cur.ParentEntryID != nil; {
		cur = byID[*cur.ParentEntryID]
		if cur.ID == entry.ID {
			return nil, httpx.Wrap(apperrors.ErrMenuMoveInvalid,
				fmt.Errorf("cannot move entry %q under its own descendant %q", entry.Code, parentCode))
		}
	}
	// 分组挂载深度限制：目标父分组必须为根分组（移动后仍是两级结构）
	if entry.EntryType == model.MenuEntryTypeGroup && parent.ParentEntryID != nil {
		return nil, httpx.Wrap(apperrors.ErrMenuDepthExceeded,
			fmt.Errorf("group %q cannot move under nested group %q", entry.Code, parentCode))
	}
	return &parent.ID, nil
}

// AddFavorite 收藏菜单节点（个人状态动作，ADR-011）：凡能读取应用菜单的
// 成员即可收藏（menu-favorites:create 授全体成员）；重复收藏幂等成功。
// 不递增菜单修订号——收藏不改变共享菜单结构。
func (s *menuService) AddFavorite(ctx context.Context, member *iammodel.User, appCode, entryCode string) (*model.MenuFavoriteMutation, error) {
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("tenant context required")
	}
	if member == nil || member.ID == 0 || member.TenantID != tenantID {
		return nil, httpx.Wrap(apperrors.ErrMemberInvalid,
			fmt.Errorf("member %d (tenant %d) not in tenant %d", memberID(member), memberTenant(member), tenantID))
	}
	snap, err := s.repo.GetSnapshot(ctx, tenantID, appCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.Wrap(apperrors.ErrNotFound, err)
		}
		return nil, err
	}
	entry, err := menuEntryByCode(snap.Entries, entryCode)
	if err != nil {
		return nil, httpx.Wrap(apperrors.ErrMenuFavoriteInvalid,
			fmt.Errorf("entry %q not found in application %s", entryCode, appCode))
	}
	if err := s.repo.CreateFavorite(ctx, &model.MenuEntryFavorite{
		TenantID:      tenantID,
		MemberID:      member.ID,
		ApplicationID: snap.ApplicationID,
		EntryID:       entry.ID,
	}); err != nil {
		return nil, err
	}
	return &model.MenuFavoriteMutation{EntryID: entry.Code, Favorited: true}, nil
}

// RemoveFavorite 取消收藏（幂等）：目标收藏不存在同样返回 Favorited=false。
func (s *menuService) RemoveFavorite(ctx context.Context, member *iammodel.User, entryCode string) (*model.MenuFavoriteMutation, error) {
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("tenant context required")
	}
	if member == nil || member.ID == 0 || member.TenantID != tenantID {
		return nil, httpx.Wrap(apperrors.ErrMemberInvalid,
			fmt.Errorf("member %d (tenant %d) not in tenant %d", memberID(member), memberTenant(member), tenantID))
	}
	if _, err := s.repo.DeleteFavoriteByCode(ctx, tenantID, member.ID, entryCode); err != nil {
		return nil, err
	}
	return &model.MenuFavoriteMutation{EntryID: entryCode, Favorited: false}, nil
}

// assetVisible 资产节点可见性判定（M2-资产-1 起按目录端口注入的存在集
// 执行）：form 节点在表单软删/不存在时不可见；目录未接入（existingFormTargets
// 为 nil）时保持旧行为——按节点存在性出网。仪表盘/页面域落地后在此扩展。
var assetVisible = func(entry *model.MenuEntry, existingFormTargets map[uint]FormTargetProjection) bool {
	switch entry.EntryType {
	case model.MenuEntryTypeForm:
		if existingFormTargets == nil {
			return true
		}
		if entry.TargetID == nil {
			return false
		}
		target, ok := existingFormTargets[*entry.TargetID]
		return ok && target.Code != "" && target.FormType != ""
	default:
		return true
	}
}

// buildMenuSnapshot 由仓储快照组装出网视图：结构完整性校验 → 可见性
// 裁剪 → 排序投影。损坏树（孤儿/父非分组/循环）记录告警并返回
// ErrMenuInvalid，不把异常数据当正常树交给前端（方案 §6.3）。
// ADR-011：可见性叠加「对成员隐藏」裁剪；capabilities 携带按钮级 actions
// （动作注册表 × 权限集 × 应用状态派生）与当前成员收藏状态。
func buildMenuSnapshot(perms map[string]bool, snap *repository.MenuSnapshot, existingFormTargets map[uint]FormTargetProjection, favorites map[uint]bool) (*model.MenuSnapshot, error) {
	byID := make(map[uint]*model.MenuEntry, len(snap.Entries))
	for i := range snap.Entries {
		byID[snap.Entries[i].ID] = &snap.Entries[i]
	}

	// 结构校验（在完整节点集上进行，先于可见性裁剪）：父引用存在且为
	// 分组、沿父链无循环。三者均为服务端数据完整性故障（写入路径被
	// Service 层校验拦截，正常不会出现）
	for i := range snap.Entries {
		entry := &snap.Entries[i]
		if err := validateMenuAncestry(byID, entry); err != nil {
			logrus.WithFields(logrus.Fields{
				"application": snap.ApplicationCode,
				"entry":       entry.Code,
			}).Warnf("application menu integrity check failed: %v", err)
			return nil, httpx.Wrap(apperrors.ErrMenuInvalid, err)
		}
	}

	// 可见性裁剪：资产节点按 assetVisible 判定，并叠加「对成员隐藏」
	// （ADR-011，导航隐藏）——hidden 节点仅对持菜单管理权限的成员可见
	//（否则无法恢复显示）；可管理菜单的成员还需看到刚创建的空分组才能
	// 继续向其中添加资产，只读成员仍只看到有可见后代的分组，避免泄露
	// 无内容的管理结构。
	canManageMenu := perms["applications:create"] || perms["applications:patch"]
	nodeVisible := func(entry *model.MenuEntry) bool {
		if !assetVisible(entry, existingFormTargets) {
			return false
		}
		return !entry.Hidden || canManageMenu
	}
	visible := make(map[uint]bool, len(snap.Entries))
	groupMemo := make(map[uint]bool, len(snap.Entries))
	for i := range snap.Entries {
		entry := &snap.Entries[i]
		switch entry.EntryType {
		case model.MenuEntryTypeGroup:
			visible[entry.ID] = canManageMenu || hasVisibleDescendant(byID, entry, groupMemo, nodeVisible)
		default:
			visible[entry.ID] = nodeVisible(entry)
		}
	}

	out := &model.MenuSnapshot{
		ApplicationCode: snap.ApplicationCode,
		MenuRevision:    snap.MenuRevision,
		RootEntryIDs:    make([]string, 0),
		EntryMap:        make(map[string]model.MenuEntryDetail, len(visible)),
		Features:        model.MenuFeatures{Workflow: false}, // 流程引擎未接入，能力注册表落地前恒 false
	}

	// 应用可编辑状态是全部按钮动作的公共因子（归档/初始化中应用只读）；
	// manage/move 口径对齐应用域 detailFor（认 applications:patch），delete
	// 认 applications:delete；favorite 凡可见即可收藏（个人状态动作）
	editable := snap.Status == model.ApplicationStatusActive && !provisionStatus(snap.ProvisionStatus)

	roots := make([]*model.MenuEntry, 0)
	for i := range snap.Entries {
		entry := &snap.Entries[i]
		if !visible[entry.ID] {
			continue
		}
		caps := model.MenuEntryCapabilities{
			View:     true,
			Manage:   perms["applications:patch"] && editable,
			Move:     perms["applications:patch"] && editable,
			Delete:   perms["applications:delete"] && editable,
			Favorite: true,
			Actions:  menuEntryActions(perms, entry, editable),
		}
		detail := menuEntryDetail(entry, byID, caps, existingFormTargets)
		detail.Favorited = favorites[entry.ID]
		out.EntryMap[entry.Code] = detail
		if entry.ParentEntryID == nil {
			roots = append(roots, entry)
		}
	}

	// 根顺序契约：sortOrder ASC, entryId ASC（方案 §6.2；仓储快照已按
	// (sort_order, code) 有序返回，此处排序为投影层兜底，二者同键）
	sort.Slice(roots, func(a, b int) bool {
		if roots[a].SortOrder != roots[b].SortOrder {
			return roots[a].SortOrder < roots[b].SortOrder
		}
		return roots[a].Code < roots[b].Code
	})
	for _, root := range roots {
		out.RootEntryIDs = append(out.RootEntryIDs, root.Code)
	}
	return out, nil
}

// menuEntryActions 节点按钮能力投影（ADR-011）：按节点类型查动作注册表，
// 以「授权键命中 × 应用可编辑」求值后映射为出网结构。注册表未登记的
// 动作（如分组上的表单专属动作）保持零值 false。
func menuEntryActions(perms map[string]bool, entry *model.MenuEntry, editable bool) model.MenuEntryActions {
	granted := authorization.MenuActionsOf(perms, entry.EntryType)
	actions := model.MenuEntryActions{}
	if editable {
		actions.Edit = granted[authorization.MenuActionEdit]
		actions.Rename = granted[authorization.MenuActionRename]
		actions.SwitchType = granted[authorization.MenuActionSwitchType]
		actions.ReferenceView = granted[authorization.MenuActionReferenceView]
		actions.CopyInApp = granted[authorization.MenuActionCopyInApp]
		actions.CopyCrossApp = granted[authorization.MenuActionCopyCrossApp]
		actions.Move = granted[authorization.MenuActionMove]
		actions.Hide = granted[authorization.MenuActionHide]
		actions.Delete = granted[authorization.MenuActionDelete]
	}
	return actions
}

// validateMenuAncestry 校验单节点父链完整性：父引用在集合内、父为分组、
// 沿链可达根（无循环）。返回的错误仅入日志（Wrap 进 ErrMenuInvalid）
func validateMenuAncestry(byID map[uint]*model.MenuEntry, entry *model.MenuEntry) error {
	if entry.ParentEntryID == nil {
		return nil
	}
	parent, ok := byID[*entry.ParentEntryID]
	if !ok {
		return fmt.Errorf("entry %d references missing parent %d", entry.ID, *entry.ParentEntryID)
	}
	if parent.EntryType != model.MenuEntryTypeGroup {
		return fmt.Errorf("entry %d has non-group parent %d", entry.ID, parent.ID)
	}
	// 沿父链上行：步数超过节点总数仍未到根即存在循环
	seen := 0
	for cur := entry; cur.ParentEntryID != nil; {
		cur = byID[*cur.ParentEntryID]
		seen++
		if seen > len(byID) {
			return fmt.Errorf("cycle detected via entry %d", entry.ID)
		}
	}
	return nil
}

// hasVisibleDescendant 自底向上裁剪：分组可见当且仅当存在可见后代节点
// （可见资产，或自身有可见后代的子分组）；空分组被裁剪（方案目标 3）。
// memo 缓存已判定分组避免链形树重复递归；树已被 validateMenuAncestry
// 保证无环，递归必然终止。visible 为可见性谓词（含隐藏裁剪，ADR-011）。
func hasVisibleDescendant(byID map[uint]*model.MenuEntry, group *model.MenuEntry, memo map[uint]bool, visible func(*model.MenuEntry) bool) bool {
	if cached, ok := memo[group.ID]; ok {
		return cached
	}
	// 先写 false 防御：无环前提下不会回读自身，仅为内存安全兜底
	memo[group.ID] = false
	keep := false
	for _, entry := range byID {
		if entry.ParentEntryID == nil || *entry.ParentEntryID != group.ID {
			continue
		}
		if entry.EntryType != model.MenuEntryTypeGroup {
			if visible(entry) {
				keep = true
				break
			}
			continue
		}
		if hasVisibleDescendant(byID, entry, memo, visible) {
			keep = true
			break
		}
	}
	memo[group.ID] = keep
	return keep
}

// menuEntryDetail 节点出网投影：icon/color 空串投影为 null；parentEntryId
// 由 byID 反查父节点编码（null 即根节点）。M2-资产-1 起 form 节点按目录
// 存在集投影 target（表单公开编码为 form_ 前缀稳定编码，formType 为表单域事实）；
// 目录未接入（existingFormTargets nil）时 target 保持不投影。仪表盘/页面域随
// 各自资产批次扩展
func menuEntryDetail(entry *model.MenuEntry, byID map[uint]*model.MenuEntry, caps model.MenuEntryCapabilities, existingFormTargets map[uint]FormTargetProjection) model.MenuEntryDetail {
	detail := model.MenuEntryDetail{
		EntryID:      entry.Code,
		Type:         entry.EntryType,
		Name:         entry.Name,
		SortOrder:    entry.SortOrder,
		Capabilities: caps,
	}
	if entry.ParentEntryID != nil {
		if parent, ok := byID[*entry.ParentEntryID]; ok {
			parentCode := parent.Code
			detail.ParentEntryID = &parentCode
		}
	}
	if entry.Icon != "" {
		detail.Icon = &entry.Icon
	}
	if entry.Color != "" {
		detail.Color = &entry.Color
	}
	// 目录接入后才投影（existingFormTargets 非 nil 且表单存在）；未接入时保持
	// 旧行为（target 出网 null），与裁剪钩子的 nil 语义一致。
	if entry.EntryType == model.MenuEntryTypeForm && entry.TargetID != nil && existingFormTargets != nil {
		target, ok := existingFormTargets[*entry.TargetID]
		if !ok || target.Code == "" || target.FormType == "" {
			return detail
		}
		detail.Target = &model.MenuEntryTarget{
			Type:     model.MenuEntryTypeForm,
			Code:     target.Code,
			FormType: target.FormType,
		}
	}
	return detail
}

// provisionStatus 与应用域 isProvisioning 同口径（pending/running 进行中）；
// 菜单服务不依赖 applicationService 实例，独立成函数避免循环依赖
func provisionStatus(status string) bool {
	return status == model.ProvisionStatusPending || status == model.ProvisionStatusRunning
}
