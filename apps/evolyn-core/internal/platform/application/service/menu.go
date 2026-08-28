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

	return buildMenuSnapshot(perms, snap, existingFormTargets)
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
// ErrMenuInvalid，不把异常数据当正常树交给前端（方案 §6.3）
func buildMenuSnapshot(perms map[string]bool, snap *repository.MenuSnapshot, existingFormTargets map[uint]FormTargetProjection) (*model.MenuSnapshot, error) {
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

	// 可见性裁剪：资产节点按 assetVisible 判定；可管理菜单的成员还需看到
	// 刚创建的空分组才能继续向其中添加资产，只读成员仍只看到有可见后代
	// 的分组，避免泄露无内容的管理结构。
	canManageMenu := perms["applications:create"] || perms["applications:patch"]
	visible := make(map[uint]bool, len(snap.Entries))
	groupMemo := make(map[uint]bool, len(snap.Entries))
	for i := range snap.Entries {
		entry := &snap.Entries[i]
		switch entry.EntryType {
		case model.MenuEntryTypeGroup:
			visible[entry.ID] = canManageMenu || hasVisibleDescendant(byID, entry, groupMemo, existingFormTargets)
		default:
			visible[entry.ID] = assetVisible(entry, existingFormTargets)
		}
	}

	out := &model.MenuSnapshot{
		ApplicationCode: snap.ApplicationCode,
		MenuRevision:    snap.MenuRevision,
		RootEntryIDs:    make([]string, 0),
		EntryMap:        make(map[string]model.MenuEntryDetail, len(visible)),
		Features:        model.MenuFeatures{Workflow: false}, // 流程引擎未接入，能力注册表落地前恒 false
	}

	// capabilities 口径对齐应用域 detailFor：manage/move 只认 patch 且
	// 应用可编辑（active 且非初始化中），delete 认 delete 且非初始化中；
	// favorite 待个人收藏域落地，首期恒 false
	editable := snap.Status == model.ApplicationStatusActive && !provisionStatus(snap.ProvisionStatus)
	caps := model.MenuEntryCapabilities{
		View:     true,
		Manage:   perms["applications:patch"] && editable,
		Move:     perms["applications:patch"] && editable,
		Delete:   perms["applications:delete"] && editable,
		Favorite: false,
	}

	roots := make([]*model.MenuEntry, 0)
	for i := range snap.Entries {
		entry := &snap.Entries[i]
		if !visible[entry.ID] {
			continue
		}
		out.EntryMap[entry.Code] = menuEntryDetail(entry, byID, caps, existingFormTargets)
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
// 保证无环，递归必然终止
func hasVisibleDescendant(byID map[uint]*model.MenuEntry, group *model.MenuEntry, memo map[uint]bool, existingFormTargets map[uint]FormTargetProjection) bool {
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
			if assetVisible(entry, existingFormTargets) {
				keep = true
				break
			}
			continue
		}
		if hasVisibleDescendant(byID, entry, memo, existingFormTargets) {
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
