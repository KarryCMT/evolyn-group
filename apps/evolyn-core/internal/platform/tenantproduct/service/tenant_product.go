package service

import (
	"context"
	"errors"
	"fmt"

	auditservice "evolyn/internal/platform/audit/service"
	"evolyn/internal/platform/httpx"
	iammodel "evolyn/internal/platform/iam/model"
	apperrors "evolyn/internal/platform/tenantproduct"
	"evolyn/internal/platform/tenantproduct/model"
	"evolyn/internal/platform/tenantproduct/repository"

	"gorm.io/gorm"
)

// tenantProductService 产品中心服务实现：启停与范围更新经 TxManager 单事务
// 提交（文档 8.1），审计在提交成功后 best-effort 记录（业务回滚不留假流水）
type tenantProductService struct {
	tx       TxManager
	repo     repository.Repository
	editions EditionReader
	audit    auditservice.Recorder
}

func NewTenantProductService(
	tx TxManager,
	repo repository.Repository,
	editions EditionReader,
	audit auditservice.Recorder,
) TenantProductService {
	return &tenantProductService{tx: tx, repo: repo, editions: editions, audit: audit}
}

// ---- 读取 ----

// List 产品中心卡片列表：目录全量（含平台停用项，卡片仍可见但入口不可用）
// 按展示顺序返回；配置缺失（目录先于回填策略新增）时保守合成为停用卡片
func (s *tenantProductService) List(ctx context.Context, tenantID uint) (*model.ProductCenterView, error) {
	catalogs, err := s.repo.ListCatalog(ctx)
	if err != nil {
		return nil, err
	}
	configs, err := s.repo.ListConfigsByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	configByProduct := make(map[uint]*model.TenantProductConfig, len(configs))
	for i := range configs {
		configByProduct[configs[i].ProductID] = &configs[i]
	}

	// 版本投影与列表同源失败：edition 域读取故障时整页失败（与版本信息页一致）
	editionView, err := s.readEdition(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	items := make([]model.ProductCard, 0, len(catalogs))
	for i := range catalogs {
		card, err := s.buildCard(ctx, tenantID, &catalogs[i], configByProduct[catalogs[i].ID], editionView)
		if err != nil {
			return nil, err
		}
		items = append(items, *card)
	}
	return &model.ProductCenterView{Items: items}, nil
}

// latestCard 写操作成功后的最新卡片组装：配置行必定存在（事务内刚写过）。
// edition 读取故障时降级为空版本视图——写已提交成功，不能因展示附属信息
// 失败诱导客户端重试已生效的变更
func (s *tenantProductService) latestCard(ctx context.Context, tenantID uint, productCode string) (*model.ProductCard, error) {
	catalog, err := s.repo.GetCatalogByCode(ctx, productCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrProductNotFound
		}
		return nil, err
	}
	config, err := s.repo.GetConfig(ctx, tenantID, catalog.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrProductNotFound
		}
		return nil, err
	}
	editionView, _ := s.readEdition(ctx, tenantID)
	return s.buildCard(ctx, tenantID, catalog, config, editionView)
}

// buildCard 组装单张产品卡片；config 为 nil 时保守合成停用默认视图
// （目录先于存量回填/开通种子到达是唯一场景，写入路径仍按未初始化拒绝）
func (s *tenantProductService) buildCard(
	ctx context.Context,
	tenantID uint,
	catalog *model.ProductCatalog,
	config *model.TenantProductConfig,
	editionView model.ProductEdition,
) (*model.ProductCard, error) {
	card := &model.ProductCard{
		Code:      catalog.Code,
		Name:      catalog.Name,
		Icon:      catalog.Icon,
		EntryPath: catalog.EntryPath,
		Edition:   editionView,
	}
	if config == nil {
		card.Enabled = false
		card.AccessScope = model.AccessScopeView{Mode: model.ScopeModeAll, DepartmentIds: []uint{}, MemberIds: []uint{}, Selections: []model.ScopeSelection{}}
		count, err := s.repo.CountActiveMembers(ctx, tenantID)
		if err != nil {
			return nil, err
		}
		card.AccessScope.EligibleMemberCount = count
		return card, nil
	}

	card.Enabled = config.Enabled
	card.Revision = config.Revision

	scope := model.AccessScopeView{
		Mode:          config.ScopeMode,
		DepartmentIds: []uint{},
		MemberIds:     []uint{},
		Selections:    []model.ScopeSelection{},
	}
	if config.ScopeMode == model.ScopeModeAll {
		count, err := s.repo.CountActiveMembers(ctx, tenantID)
		if err != nil {
			return nil, err
		}
		scope.EligibleMemberCount = count
	} else {
		if err := s.buildPartialScope(ctx, tenantID, config.ID, &scope); err != nil {
			return nil, err
		}
	}
	card.AccessScope = scope
	return card, nil
}

// buildPartialScope 组装 partial 范围视图：悬挂引用（部门删除/停用、成员
// 离职/禁用/跨租户）直接从展示中丢弃，不放大范围也不报错（文档 5.5）；
// eligibleMemberCount 为过滤后有效范围内 active 成员去重计数
func (s *tenantProductService) buildPartialScope(ctx context.Context, tenantID, configID uint, scope *model.AccessScopeView) error {
	departmentIDs, err := s.repo.ListScopeDepartments(ctx, configID)
	if err != nil {
		return err
	}
	memberIDs, err := s.repo.ListScopeMembers(ctx, configID)
	if err != nil {
		return err
	}
	departments, err := s.repo.ListTenantDepartments(ctx, tenantID)
	if err != nil {
		return err
	}

	// 有效部门 = 存在且 active；同时用于子部门展开
	validDepartments := make(map[uint]iammodel.Department, len(departments))
	for _, dept := range departments {
		if dept.Status == iammodel.DeptActive {
			validDepartments[dept.ID] = dept
		}
	}
	for _, id := range departmentIDs {
		if dept, ok := validDepartments[id]; ok {
			scope.DepartmentIds = append(scope.DepartmentIds, id)
			scope.Selections = append(scope.Selections, model.ScopeSelection{
				Type: "department", ID: id, Label: dept.Name,
			})
		}
	}

	// 有效成员 = 存在且 active；标签取当前昵称
	members, err := s.repo.ListMembersByIDs(ctx, tenantID, memberIDs)
	if err != nil {
		return err
	}
	activeMembers := make(map[uint]iammodel.User, len(members))
	for _, member := range members {
		if member.Status == iammodel.MemberStatusActive {
			activeMembers[member.ID] = member
		}
	}
	for _, id := range memberIDs {
		if member, ok := activeMembers[id]; ok {
			scope.MemberIds = append(scope.MemberIds, id)
			scope.Selections = append(scope.Selections, model.ScopeSelection{
				Type: "member", ID: id, Label: member.Nickname,
			})
		}
	}

	expanded := expandActiveDescendants(departmentIDs, departments)
	count, err := s.repo.CountActiveMembersInScope(ctx, tenantID, scope.MemberIds, expanded)
	if err != nil {
		return err
	}
	scope.EligibleMemberCount = count
	return nil
}

// readEdition 读取版本投影（套餐编码/名称/订阅状态），nil 端口返回空视图
func (s *tenantProductService) readEdition(ctx context.Context, tenantID uint) (model.ProductEdition, error) {
	if s.editions == nil {
		return model.ProductEdition{}, nil
	}
	current, err := s.editions.GetCurrent(ctx, tenantID)
	if err != nil {
		return model.ProductEdition{}, err
	}
	if current == nil {
		return model.ProductEdition{}, nil
	}
	return model.ProductEdition{
		PlanCode: current.Subscription.PlanCode,
		PlanName: current.Subscription.PlanName,
		Status:   current.Subscription.Status,
	}, nil
}

// ---- 写入 ----

// SetEnabled 启停产品（文档 6.2）：锁定配置行 → 校验 revision → 乐观更新
// 并递增版本号；提交成功后 best-effort 审计，返回最新卡片
func (s *tenantProductService) SetEnabled(ctx context.Context, tenantID uint, productCode string, req *model.UpdateEnabledRequest) (*model.ProductCard, error) {
	catalog, err := s.loadCatalog(ctx, productCode)
	if err != nil {
		return nil, err
	}

	var beforeEnabled bool
	err = s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		config, err := s.repo.LockConfig(tctx, tenantID, catalog.ID)
		if err != nil {
			return wrapConfigError(err)
		}
		beforeEnabled = config.Enabled
		if config.Revision != req.Revision {
			return apperrors.ErrRevisionConflict
		}
		ok, err := s.repo.UpdateEnabledWithRevision(tctx, config.ID, config.Revision, req.Enabled)
		if err != nil {
			return err
		}
		if !ok {
			return apperrors.ErrRevisionConflict
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// 审计在事务提交成功后独立写入（best-effort，失败不回滚业务）
	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "tenantproduct", Action: "update_enabled", ResourceType: "tenant_product",
			ResourceID: catalog.Code, TenantID: tenantID,
			Before: map[string]any{"enabled": beforeEnabled},
			After:  map[string]any{"enabled": req.Enabled},
		})
	}
	return s.latestCard(ctx, tenantID, productCode)
}

// UpdateAccessScope 全量替换可用范围（文档 6.3）：形状校验事务外快速失败；
// 事务内锁定配置 → revision 校验 → 部门/成员同租户有效性校验 → 更新模式
// → 全量替换关联；提交成功后 best-effort 审计，返回最新卡片
func (s *tenantProductService) UpdateAccessScope(ctx context.Context, tenantID uint, productCode string, req *model.UpdateAccessScopeRequest) (*model.ProductCard, error) {
	if req.Mode != model.ScopeModeAll && req.Mode != model.ScopeModePartial {
		return nil, apperrors.ErrScopeInvalid
	}
	// 重复 ID 去重（文档 11 用例 4），保持首次出现顺序
	departmentIDs := dedupeUint(req.DepartmentIds)
	memberIDs := dedupeUint(req.MemberIds)
	if req.Mode == model.ScopeModeAll {
		// all 不允许携带 ID：携带即视为参数错误，避免静默丢弃造成误解
		if len(departmentIDs) > 0 || len(memberIDs) > 0 {
			return nil, apperrors.ErrScopeInvalid
		}
	} else if len(departmentIDs) == 0 && len(memberIDs) == 0 {
		// partial 空范围不允许：既不能退化为「无人可用」也不能扩大为全部成员
		return nil, apperrors.ErrScopeEmpty
	}

	catalog, err := s.loadCatalog(ctx, productCode)
	if err != nil {
		return nil, err
	}

	var beforeSnap map[string]any
	err = s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		config, err := s.repo.LockConfig(tctx, tenantID, catalog.ID)
		if err != nil {
			return wrapConfigError(err)
		}
		if config.Revision != req.Revision {
			return apperrors.ErrRevisionConflict
		}

		if beforeSnap, err = s.scopeSnapshot(tctx, config); err != nil {
			return err
		}

		// partial：在写事务内校验全部部门/成员属于当前租户且有效（文档 7：
		// 禁止以裸 ID 写关联表；跨租户/离职/停用一律稳定业务错误码拒绝）
		if req.Mode == model.ScopeModePartial {
			departments, err := s.repo.ListTenantDepartments(tctx, tenantID)
			if err != nil {
				return err
			}
			validDepartments := make(map[uint]struct{}, len(departments))
			for _, dept := range departments {
				if dept.Status == iammodel.DeptActive {
					validDepartments[dept.ID] = struct{}{}
				}
			}
			for _, id := range departmentIDs {
				if _, ok := validDepartments[id]; !ok {
					return wrapInvalidID(apperrors.ErrDepartmentInvalid, id, "department")
				}
			}
			members, err := s.repo.ListMembersByIDs(tctx, tenantID, memberIDs)
			if err != nil {
				return err
			}
			activeMembers := make(map[uint]struct{}, len(members))
			for _, member := range members {
				if member.Status == iammodel.MemberStatusActive {
					activeMembers[member.ID] = struct{}{}
				}
			}
			for _, id := range memberIDs {
				if _, ok := activeMembers[id]; !ok {
					return wrapInvalidID(apperrors.ErrMemberInvalid, id, "member")
				}
			}
		}

		ok, err := s.repo.UpdateScopeWithRevision(tctx, config.ID, config.Revision, req.Mode)
		if err != nil {
			return err
		}
		if !ok {
			return apperrors.ErrRevisionConflict
		}
		return s.repo.ReplaceScope(tctx, config, departmentIDs, memberIDs)
	})
	if err != nil {
		return nil, err
	}

	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "tenantproduct", Action: "update_scope", ResourceType: "tenant_product",
			ResourceID: catalog.Code, TenantID: tenantID,
			Before: beforeSnap,
			After: map[string]any{
				"mode":          req.Mode,
				"departmentIds": departmentIDs,
				"memberIds":     memberIDs,
			},
		})
	}
	return s.latestCard(ctx, tenantID, productCode)
}

// SeedDefaults 租户开通事务内初始化产品配置（文档 8.2）：为全部当前
// active 目录创建 enabled=true、scope=all 的配置行，不建范围关联；
// 已有配置的产品跳过（幂等）。调用方事务回滚时本方法写入一并回滚
func (s *tenantProductService) SeedDefaults(ctx context.Context, tenantID uint) error {
	catalogs, err := s.repo.ListCatalog(ctx)
	if err != nil {
		return err
	}
	for i := range catalogs {
		if catalogs[i].Status != model.CatalogStatusActive {
			continue
		}
		_, err := s.repo.GetConfig(ctx, tenantID, catalogs[i].ID)
		if err == nil {
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		config := &model.TenantProductConfig{
			ProductID: catalogs[i].ID,
			Enabled:   true,
			ScopeMode: model.ScopeModeAll,
			Revision:  1,
		}
		// 开通事务 ctx 指向新租户，但仓储统一剥离上下文，显式赋值与
		// Callback 回填口径一致（同 memberfield SeedDefaults）
		config.TenantID = tenantID
		if err := s.repo.CreateConfig(ctx, config); err != nil {
			return err
		}
	}
	return nil
}

// ---- 内部助手 ----

func (s *tenantProductService) loadCatalog(ctx context.Context, productCode string) (*model.ProductCatalog, error) {
	catalog, err := s.repo.GetCatalogByCode(ctx, productCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrProductNotFound
		}
		return nil, err
	}
	return catalog, nil
}

// scopeSnapshot 事务内读取替换前的范围现状（审计 Before）
func (s *tenantProductService) scopeSnapshot(ctx context.Context, config *model.TenantProductConfig) (map[string]any, error) {
	departmentIDs, err := s.repo.ListScopeDepartments(ctx, config.ID)
	if err != nil {
		return nil, err
	}
	memberIDs, err := s.repo.ListScopeMembers(ctx, config.ID)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"mode":          config.ScopeMode,
		"departmentIds": departmentIDs,
		"memberIds":     memberIDs,
	}, nil
}

// wrapConfigError 配置行读取错误归一：未初始化映射为稳定 404 业务码
func wrapConfigError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return apperrors.ErrProductNotFound
	}
	return err
}

// wrapInvalidID 为业务错误附加原始错误（细节只入日志，ADR-008）
func wrapInvalidID(biz *httpx.BizError, id uint, kind string) error {
	return httpx.Wrap(biz, fmt.Errorf("invalid %s id %d", kind, id))
}

// dedupeUint 去重并保持首次出现顺序
func dedupeUint(ids []uint) []uint {
	seen := make(map[uint]struct{}, len(ids))
	result := make([]uint, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

// containsUint 线性包含判定（范围关联规模为配置级小集合，无需索引结构）
func containsUint(ids []uint, target uint) bool {
	for _, id := range ids {
		if id == target {
			return true
		}
	}
	return false
}

// expandActiveDescendants 部门子树展开（文档 5.5）：自选中部门沿有效
// （active 且未软删，软删已由查询过滤）部门向下收集全部后代；停用节点
// 自身被忽略且不再下钻——其整棵子树不因父级被选中而恢复可用
func expandActiveDescendants(selected []uint, departments []iammodel.Department) []uint {
	children := make(map[uint][]uint, len(departments))
	active := make(map[uint]struct{}, len(departments))
	for _, dept := range departments {
		if dept.Status != iammodel.DeptActive {
			continue
		}
		active[dept.ID] = struct{}{}
		if dept.ParentId != nil {
			children[*dept.ParentId] = append(children[*dept.ParentId], dept.ID)
		}
	}

	expanded := make(map[uint]struct{})
	queue := make([]uint, 0, len(selected))
	for _, id := range selected {
		if _, ok := active[id]; !ok {
			continue
		}
		if _, seen := expanded[id]; seen {
			continue
		}
		expanded[id] = struct{}{}
		queue = append(queue, id)
	}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, child := range children[current] {
			if _, ok := active[child]; !ok {
				continue
			}
			if _, seen := expanded[child]; seen {
				continue
			}
			expanded[child] = struct{}{}
			queue = append(queue, child)
		}
	}

	result := make([]uint, 0, len(expanded))
	for id := range expanded {
		result = append(result, id)
	}
	return result
}
