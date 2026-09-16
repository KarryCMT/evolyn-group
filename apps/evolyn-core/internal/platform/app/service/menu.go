package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"evolyn/internal/contextx"
	"evolyn/internal/metrics"
	kernel "evolyn/internal/model"
	apperrors "evolyn/internal/platform/app"
	"evolyn/internal/platform/app/model"
	"evolyn/internal/platform/app/repository"
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
	tx       TxManager
	repo     repository.MenuRepository
	audit    auditservice.Recorder
	access   AppAccessEvaluator
	formDir  FormDirectory
	formPerm FormPermissionDirectory
}

// NewMenuService 构造菜单服务；访问判定与鉴权中间件同源（复用应用域
// AppAccessEvaluator，§6.1：Service 复核不能仅依赖中间件）
func NewMenuService(tx TxManager, repo repository.MenuRepository, audit auditservice.Recorder, access AppAccessEvaluator) AppMenuService {
	return &menuService{tx: tx, repo: repo, audit: audit, access: access}
}

// UseFormPermissionDirectory 注入表单权限裁剪窄端口（表单权限 P1，装配期
// 一次性调用）：成员侧表单节点按「权限组入口判定 view ∨ add」二次裁剪；
// 未注入时保持既有可见性行为（仅存在性裁剪）。
func (s *menuService) UseFormPermissionDirectory(dir FormPermissionDirectory) {
	s.formPerm = dir
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
// 返回空树（rootMenuIds 空）是合法结果，触发前端空应用引导
func (s *menuService) GetMenu(ctx context.Context, member *iammodel.User, code string) (*model.MenuSnapshot, error) {
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("tenant context required")
	}

	// 与路由层 GET /apps/code/:code/menu（verb=get）同口径复核；
	// 无 apps:get 时按「应用不存在」出网（§6.1 统一口径），
	// 细节经 Wrap 只入日志
	perms := s.access.Permissions(ctx, member)
	if !perms["apps:get"] {
		return nil, httpx.Wrap(apperrors.ErrNotFound,
			fmt.Errorf("member %d (tenant %d) cannot read menu of app %s", memberID(member), tenantID, code))
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
	favorites, err := s.repo.FavoriteMenuIDs(ctx, tenantID, member.ID, snap.AppID)
	if err != nil {
		return nil, err
	}

	// M2-资产-1 / 表单权限 P1：表单节点的存在性裁剪与入口权限裁剪，
	// 与收藏写入（AddFavorite）、收藏列表（ListFavorites）共用同一输入，
	// 保证「可见即口径一致」的唯一事实源
	existingFormTargets, visibleFormIDs, err := s.formVisibilityInputs(ctx, member.ID, formTargetIDsOf(snap.Nodes))
	if err != nil {
		return nil, err
	}

	return buildMenuSnapshot(perms, snap, existingFormTargets, favorites, visibleFormIDs)
}

// formTargetIDsOf 收集节点集合中的表单资产内部 ID（目录/权限端口入参）。
func formTargetIDsOf(nodes []model.MenuNode) []uint {
	formIDs := make([]uint, 0)
	for i := range nodes {
		if nodes[i].MenuType == model.MenuTypeForm && nodes[i].TargetID != nil {
			formIDs = append(formIDs, *nodes[i].TargetID)
		}
	}
	return formIDs
}

// formVisibilityInputs 表单资产可见性端口输入（读侧统一事实源，P1 收敛）：
// existingFormTargets 为 nil 表达「目录端口未接入」（旧行为：不裁剪、
// target 不投影），目录接入且确无表单时为空 map（form 节点按不存在裁剪）
// ——两种空态语义必须区分；visibleFormIDs 的 nil 语义同构（权限端口
// 未接入 = 仅按存在性放行）。
func (s *menuService) formVisibilityInputs(ctx context.Context, memberID uint, formIDs []uint) (map[uint]FormTargetProjection, map[uint]bool, error) {
	var existingFormTargets map[uint]FormTargetProjection
	var visibleFormIDs map[uint]bool
	if s.formDir != nil && len(formIDs) > 0 {
		var err error
		if existingFormTargets, err = s.formDir.ExistingFormTargets(ctx, formIDs); err != nil {
			return nil, nil, err
		}
	}
	if s.formPerm != nil && len(formIDs) > 0 {
		var err error
		if visibleFormIDs, err = s.formPerm.VisibleFormIDs(ctx, memberID, formIDs); err != nil {
			return nil, nil, err
		}
	}
	return existingFormTargets, visibleFormIDs, nil
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
	if !perms["apps:create"] {
		return nil, httpx.Wrap(apperrors.ErrForbidden,
			fmt.Errorf("member %d cannot create menu group in app %s", member.ID, code))
	}

	name := strings.TrimSpace(req.Name)
	if name == "" || utf8.RuneCountInString(name) > 128 {
		return nil, httpx.Wrap(apperrors.ErrMenuNameInvalid,
			fmt.Errorf("menu group name rune count %d", utf8.RuneCountInString(name)))
	}
	var parentCode *string
	if req.ParentMenuID != nil {
		trimmed := strings.TrimSpace(*req.ParentMenuID)
		if trimmed == "" {
			return nil, httpx.Wrap(apperrors.ErrMenuParentInvalid, errors.New("parent node id is blank"))
		}
		parentCode = &trimmed
	}

	var created *model.MenuNode
	// 应用维度快照（000064）：事务内读取，提交后供审计固化
	var appID uint
	var appCode, appName string
	if s.tx == nil {
		return nil, errors.New("app menu transaction manager is required")
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
				fmt.Errorf("app %d provisioning (%s)", snap.AppID, snap.ProvisionStatus))
		}
		if snap.Status != model.AppStatusActive {
			return httpx.Wrap(apperrors.ErrStatusInvalid,
				fmt.Errorf("app %d status %s", snap.AppID, snap.Status))
		}

		advanced, err := s.repo.BumpMenuRevisionFrom(tctx, snap.AppID, req.BaseMenuRevision)
		if err != nil {
			return err
		}
		if !advanced {
			return httpx.Wrap(apperrors.ErrMenuVersionConflict,
				fmt.Errorf("app %d menu revision changed from base %d", snap.AppID, req.BaseMenuRevision))
		}

		var parentMenuID *uint
		if parentCode != nil {
			parent, err := menuNodeByCode(snap.Nodes, *parentCode)
			if err != nil || parent.MenuType != model.MenuTypeGroup {
				return httpx.Wrap(apperrors.ErrMenuParentInvalid,
					fmt.Errorf("parent node %q is missing or not a group", *parentCode))
			}
			// 根分组为第一级；其子分组为第二级。第二级下不再允许创建分组。
			if parent.ParentMenuID != nil {
				return httpx.Wrap(apperrors.ErrMenuDepthExceeded,
					fmt.Errorf("parent node %q is already a nested group", *parentCode))
			}
			parentMenuID = &parent.ID
		}

		sortOrder, err := s.repo.MaxSortOrder(tctx, snap.AppID, parentMenuID)
		if err != nil {
			return err
		}
		appID, appCode, appName = snap.AppID, snap.AppCode, snap.AppName
		created, err = s.repo.CreateGroupNode(tctx, &model.MenuNode{
			TenantID:     tenantID,
			AppID:        snap.AppID,
			ParentMenuID: parentMenuID,
			MenuType:     model.MenuTypeGroup,
			Name:         name,
			SortOrder:    sortOrder + 1024,
		})
		return err
	})
	if err != nil {
		return nil, err
	}

	result := &model.MenuGroupMutation{
		MenuID:       created.Code,
		ParentMenuID: parentCode,
		Name:         name,
		MenuRevision: req.BaseMenuRevision + 1,
	}
	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "app", Action: "create", ResourceType: "app_menu_node",
			ResourceID: created.Code,
			After: map[string]any{
				"appCode":      code,
				"menuCode":     created.Code,
				"menuType":     model.MenuTypeGroup,
				"parentMenuId": parentCode,
				"name":         name,
			},
			// 应用维度快照（000064）：菜单操作归属产品日志的菜单配置分类
			AppID:   appID,
			AppCode: appCode,
			AppName: appName,
		})
	}
	return result, nil
}

// menuNodeByCode 在已与 menuRevision 同快照读取的节点集合中定位父节点，
// 避免条件递增前后再发起一次可能读到不同版本的查询。
func menuNodeByCode(nodes []model.MenuNode, code string) (*model.MenuNode, error) {
	for i := range nodes {
		if nodes[i].Code == code {
			return &nodes[i], nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

// UpdateNode 菜单节点管理更新（ADR-011）：分组改名 / 资产节点对成员隐藏 /
// 移动节点，统一走 menuRevision 乐观锁串行化。资产节点名称以资产域为事实
// 源，本接口拒绝改名（经对应资产接口修改后同事务同步回节点）。
func (s *menuService) UpdateNode(ctx context.Context, member *iammodel.User, code, menuCode string, req *model.UpdateMenuNodeRequest) (*model.MenuNodeMutation, error) {
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("tenant context required")
	}
	if req == nil {
		return nil, httpx.Wrap(apperrors.ErrMenuNameInvalid, errors.New("update menu node request is nil"))
	}
	if member == nil || member.ID == 0 || member.TenantID != tenantID {
		return nil, httpx.Wrap(apperrors.ErrMemberInvalid,
			fmt.Errorf("member %d (tenant %d) not in tenant %d", memberID(member), memberTenant(member), tenantID))
	}
	// 与路由层 PATCH /apps/...（verb=patch）同口径复核
	perms := s.access.Permissions(ctx, member)
	if !perms["apps:patch"] {
		return nil, httpx.Wrap(apperrors.ErrForbidden,
			fmt.Errorf("member %d cannot update menu node in app %s", member.ID, code))
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
				fmt.Errorf("menu node name rune count %d", utf8.RuneCountInString(name)))
		}
		newName = &name
	}
	// 移动目标：nil 指针=未提交移动；非 nil 空串=移动到根级；非空=分组编码
	var newParentCode *string
	if req.ParentMenuCode != nil {
		trimmed := strings.TrimSpace(*req.ParentMenuCode)
		newParentCode = &trimmed
	}
	if newName == nil && req.Hidden == nil && newParentCode == nil {
		return nil, httpx.Wrap(apperrors.ErrMenuNameInvalid, errors.New("no updatable field provided"))
	}

	var updated *model.MenuNode
	var newRevision int64
	// 应用维度快照（000064）：事务内读取，提交后供审计固化
	var appID uint
	var appCode, appName string
	if s.tx == nil {
		return nil, errors.New("app menu transaction manager is required")
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
				fmt.Errorf("app %d provisioning (%s)", snap.AppID, snap.ProvisionStatus))
		}
		if snap.Status != model.AppStatusActive {
			return httpx.Wrap(apperrors.ErrStatusInvalid,
				fmt.Errorf("app %d status %s", snap.AppID, snap.Status))
		}

		advanced, err := s.repo.BumpMenuRevisionFrom(tctx, snap.AppID, req.BaseMenuRevision)
		if err != nil {
			return err
		}
		if !advanced {
			return httpx.Wrap(apperrors.ErrMenuVersionConflict,
				fmt.Errorf("app %d menu revision changed from base %d", snap.AppID, req.BaseMenuRevision))
		}

		node, err := menuNodeByCode(snap.Nodes, menuCode)
		if err != nil {
			return httpx.Wrap(apperrors.ErrMenuNotFound,
				fmt.Errorf("node %q not found in app %s", menuCode, code))
		}

		fields := make(map[string]interface{}, 3)
		if newName != nil {
			// 资产节点的展示名由资产域事实源管理，拒绝旁路改名
			if node.MenuType != model.MenuTypeGroup {
				return httpx.Wrap(apperrors.ErrMenuNodeRenameForbidden,
					fmt.Errorf("node %q is an asset node of type %s", menuCode, node.MenuType))
			}
			fields["name"] = *newName
		}
		if req.Hidden != nil {
			// 分组可见性由后代派生，「对成员隐藏」仅对资产节点成立
			if node.MenuType == model.MenuTypeGroup {
				return httpx.Wrap(apperrors.ErrMenuHiddenInvalid,
					fmt.Errorf("node %q is a group node", menuCode))
			}
			fields["hidden"] = *req.Hidden
		}
		if newParentCode != nil {
			parentMenuID, err := resolveMoveParent(snap.Nodes, node, *newParentCode)
			if err != nil {
				return err
			}
			sortOrder, err := s.repo.MaxSortOrder(tctx, snap.AppID, parentMenuID)
			if err != nil {
				return err
			}
			fields["parent_menu_id"] = parentMenuID
			// 移动节点追加到目标父节点末位（服务端排序，不信任客户端排序值）
			fields["sort_order"] = sortOrder + 1024
		}
		if err := s.repo.UpdateNodeFields(tctx, snap.AppID, node.ID, fields); err != nil {
			return err
		}
		appID, appCode, appName = snap.AppID, snap.AppCode, snap.AppName
		updated = node
		newRevision = req.BaseMenuRevision + 1
		return nil
	})
	if err != nil {
		return nil, err
	}

	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "app", Action: "update", ResourceType: "app_menu_node",
			ResourceID: updated.Code,
			After: map[string]any{
				"appCode":        code,
				"menuCode":       updated.Code,
				"name":           newName,
				"hidden":         req.Hidden,
				"parentMenuCode": newParentCode,
			},
			// 应用维度快照（000064）：菜单操作归属产品日志的菜单配置分类
			AppID:   appID,
			AppCode: appCode,
			AppName: appName,
		})
	}
	return &model.MenuNodeMutation{MenuID: updated.Code, MenuRevision: newRevision}, nil
}

// resolveMoveParent 校验移动目标父节点：空串移动到根级；非空须为同应用
// 未软删分组，且不是被移动节点自身或其后代（防自环），分组移动须落到根
// 分组下（层级上限与创建分组一致：根分组+子分组两层）。
func resolveMoveParent(nodes []model.MenuNode, node *model.MenuNode, parentCode string) (*uint, error) {
	if parentCode == "" {
		return nil, nil // 根级
	}
	parent, err := menuNodeByCode(nodes, parentCode)
	if err != nil || parent.MenuType != model.MenuTypeGroup {
		return nil, httpx.Wrap(apperrors.ErrMenuParentInvalid,
			fmt.Errorf("move parent %q is missing or not a group", parentCode))
	}
	if parent.ID == node.ID {
		return nil, httpx.Wrap(apperrors.ErrMenuMoveInvalid,
			fmt.Errorf("cannot move node %q under itself", node.Code))
	}
	// 沿父链上行：目标父节点处于被移动节点子树内即形成环
	byID := make(map[uint]*model.MenuNode, len(nodes))
	for i := range nodes {
		byID[nodes[i].ID] = &nodes[i]
	}
	for cur := parent; cur.ParentMenuID != nil; {
		cur = byID[*cur.ParentMenuID]
		if cur.ID == node.ID {
			return nil, httpx.Wrap(apperrors.ErrMenuMoveInvalid,
				fmt.Errorf("cannot move node %q under its own descendant %q", node.Code, parentCode))
		}
	}
	// 分组挂载深度限制：目标父分组必须为根分组（移动后仍是两级结构）
	if node.MenuType == model.MenuTypeGroup && parent.ParentMenuID != nil {
		return nil, httpx.Wrap(apperrors.ErrMenuDepthExceeded,
			fmt.Errorf("group %q cannot move under nested group %q", node.Code, parentCode))
	}
	return &parent.ID, nil
}

// AddFavorite 收藏菜单节点（个人状态动作，ADR-011）。P1 起收藏资格收敛为
// 服务层统一策略 canFavorite，与菜单读取（GetMenu）同源，不依赖路由权限或
// 前端按钮显隐：
//
//	canFavorite(member, app, node)
//	  = appCanRead(member, app)          —— apps:get 复核，缺失统一 APP_NOT_FOUND
//	  ∧ app 可用（active 且非初始化中）  —— 归档/初始化中应用不可收藏
//	  ∧ node ∈ {form, dashboard, page}   —— group 是树结构，不可收藏
//	  ∧ node 在成员有效可见集内          —— 菜单 hidden、资产软删、表单入口
//	                                       权限（view ∨ add）与 GetMenu 同口径
//
// 重复收藏幂等成功；不递增菜单修订号——收藏不改变共享菜单结构。
func (s *menuService) AddFavorite(ctx context.Context, member *iammodel.User, appCode, menuCode string) (*model.MenuFavoriteMutation, error) {
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("tenant context required")
	}
	if member == nil || member.ID == 0 || member.TenantID != tenantID {
		return nil, httpx.Wrap(apperrors.ErrMemberInvalid,
			fmt.Errorf("member %d (tenant %d) not in tenant %d", memberID(member), memberTenant(member), tenantID))
	}
	// 与菜单读取同口径复核 apps:get：无读取权按「应用不存在」出网，
	// 不泄露应用存在性与节点细节（§6.1）
	perms := s.access.Permissions(ctx, member)
	if !perms["apps:get"] {
		return nil, httpx.Wrap(apperrors.ErrNotFound,
			fmt.Errorf("member %d (tenant %d) cannot read app %s", memberID(member), tenantID, appCode))
	}
	snap, err := s.repo.GetSnapshot(ctx, tenantID, appCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.Wrap(apperrors.ErrNotFound, err)
		}
		return nil, err
	}
	if !appUsable(snap) {
		return nil, httpx.Wrap(apperrors.ErrStatusInvalid,
			fmt.Errorf("app %d status %s (%s) not favoritable", snap.AppID, snap.Status, snap.ProvisionStatus))
	}
	node, err := menuNodeByCode(snap.Nodes, menuCode)
	if err != nil {
		return nil, httpx.Wrap(apperrors.ErrMenuFavoriteInvalid,
			fmt.Errorf("node %q not found in app %s", menuCode, appCode))
	}
	// 分组节点是树结构不是可打开入口，不可收藏
	if node.MenuType == model.MenuTypeGroup {
		return nil, httpx.Wrap(apperrors.ErrMenuFavoriteInvalid,
			fmt.Errorf("node %q is a group", menuCode))
	}
	existingFormTargets, visibleFormIDs, err := s.formVisibilityInputs(ctx, member.ID, formTargetIDsOf(snap.Nodes))
	if err != nil {
		return nil, err
	}
	// 有效可见性复核：与 GetMenu 的节点裁剪完全同源（资产软删/入口权限/
	// hidden 仅菜单管理成员可见），防止「知道编码即可收藏不可见节点」
	if !memberNodeVisible(perms, node, existingFormTargets, visibleFormIDs) {
		metrics.MenuFavoriteVisibilityRejectedTotal.Inc()
		return nil, httpx.Wrap(apperrors.ErrMenuFavoriteInvalid,
			fmt.Errorf("node %q in app %s is not visible to member %d", menuCode, appCode, member.ID))
	}
	if err := s.repo.CreateFavorite(ctx, &model.MenuFavorite{
		TenantID: tenantID,
		MemberID: member.ID,
		AppID:    snap.AppID,
		MenuID:   node.ID,
	}); err != nil {
		return nil, err
	}
	metrics.MenuFavoriteCreateTotal.Inc()
	return &model.MenuFavoriteMutation{MenuID: node.Code, Favorited: true}, nil
}

// RemoveFavorite 取消收藏（幂等）：目标收藏不存在同样返回 Favorited=false。
// 取消不要求目标仍可见——用户需要能清除已被隐藏或已删除前留下的个人状态。
func (s *menuService) RemoveFavorite(ctx context.Context, member *iammodel.User, menuCode string) (*model.MenuFavoriteMutation, error) {
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("tenant context required")
	}
	if member == nil || member.ID == 0 || member.TenantID != tenantID {
		return nil, httpx.Wrap(apperrors.ErrMemberInvalid,
			fmt.Errorf("member %d (tenant %d) not in tenant %d", memberID(member), memberTenant(member), tenantID))
	}
	if _, err := s.repo.DeleteFavoriteByCode(ctx, tenantID, member.ID, menuCode); err != nil {
		return nil, err
	}
	metrics.MenuFavoriteRemoveTotal.Inc()
	return &model.MenuFavoriteMutation{MenuID: menuCode, Favorited: false}, nil
}

// ---- 「我的收藏」跨应用列表（P2） ----

// 收藏列表分页参数：默认 20、上限 100（与应用列表同口径）；扫描批量固定
// 100 行，批内经可见性过滤后再决定是否续拉（读侧过滤可能产生短页，以
// nextCursor 续拉而非按页大小推断）
const (
	favoriteListDefaultLimit = 20
	favoriteListMaxLimit     = 100
	favoriteScanBatchSize    = 100
)

// favoriteCursor 「我的收藏」游标载荷：(createdAt 纳秒, 收藏 id)，
// 排序 created_at DESC、id DESC；对客户端不透明（base64url），只允许原样回传
type favoriteCursor struct {
	CreatedAtNano int64 `json:"createdAtNano"`
	FavoriteID    uint  `json:"favoriteId"`
}

func encodeFavoriteCursor(createdAt time.Time, favoriteID uint) string {
	data, _ := json.Marshal(favoriteCursor{CreatedAtNano: createdAt.UnixNano(), FavoriteID: favoriteID})
	return base64.RawURLEncoding.EncodeToString(data)
}

// decodeFavoriteCursor 解析并校验游标载荷（字段零值视为非法，防垃圾输入穿透分页）
func decodeFavoriteCursor(raw string) (favoriteCursor, error) {
	data, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return favoriteCursor{}, err
	}
	var payload favoriteCursor
	if err := json.Unmarshal(data, &payload); err != nil {
		return favoriteCursor{}, err
	}
	if payload.FavoriteID == 0 || payload.CreatedAtNano <= 0 {
		return favoriteCursor{}, fmt.Errorf("invalid favorite cursor")
	}
	return payload, nil
}

// ListFavorites 当前成员的跨应用收藏列表（GET /menu-favorites，P2）：
// (tenant_id, member_id) 命中索引连接未软删应用与节点后，逐批复用菜单
// 同口径可见性裁剪——失效记录（应用归档/节点隐藏/资产权限收回）只过滤
// 不删除，恢复后收藏自然恢复展示（方案 §4.3）。
func (s *menuService) ListFavorites(ctx context.Context, member *iammodel.User, query model.ListMenuFavoritesQuery) (*model.MenuFavoritePage, error) {
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("tenant context required")
	}
	if member == nil || member.ID == 0 || member.TenantID != tenantID {
		return nil, httpx.Wrap(apperrors.ErrMemberInvalid,
			fmt.Errorf("member %d (tenant %d) not in tenant %d", memberID(member), memberTenant(member), tenantID))
	}
	// 与路由层 GET /menu-favorites（verb=list）同口径复核；个人数据面，
	// 无应用存在性可泄露，按 FORBIDDEN 出网
	perms := s.access.Permissions(ctx, member)
	if !perms["menu-favorites:list"] {
		return nil, httpx.Wrap(apperrors.ErrForbidden,
			fmt.Errorf("member %d cannot list menu favorites", member.ID))
	}

	limit := query.Limit
	if limit <= 0 {
		limit = favoriteListDefaultLimit
	}
	if limit > favoriteListMaxLimit {
		limit = favoriteListMaxLimit
	}
	var (
		scanBeforeAt time.Time
		scanBeforeID uint
		hasCursor    bool
	)
	if raw := strings.TrimSpace(query.Cursor); raw != "" {
		payload, err := decodeFavoriteCursor(raw)
		if err != nil {
			return nil, httpx.Wrap(apperrors.ErrCursorInvalid, err)
		}
		scanBeforeAt = time.Unix(0, payload.CreatedAtNano)
		scanBeforeID = payload.FavoriteID
		hasCursor = true
	}

	// hidden 节点仅菜单管理成员可见（与 GetMenu 同口径）；权限集全局一次
	canManageMenu := canManageMenuPerms(perms)
	items := make([]model.MenuFavoriteItem, 0, limit)
	var lastRow *repository.FavoriteRow
	hasMore := false

	// 扫描循环：批内过滤后不足 limit 则续拉下一批；满页后继续扫描直到
	// 探测到下一条可见条目（hasMore=true）或行源耗尽，保证 nextCursor
	// 精确表达「确有下一页可见条目」（读侧过滤产生短页时不用 limit+1
	// 探测行数，而以可见条目为准）。每轮游标严格前进，循环必然终止。
	for {
		rows, err := s.repo.ListMemberFavorites(ctx, tenantID, member.ID, repository.FavoriteListParams{
			Limit:     favoriteScanBatchSize,
			HasCursor: hasCursor,
			BeforeAt:  scanBeforeAt,
			BeforeID:  scanBeforeID,
		})
		if err != nil {
			return nil, err
		}
		// 批内表单可见性输入（与 GetMenu/AddFavorite 共用端口）
		formIDs := make([]uint, 0, len(rows))
		for i := range rows {
			if rows[i].MenuType == model.MenuTypeForm && rows[i].TargetID != nil {
				formIDs = append(formIDs, *rows[i].TargetID)
			}
		}
		existingFormTargets, visibleFormIDs, err := s.formVisibilityInputs(ctx, member.ID, formIDs)
		if err != nil {
			return nil, err
		}
		for i := range rows {
			row := &rows[i]
			if !favoriteRowVisible(row, canManageMenu, existingFormTargets, visibleFormIDs) {
				continue
			}
			if len(items) < limit {
				items = append(items, favoriteItem(row, existingFormTargets))
				lastRow = row
				continue
			}
			// 满页后仍出现可见条目：确认存在下一页，停止扫描
			hasMore = true
			break
		}
		if hasMore || len(rows) < favoriteScanBatchSize {
			break // 探测成功或行源耗尽
		}
		// 从本批最后一行（未过滤）之后继续扫描
		tail := rows[len(rows)-1]
		scanBeforeAt = tail.CreatedAt
		scanBeforeID = tail.FavoriteID
		hasCursor = true
	}

	page := &model.MenuFavoritePage{Items: items}
	// 游标锚定最后一条「已返回」条目：批内被过滤的行天然落在游标之后，
	// 续拉不会重复或跳过可见条目
	if lastRow != nil && hasMore {
		page.NextCursor = encodeFavoriteCursor(lastRow.CreatedAt, lastRow.FavoriteID)
	}
	metrics.MenuFavoriteListTotal.Inc()
	return page, nil
}

// favoriteRowVisible 收藏行读侧可见性（P2，与 GetMenu 有效快照同口径）：
// 应用可用（active 且非初始化中，归档应用收藏保留但不展示）∧ 非分组
// （防御历史脏数据）∧ 资产可见（表单软删/入口权限）∧ 非隐藏或成员可
// 管理菜单。失效记录只过滤、不在读取路径删除（方案 §5.4）。
func favoriteRowVisible(row *repository.FavoriteRow, canManageMenu bool, existingFormTargets map[uint]FormTargetProjection, visibleFormIDs map[uint]bool) bool {
	if row.AppStatus != model.AppStatusActive || provisionStatus(row.AppProvisionStatus) {
		return false
	}
	if row.MenuType == model.MenuTypeGroup {
		return false
	}
	node := &model.MenuNode{MenuType: row.MenuType, TargetID: row.TargetID}
	if !assetVisible(node, existingFormTargets, visibleFormIDs) {
		return false
	}
	return !row.Hidden || canManageMenu
}

// favoriteItem 收藏行出网投影：icon 空串投影为 null；target 与
// menuNodeDetail 同口径（仅 form 节点且目录可投影时返回）。
func favoriteItem(row *repository.FavoriteRow, existingFormTargets map[uint]FormTargetProjection) model.MenuFavoriteItem {
	item := model.MenuFavoriteItem{
		App: model.MenuFavoriteAppRef{Code: row.AppCode, Name: row.AppName},
		Node: model.MenuFavoriteNodeRef{
			MenuID: row.MenuCode,
			Name:   row.MenuName,
			Type:   row.MenuType,
		},
		FavoritedAt: kernel.JSONTime(row.CreatedAt),
	}
	if row.Icon != "" {
		item.Node.Icon = &row.Icon
	}
	if row.MenuType == model.MenuTypeForm && row.TargetID != nil && existingFormTargets != nil {
		if target, ok := existingFormTargets[*row.TargetID]; ok && target.Code != "" && target.FormType != "" {
			item.Node.Target = &model.MenuNodeTarget{
				Type:     model.MenuTypeForm,
				Code:     target.Code,
				FormType: target.FormType,
			}
		}
	}
	return item
}

// appUsable 应用处于可收藏状态（canFavorite 的应用因子，P1）：active 且
// 非初始化中——与菜单快照的可编辑因子同源，归档/初始化中应用不可收藏
func appUsable(snap *repository.MenuSnapshot) bool {
	return snap.Status == model.AppStatusActive && !provisionStatus(snap.ProvisionStatus)
}

// canManageMenuPerms 成员是否可管理菜单（apps:create/patch）：hidden 节点
// 对其保持可见（否则无法恢复显示），与 buildMenuSnapshot 同口径
func canManageMenuPerms(perms map[string]bool) bool {
	return perms["apps:create"] || perms["apps:patch"]
}

// memberNodeVisible 成员对资产节点的有效可见性（P1 与 buildMenuSnapshot 的
// nodeVisible 同源）：资产可见（目录存在集 × 表单入口权限）∧ 非隐藏或
// 成员可管理菜单
func memberNodeVisible(perms map[string]bool, node *model.MenuNode, existingFormTargets map[uint]FormTargetProjection, visibleFormIDs map[uint]bool) bool {
	if !assetVisible(node, existingFormTargets, visibleFormIDs) {
		return false
	}
	return !node.Hidden || canManageMenuPerms(perms)
}

// assetVisible 资产节点可见性判定（M2-资产-1 起按目录端口注入的存在集
// 执行）：form 节点在表单软删/不存在时不可见；目录未接入（existingFormTargets
// 为 nil）时保持旧行为——按节点存在性出网。仪表盘/页面域落地后在此扩展。
// 表单权限 P1（S5/S8）：visibleFormIDs 非 nil 时 form 节点叠加权限组入口
// 判定（view ∨ add）——nil 语义 = 端口未接入保持旧行为，与存在集的 nil 语义
// 同构；接入后未命中权限组的表单在成员侧隐藏。
var assetVisible = func(node *model.MenuNode, existingFormTargets map[uint]FormTargetProjection, visibleFormIDs map[uint]bool) bool {
	switch node.MenuType {
	case model.MenuTypeForm:
		if existingFormTargets == nil {
			return true
		}
		if node.TargetID == nil {
			return false
		}
		target, ok := existingFormTargets[*node.TargetID]
		if !ok || target.Code == "" || target.FormType == "" {
			return false
		}
		if visibleFormIDs == nil {
			return true // 权限裁剪端口未接入：仅按存在性放行
		}
		return visibleFormIDs[*node.TargetID]
	default:
		return true
	}
}

// buildMenuSnapshot 由仓储快照组装出网视图：结构完整性校验 → 可见性
// 裁剪 → 排序投影。损坏树（孤儿/父非分组/循环）记录告警并返回
// ErrMenuInvalid，不把异常数据当正常树交给前端（方案 §6.3）。
// ADR-011：可见性叠加「对成员隐藏」裁剪；capabilities 携带按钮级 actions
// （动作注册表 × 权限集 × 应用状态派生）与当前成员收藏状态。
// 表单权限 P1：visibleFormIDs 为权限组入口判定结果（nil = 端口未接入）。
func buildMenuSnapshot(perms map[string]bool, snap *repository.MenuSnapshot, existingFormTargets map[uint]FormTargetProjection, favorites map[uint]bool, visibleFormIDs map[uint]bool) (*model.MenuSnapshot, error) {
	byID := make(map[uint]*model.MenuNode, len(snap.Nodes))
	for i := range snap.Nodes {
		byID[snap.Nodes[i].ID] = &snap.Nodes[i]
	}

	// 结构校验（在完整节点集上进行，先于可见性裁剪）：父引用存在且为
	// 分组、沿父链无循环。三者均为服务端数据完整性故障（写入路径被
	// Service 层校验拦截，正常不会出现）
	for i := range snap.Nodes {
		node := &snap.Nodes[i]
		if err := validateMenuAncestry(byID, node); err != nil {
			logrus.WithFields(logrus.Fields{
				"app":  snap.AppCode,
				"node": node.Code,
			}).Warnf("app menu integrity check failed: %v", err)
			return nil, httpx.Wrap(apperrors.ErrMenuInvalid, err)
		}
	}

	// 可见性裁剪：资产节点按 assetVisible 判定，并叠加「对成员隐藏」
	// （ADR-011，导航隐藏）——hidden 节点仅对持菜单管理权限的成员可见
	//（否则无法恢复显示）；可管理菜单的成员还需看到刚创建的空分组才能
	// 继续向其中添加资产，只读成员仍只看到有可见后代的分组，避免泄露
	// 无内容的管理结构。nodeVisible 与 AddFavorite 的 memberNodeVisible 同源。
	canManageMenu := canManageMenuPerms(perms)
	nodeVisible := func(node *model.MenuNode) bool {
		return memberNodeVisible(perms, node, existingFormTargets, visibleFormIDs)
	}
	visible := make(map[uint]bool, len(snap.Nodes))
	groupMemo := make(map[uint]bool, len(snap.Nodes))
	for i := range snap.Nodes {
		node := &snap.Nodes[i]
		switch node.MenuType {
		case model.MenuTypeGroup:
			visible[node.ID] = canManageMenu || hasVisibleDescendant(byID, node, groupMemo, nodeVisible)
		default:
			visible[node.ID] = nodeVisible(node)
		}
	}

	out := &model.MenuSnapshot{
		AppCode:      snap.AppCode,
		MenuRevision: snap.MenuRevision,
		RootMenuIDs:  make([]string, 0),
		NodeMap:      make(map[string]model.MenuNodeDetail, len(visible)),
		Features:     model.MenuFeatures{Workflow: false}, // 流程引擎未接入，能力注册表落地前恒 false
	}

	// 应用可编辑状态是全部按钮动作的公共因子（归档/初始化中应用只读）；
	// favorite 与 AddFavorite 的 canFavorite 策略同源（P1）：仅资产叶子
	// 节点且应用可用（active 且非初始化中）——分组节点与非可用应用投影
	// false，前端据此不渲染收藏入口；按钮细节由 menuNodeActions 按动作
	// 注册表 × 权限集投影（ADR-011，actions 为唯一按钮事实源）
	editable := appUsable(snap)

	roots := make([]*model.MenuNode, 0)
	for i := range snap.Nodes {
		node := &snap.Nodes[i]
		if !visible[node.ID] {
			continue
		}
		caps := model.MenuNodeCapabilities{
			View:     true,
			Favorite: editable && node.MenuType != model.MenuTypeGroup,
			Actions:  menuNodeActions(perms, node, editable),
		}
		detail := menuNodeDetail(node, byID, caps, existingFormTargets)
		detail.Favorited = favorites[node.ID]
		out.NodeMap[node.Code] = detail
		if node.ParentMenuID == nil {
			roots = append(roots, node)
		}
	}

	// 根顺序契约：sortOrder ASC, menuId ASC（方案 §6.2；仓储快照已按
	// (sort_order, code) 有序返回，此处排序为投影层兜底，二者同键）
	sort.Slice(roots, func(a, b int) bool {
		if roots[a].SortOrder != roots[b].SortOrder {
			return roots[a].SortOrder < roots[b].SortOrder
		}
		return roots[a].Code < roots[b].Code
	})
	for _, root := range roots {
		out.RootMenuIDs = append(out.RootMenuIDs, root.Code)
	}
	return out, nil
}

// menuNodeActions 节点按钮能力投影（ADR-011）：按节点类型查动作注册表，
// 以「授权键命中 × 应用可编辑」求值后映射为出网结构。注册表未登记的
// 动作（如分组上的表单专属动作）保持零值 false。
func menuNodeActions(perms map[string]bool, node *model.MenuNode, editable bool) model.MenuNodeActions {
	granted := authorization.MenuActionsOf(perms, node.MenuType)
	actions := model.MenuNodeActions{}
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
func validateMenuAncestry(byID map[uint]*model.MenuNode, node *model.MenuNode) error {
	if node.ParentMenuID == nil {
		return nil
	}
	parent, ok := byID[*node.ParentMenuID]
	if !ok {
		return fmt.Errorf("node %d references missing parent %d", node.ID, *node.ParentMenuID)
	}
	if parent.MenuType != model.MenuTypeGroup {
		return fmt.Errorf("node %d has non-group parent %d", node.ID, parent.ID)
	}
	// 沿父链上行：步数超过节点总数仍未到根即存在循环
	seen := 0
	for cur := node; cur.ParentMenuID != nil; {
		cur = byID[*cur.ParentMenuID]
		seen++
		if seen > len(byID) {
			return fmt.Errorf("cycle detected via node %d", node.ID)
		}
	}
	return nil
}

// hasVisibleDescendant 自底向上裁剪：分组可见当且仅当存在可见后代节点
// （可见资产，或自身有可见后代的子分组）；空分组被裁剪（方案目标 3）。
// memo 缓存已判定分组避免链形树重复递归；树已被 validateMenuAncestry
// 保证无环，递归必然终止。visible 为可见性谓词（含隐藏裁剪，ADR-011）。
func hasVisibleDescendant(byID map[uint]*model.MenuNode, group *model.MenuNode, memo map[uint]bool, visible func(*model.MenuNode) bool) bool {
	if cached, ok := memo[group.ID]; ok {
		return cached
	}
	// 先写 false 防御：无环前提下不会回读自身，仅为内存安全兜底
	memo[group.ID] = false
	keep := false
	for _, node := range byID {
		if node.ParentMenuID == nil || *node.ParentMenuID != group.ID {
			continue
		}
		if node.MenuType != model.MenuTypeGroup {
			if visible(node) {
				keep = true
				break
			}
			continue
		}
		if hasVisibleDescendant(byID, node, memo, visible) {
			keep = true
			break
		}
	}
	memo[group.ID] = keep
	return keep
}

// menuNodeDetail 节点出网投影：icon/color 空串投影为 null；parentMenuId
// 由 byID 反查父节点编码（null 即根节点）。M2-资产-1 起 form 节点按目录
// 存在集投影 target（表单公开编码为 form_ 前缀稳定编码，formType 为表单域事实）；
// 目录未接入（existingFormTargets nil）时 target 保持不投影。仪表盘/页面域随
// 各自资产批次扩展
func menuNodeDetail(node *model.MenuNode, byID map[uint]*model.MenuNode, caps model.MenuNodeCapabilities, existingFormTargets map[uint]FormTargetProjection) model.MenuNodeDetail {
	detail := model.MenuNodeDetail{
		MenuID:       node.Code,
		Type:         node.MenuType,
		Name:         node.Name,
		SortOrder:    node.SortOrder,
		Capabilities: caps,
	}
	if node.ParentMenuID != nil {
		if parent, ok := byID[*node.ParentMenuID]; ok {
			parentCode := parent.Code
			detail.ParentMenuID = &parentCode
		}
	}
	if node.Icon != "" {
		detail.Icon = &node.Icon
	}
	if node.Color != "" {
		detail.Color = &node.Color
	}
	// 目录接入后才投影（existingFormTargets 非 nil 且表单存在）；未接入时保持
	// 旧行为（target 出网 null），与裁剪钩子的 nil 语义一致。
	if node.MenuType == model.MenuTypeForm && node.TargetID != nil && existingFormTargets != nil {
		target, ok := existingFormTargets[*node.TargetID]
		if !ok || target.Code == "" || target.FormType == "" {
			return detail
		}
		detail.Target = &model.MenuNodeTarget{
			Type:     model.MenuTypeForm,
			Code:     target.Code,
			FormType: target.FormType,
		}
	}
	return detail
}

// provisionStatus 与应用域 isProvisioning 同口径（pending/running 进行中）；
// 菜单服务不依赖 appService 实例，独立成函数避免循环依赖
func provisionStatus(status string) bool {
	return status == model.ProvisionStatusPending || status == model.ProvisionStatusRunning
}
