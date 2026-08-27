package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"

	"evolyn/internal/contextx"
	apperrors "evolyn/internal/platform/application"
	"evolyn/internal/platform/application/model"
	"evolyn/internal/platform/application/repository"
	"evolyn/internal/platform/httpx"
	iammodel "evolyn/internal/platform/iam/model"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// menuService 应用菜单服务（M2-菜单-1 只读骨架，方案 §6）：读取快照 →
// 树完整性校验 → 可见性裁剪 → capabilities 派生。写路径（分组创建/移动/
// 重排/删除与 baseMenuRevision 条件递增）随 M2-菜单-3 落地
type menuService struct {
	repo    repository.MenuRepository
	access  ApplicationAccessEvaluator
	formDir FormDirectory
}

// NewMenuService 构造菜单服务；访问判定与鉴权中间件同源（复用应用域
// ApplicationAccessEvaluator，§6.1：Service 复核不能仅依赖中间件）
func NewMenuService(repo repository.MenuRepository, access ApplicationAccessEvaluator) ApplicationMenuService {
	return &menuService{repo: repo, access: access}
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

	// M2-资产-1：表单目标存在性批量查询。existingForms 以 nil 表达「目录
	// 未接入」（旧行为：不裁剪、不投影）；目录接入且确无表单时为空 map，
	// form 节点按不存在裁剪——两种空态语义必须区分。
	var existingForms map[uint]bool
	formIDs := make([]uint, 0)
	for i := range snap.Entries {
		if snap.Entries[i].EntryType == model.MenuEntryTypeForm && snap.Entries[i].TargetID != nil {
			formIDs = append(formIDs, *snap.Entries[i].TargetID)
		}
	}
	if s.formDir != nil && len(formIDs) > 0 {
		existingForms, err = s.formDir.ExistingFormIDs(ctx, formIDs)
		if err != nil {
			return nil, err
		}
	}

	return buildMenuSnapshot(perms, snap, existingForms)
}

// assetVisible 资产节点可见性判定（M2-资产-1 起按目录端口注入的存在集
// 执行）：form 节点在表单软删/不存在时不可见；目录未接入（existingForms
// 为 nil）时保持旧行为——按节点存在性出网。仪表盘/页面域落地后在此扩展。
var assetVisible = func(entry *model.MenuEntry, existingForms map[uint]bool) bool {
	switch entry.EntryType {
	case model.MenuEntryTypeForm:
		if existingForms == nil {
			return true
		}
		return entry.TargetID != nil && existingForms[*entry.TargetID]
	default:
		return true
	}
}

// buildMenuSnapshot 由仓储快照组装出网视图：结构完整性校验 → 可见性
// 裁剪 → 排序投影。损坏树（孤儿/父非分组/循环）记录告警并返回
// ErrMenuInvalid，不把异常数据当正常树交给前端（方案 §6.3）
func buildMenuSnapshot(perms map[string]bool, snap *repository.MenuSnapshot, existingForms map[uint]bool) (*model.MenuSnapshot, error) {
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

	// 可见性裁剪（方案 §6.3）：资产节点按 assetVisible 判定（资产域落地
	// 前默认可见，见钩子注释），随后自底向上裁掉没有可见后代的分组
	//（含空分组）
	visible := make(map[uint]bool, len(snap.Entries))
	groupMemo := make(map[uint]bool, len(snap.Entries))
	for i := range snap.Entries {
		entry := &snap.Entries[i]
		switch entry.EntryType {
		case model.MenuEntryTypeGroup:
			visible[entry.ID] = hasVisibleDescendant(byID, entry, groupMemo, existingForms)
		default:
			visible[entry.ID] = assetVisible(entry, existingForms)
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
		out.EntryMap[entry.Code] = menuEntryDetail(entry, byID, caps, existingForms)
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
func hasVisibleDescendant(byID map[uint]*model.MenuEntry, group *model.MenuEntry, memo map[uint]bool, existingForms map[uint]bool) bool {
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
			if assetVisible(entry, existingForms) {
				keep = true
				break
			}
			continue
		}
		if hasVisibleDescendant(byID, entry, memo, existingForms) {
			keep = true
			break
		}
	}
	memo[group.ID] = keep
	return keep
}

// menuEntryDetail 节点出网投影：icon/color 空串投影为 null；parentEntryId
// 由 byID 反查父节点编码（null 即根节点）。M2-资产-1 起 form 节点按目录
// 存在集投影 target（表单公开编码 = 数字 ID 字符串，与工作区路由参数一致）；
// 目录未接入（existingForms nil）时 target 保持不投影。仪表盘/页面域随
// 各自资产批次扩展
func menuEntryDetail(entry *model.MenuEntry, byID map[uint]*model.MenuEntry, caps model.MenuEntryCapabilities, existingForms map[uint]bool) model.MenuEntryDetail {
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
	// 目录接入后才投影（existingForms 非 nil 且表单存在）；未接入时保持
	// 旧行为（target 出网 null），与裁剪钩子的 nil 语义一致。
	if entry.EntryType == model.MenuEntryTypeForm && entry.TargetID != nil &&
		existingForms != nil && existingForms[*entry.TargetID] {
		detail.Target = &model.MenuEntryTarget{
			Type: model.MenuEntryTypeForm,
			ID:   strconv.FormatUint(uint64(*entry.TargetID), 10),
		}
	}
	return detail
}

// provisionStatus 与应用域 isProvisioning 同口径（pending/running 进行中）；
// 菜单服务不依赖 applicationService 实例，独立成函数避免循环依赖
func provisionStatus(status string) bool {
	return status == model.ProvisionStatusPending || status == model.ProvisionStatusRunning
}
