// FormPermissionEvaluator 权限组判定器（表单权限 P1，设计 §8.1）。
//
// 组绑定判定（S2 定版）：操作、字段、数据范围不得跨组扁平化并集——每次具体
// 判定按「命中组含此操作 ∧ 该组数据范围命中目标记录 ∧ 该组字段矩阵放行」的
// 组内合取执行；并集仅允许出现在同一维度内的多组合成。杜绝「A 组可编辑 X 范围
// + B 组可查看 Y 范围 → 可编辑 Y」的串联越权。
//
// 语义基线：
//   - S3 数据面旁路仅认显式动作键 form-data:admin（经 AccessEvaluator 权限集
//     解析，与鉴权中间件同源）；form-permissions:* 不触发旁路；
//   - S4 表单不存在任何权限组行（含禁用组）时 Baseline 放行（存量行为零变更）；
//   - S5 存在 ≥1 个权限组行（含禁用组）即收口：非管理员成员权限 = 命中的
//     启用组判定结果，未命中任何启用组 = 无权限；
//   - S6 数据范围空条件 = 全部数据；
//   - S7 字段矩阵 deny-by-default：判定时以当前字段清单 ∖ 矩阵覆盖 = 默认
//     不可见不可编辑，矩阵中已不在清单的字段条目忽略。
//
// 数据范围的内存匹配镜像 §5.2 的 SQL NULL 语义表（P1 执行点：旧记录操作判定
// 走内存匹配；P2 列表过滤切 SQL 编译，两侧语义一致）。
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	apperrors "evolyn/internal/platform/form"
	"evolyn/internal/platform/form/model"
	"evolyn/internal/platform/form/repository"
	"evolyn/internal/platform/httpx"
	iammodel "evolyn/internal/platform/iam/model"

	"gorm.io/gorm"
)

// formDataAdminPermissionKey 数据面旁路动作键（S3）：经权限集窄端口解析，
// 与 PermissionsOf 的动作资源注册表展开同形
const formDataAdminPermissionKey = "form-data:admin"

// FieldPermission 单字段的判定结果（可见/可编辑）
type FieldPermission struct {
	Visible  bool
	Editable bool
}

// MatchedGroup 成员命中的单个启用权限组：保留组内四要素关联，禁止消费方
// 跨组扁平化（S2 的代码化表达）。Fields 为经 S7 deny-by-default 合并后的矩阵。
type MatchedGroup struct {
	Code       string
	Operations map[string]bool
	Fields     map[string]FieldPermission
	DataScope  model.PermissionDataScopeSpec
}

// ResolvedFormPermission 判定结果：消费方只经方法判定，不自行并集；字段
// 判定必须携带操作维度，防止跨操作混合字段权限。
type ResolvedFormPermission struct {
	Admin    bool           // form-data:admin 旁路（S3）
	Baseline bool           // S4：表单无任何权限组行，执行点按基线语义放行
	Matched  []MatchedGroup // 启用且主体命中的组（S5）

	fieldList []permissionFieldMeta // 当前字段清单（S7 合并与数据范围类型分派的事实源）
}

// EntranceAllowed 入口判定（S8 定版）= view ∨ add：菜单裁剪与运行时入口按
// 「可查看或可填写」放行——仅 add 成员进入填写模式（无数据列表入口）。
func (r *ResolvedFormPermission) EntranceAllowed() bool {
	if r.Admin || r.Baseline {
		return true
	}
	for i := range r.Matched {
		ops := r.Matched[i].Operations
		if ops[model.PermissionOpView] || ops[model.PermissionOpAdd] {
			return true
		}
	}
	return false
}

// AllowsNewRecord 新记录操作判定（add/import 与 copy 的新记录侧语义，S8）：
// 不受数据范围约束（数据范围是已有数据的管理边界，「仅录入」合法）。
func (r *ResolvedFormPermission) AllowsNewRecord(op string) bool {
	if r.Admin || r.Baseline {
		return true
	}
	for i := range r.Matched {
		if r.Matched[i].Operations[op] {
			return true
		}
	}
	return false
}

// AllowsOperation 旧记录操作判定（S2 组内合取）：
// Admin || Baseline || ∃ g ∈ Matched: op ∈ g.Operations ∧ record ∈ g.DataScope。
// values 为记录 values JSONB 解码视图（键=widgetName）。
func (r *ResolvedFormPermission) AllowsOperation(op string, values map[string]any) bool {
	if r.Admin || r.Baseline {
		return true
	}
	for i := range r.Matched {
		group := &r.Matched[i]
		if group.Operations[op] && permissionScopeMatches(group.DataScope, r.fieldList, values) {
			return true
		}
	}
	return false
}

// FieldsFor 旧记录字段矩阵（S2 组内合取后同维度并集）：
// Admin/Baseline 全量可编辑；否则 ∪{ g.Fields : g 命中 op 且 record ∈ g.DataScope }。
func (r *ResolvedFormPermission) FieldsFor(op string, values map[string]any) map[string]FieldPermission {
	if r.Admin || r.Baseline {
		return allFieldsGranted(r.fieldList)
	}
	matched := make([]map[string]FieldPermission, 0, len(r.Matched))
	for i := range r.Matched {
		group := &r.Matched[i]
		if group.Operations[op] && permissionScopeMatches(group.DataScope, r.fieldList, values) {
			matched = append(matched, group.Fields)
		}
	}
	return unionFieldPermissions(r.fieldList, matched)
}

// FieldsForNew 新记录字段矩阵：Admin/Baseline 全量；否则 ∪{ g.Fields : g 含 op }
// （新记录无范围可判，S8）。
func (r *ResolvedFormPermission) FieldsForNew(op string) map[string]FieldPermission {
	if r.Admin || r.Baseline {
		return allFieldsGranted(r.fieldList)
	}
	matched := make([]map[string]FieldPermission, 0, len(r.Matched))
	for i := range r.Matched {
		group := &r.Matched[i]
		if group.Operations[op] {
			matched = append(matched, group.Fields)
		}
	}
	return unionFieldPermissions(r.fieldList, matched)
}

// RuntimeOperations 运行时投影的记录无关操作可用性（设计 §8.2）：字典中
// 「新建记录」语义的操作键（add/import）。记录级操作（view/edit/delete/copy/
// batch_*/export 与流程特有操作）由记录接口逐行判定，不在本投影内。
func (r *ResolvedFormPermission) RuntimeOperations() []string {
	operations := make([]string, 0, 2)
	for _, op := range []string{model.PermissionOpAdd, model.PermissionOpImport} {
		if r.AllowsNewRecord(op) {
			operations = append(operations, op)
		}
	}
	return operations
}

// ---- 矩阵边界与并集 ----

// allFieldsGranted Admin/Baseline 边界：全量可见可编辑（按当前字段清单逐键
// 生成，保证出网矩阵覆盖全部值字段）
func allFieldsGranted(fieldList []permissionFieldMeta) map[string]FieldPermission {
	granted := make(map[string]FieldPermission, len(fieldList))
	for _, field := range fieldList {
		granted[field.Key] = FieldPermission{Visible: true, Editable: true}
	}
	return granted
}

// allFieldsDenied 未命中任何组（S5 收口）边界：全量拒绝
func allFieldsDenied(fieldList []permissionFieldMeta) map[string]FieldPermission {
	denied := make(map[string]FieldPermission, len(fieldList))
	for _, field := range fieldList {
		denied[field.Key] = FieldPermission{}
	}
	return denied
}

// unionFieldPermissions 同维度多组矩阵并集：visible/editable 逐字段 OR
// （并集仅允许出现在字段维度内；操作与数据范围已在入参按组绑定过滤）。
func unionFieldPermissions(fieldList []permissionFieldMeta, matrices []map[string]FieldPermission) map[string]FieldPermission {
	if len(matrices) == 0 {
		return allFieldsDenied(fieldList)
	}
	union := make(map[string]FieldPermission, len(fieldList))
	for _, matrix := range matrices {
		for key, permission := range matrix {
			existing := union[key]
			existing.Visible = existing.Visible || permission.Visible
			existing.Editable = existing.Editable || permission.Editable
			union[key] = existing
		}
	}
	return union
}

// ---- 判定器实现 ----

// PermissionSubjectSource 权限组主体解析窄端口（装配层由 iam 仓储适配）：
// 返回成员的部门命中集（直系部门 ∪ 其全部祖先，即「部门含子部门」展开）与
// 角色 ID 集（含分组角色），判定器不直连 iam 表。
type PermissionSubjectSource interface {
	MemberSubject(ctx context.Context, memberID uint) (departmentIDs []uint, roleIDs []uint, err error)
}

// FormPermissionEvaluator 权限组判定器接口（设计 §8.1）。
type FormPermissionEvaluator interface {
	// Evaluate 单表单判定（请求内一次组+主体加载）
	Evaluate(ctx context.Context, member *iammodel.User, formID uint) (*ResolvedFormPermission, error)
	// EvaluateForForms 批量判定（菜单快照等批量执行点）
	EvaluateForForms(ctx context.Context, member *iammodel.User, formIDs []uint) (map[uint]*ResolvedFormPermission, error)
}

// formPermissionEvaluator 判定器实现：判定数据面小（组数 ≤ 50/表、主体行少），
// P1 不引入缓存（§8.3）；单请求一次组+主体加载。
type formPermissionEvaluator struct {
	groups   repository.PermissionGroupRepository
	forms    repository.FormRepository
	versions repository.FormVersionRepository
	access   AccessEvaluator
	subjects PermissionSubjectSource
}

// NewFormPermissionEvaluator 构造判定器（server 装配注入；subjects 为主体
// 解析窄端口，access 为权限集窄端口与鉴权中间件同源）。
func NewFormPermissionEvaluator(
	groups repository.PermissionGroupRepository,
	forms repository.FormRepository,
	versions repository.FormVersionRepository,
	access AccessEvaluator,
	subjects PermissionSubjectSource,
) FormPermissionEvaluator {
	return &formPermissionEvaluator{groups: groups, forms: forms, versions: versions, access: access, subjects: subjects}
}

func (e *formPermissionEvaluator) Evaluate(ctx context.Context, member *iammodel.User, formID uint) (*ResolvedFormPermission, error) {
	if member == nil || member.ID == 0 {
		return nil, httpx.Wrap(apperrors.ErrForbidden, fmt.Errorf("member required for form permission evaluation"))
	}
	result, err := e.evaluateCore(ctx, member.ID, []uint{formID})
	if err != nil {
		return nil, err
	}
	return result[formID], nil
}

func (e *formPermissionEvaluator) EvaluateForForms(
	ctx context.Context, member *iammodel.User, formIDs []uint,
) (map[uint]*ResolvedFormPermission, error) {
	if member == nil || member.ID == 0 {
		return nil, httpx.Wrap(apperrors.ErrForbidden, fmt.Errorf("member required for form permission evaluation"))
	}
	return e.evaluateCore(ctx, member.ID, formIDs)
}

// evaluateCore 批量判定核心：一次加载组/主体/字段清单，按表单拆分结果。
// memberID 经窄端口/权限集重载真实身份，不信任调用方传入的角色快照。
func (e *formPermissionEvaluator) evaluateCore(
	ctx context.Context, memberID uint, formIDs []uint,
) (map[uint]*ResolvedFormPermission, error) {
	if len(formIDs) == 0 {
		return map[uint]*ResolvedFormPermission{}, nil
	}
	admin := false
	if e.access != nil {
		admin = e.access.Permissions(ctx, &iammodel.User{ID: memberID})[formDataAdminPermissionKey]
	}

	// S4/S5 事实源：存在任一权限组行（含禁用组）即收口
	existing, err := e.groups.ExistsByAssetIDs(ctx, model.PermissionAssetTypeForm, formIDs)
	if err != nil {
		return nil, err
	}
	enabledGroups, err := e.groups.ListEnabledByAssetIDs(ctx, model.PermissionAssetTypeForm, formIDs)
	if err != nil {
		return nil, err
	}

	// 主体反查：命中候选 = 启用组（禁用组不授权但维持收口）
	groupByIDs := make(map[uint]*model.AssetPermissionGroup, len(enabledGroups))
	enabledIDs := make([]uint, 0, len(enabledGroups))
	for i := range enabledGroups {
		groupByIDs[enabledGroups[i].ID] = &enabledGroups[i]
		enabledIDs = append(enabledIDs, enabledGroups[i].ID)
	}
	var groupSubjects []model.AssetPermissionGroupSubject
	if len(enabledIDs) > 0 && !admin {
		groupSubjects, err = e.groups.ListSubjectsByGroupIDs(ctx, enabledIDs)
		if err != nil {
			return nil, err
		}
	}
	var departmentIDs, roleIDs []uint
	if len(groupSubjects) > 0 && e.subjects != nil {
		departmentIDs, roleIDs, err = e.subjects.MemberSubject(ctx, memberID)
		if err != nil {
			return nil, err
		}
	}

	// 字段清单批量加载（S7 合并事实源；已发布取快照 schema，未发布回落草稿）
	fieldLists, err := e.loadFieldLists(ctx, formIDs)
	if err != nil {
		return nil, err
	}

	result := make(map[uint]*ResolvedFormPermission, len(formIDs))
	for _, formID := range formIDs {
		fieldList := fieldLists[formID]
		resolved := &ResolvedFormPermission{Admin: admin, Baseline: !existing[formID], fieldList: fieldList}
		if !admin && !resolved.Baseline {
			resolved.Matched = matchGroupsForMember(groupByIDs, groupSubjects, memberID, departmentIDs, roleIDs, fieldList)
		}
		result[formID] = resolved
	}
	return result, nil
}

// matchGroupsForMember 主体命中判定：直接成员命中 ∨ 部门命中（配置部门在
// 成员部门命中集内即命中，「部门含子部门」，语义对齐 TenantProductAccessEvaluator）
// ∨ 角色命中。解析不到的主体不命中（角色/部门删除容错）。
func matchGroupsForMember(
	groupByIDs map[uint]*model.AssetPermissionGroup,
	subjects []model.AssetPermissionGroupSubject,
	memberID uint, departmentIDs, roleIDs []uint,
	fieldList []permissionFieldMeta,
) []MatchedGroup {
	memberDepartmentSet := toUintSet(departmentIDs)
	memberRoleSet := toUintSet(roleIDs)
	matched := make([]MatchedGroup, 0)
	seen := make(map[uint]bool, len(subjects))
	for _, subject := range subjects {
		if seen[subject.GroupID] {
			continue
		}
		group, ok := groupByIDs[subject.GroupID]
		if !ok {
			continue // 启用集与主体集之间的竞态窗口：组已禁用/删除即不授权
		}
		if !permissionSubjectHits(&subject, memberID, memberDepartmentSet, memberRoleSet) {
			continue
		}
		seen[subject.GroupID] = true
		matched = append(matched, buildMatchedGroup(group, fieldList))
	}
	// 稳定输出顺序（组 id 升序为 SQL 默认序，按 code 再排序仅用于结果确定性）
	sort.Slice(matched, func(a, b int) bool { return matched[a].Code < matched[b].Code })
	return matched
}

// permissionSubjectHits 单主体命中判定
func permissionSubjectHits(
	subject *model.AssetPermissionGroupSubject, memberID uint,
	memberDepartmentSet, memberRoleSet map[uint]bool,
) bool {
	switch subject.SubjectType {
	case model.PermissionSubjectMember:
		return subject.SubjectID == memberID
	case model.PermissionSubjectDepartment:
		return memberDepartmentSet[subject.SubjectID]
	case model.PermissionSubjectRole:
		return memberRoleSet[subject.SubjectID]
	default:
		return false
	}
}

// buildMatchedGroup 组四要素投影：操作键集合化 + S7 deny-by-default 字段合并
func buildMatchedGroup(group *model.AssetPermissionGroup, fieldList []permissionFieldMeta) MatchedGroup {
	operations := make(map[string]bool, len(group.Operations))
	for _, op := range group.Operations {
		operations[op] = true
	}
	scope := model.PermissionDataScopeSpec(group.DataScope)
	scope.Normalize()
	return MatchedGroup{
		Code:       group.Code,
		Operations: operations,
		Fields:     mergeGroupFieldMatrix(group.FieldPermissions, fieldList),
		DataScope:  scope,
	}
}

// mergeGroupFieldMatrix S7 deny-by-default 合并：以当前字段清单为基准，矩阵
// 中缺失的字段默认不可见不可编辑；矩阵中已不存在于清单的字段条目忽略。
func mergeGroupFieldMatrix(matrix model.PermissionFieldRules, fieldList []permissionFieldMeta) map[string]FieldPermission {
	configured := make(map[string]model.PermissionFieldRule, len(matrix))
	for _, rule := range matrix {
		configured[rule.Field] = rule
	}
	merged := make(map[string]FieldPermission, len(fieldList))
	for _, field := range fieldList {
		if rule, ok := configured[field.Key]; ok {
			merged[field.Key] = FieldPermission{Visible: rule.Visible, Editable: rule.Editable}
			continue
		}
		merged[field.Key] = FieldPermission{}
	}
	return merged
}

func toUintSet(ids []uint) map[uint]bool {
	set := make(map[uint]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	return set
}

// ---- 字段清单批量加载 ----

// loadFieldLists 批量加载表单当前字段清单：已发布表单取最新快照 schema，
// 未发布回落草稿（同一提取器）。
func (e *formPermissionEvaluator) loadFieldLists(
	ctx context.Context, formIDs []uint,
) (map[uint][]permissionFieldMeta, error) {
	lists := make(map[uint][]permissionFieldMeta, len(formIDs))
	formByID := make(map[uint]*model.Form, len(formIDs))
	versionIDByForm := make(map[uint]uint, len(formIDs))
	for _, formID := range formIDs {
		form, err := e.forms.GetByID(ctx, formID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue // 表单已删除：空清单（调用方先有资产行，防御分支）
			}
			return nil, err
		}
		formByID[formID] = form
		if form.LatestVersionID != nil {
			versionIDByForm[formID] = *form.LatestVersionID
		}
	}

	contentByForm := make(map[uint]model.JSONContent, len(formByID))
	for formID, versionID := range versionIDByForm {
		version, err := e.versions.GetByID(ctx, versionID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue // 快照缺失异常态：回落草稿
			}
			return nil, err
		}
		contentByForm[formID] = version.Content
	}
	for formID, form := range formByID {
		if _, ok := contentByForm[formID]; !ok {
			contentByForm[formID] = form.DraftContent
		}
	}
	for formID, content := range contentByForm {
		fieldList, err := snapshotPermissionFieldList(content)
		if err != nil {
			return nil, err
		}
		lists[formID] = fieldList
	}
	return lists, nil
}

// snapshotPermissionFieldList 快照文档 → 权限域字段清单
func snapshotPermissionFieldList(content model.JSONContent) ([]permissionFieldMeta, error) {
	root := make(map[string]any)
	if err := json.Unmarshal([]byte(content), &root); err != nil {
		return nil, fmt.Errorf("snapshot decode: %w", err)
	}
	return buildPermissionFieldList(root)
}

// ---- 数据范围内存匹配（镜像 §5.2 SQL NULL 语义表） ----

// numericGuardPattern 数值守卫正则（§5.2：不满足守卫的表达式整体返回 NULL，
// 不进入 cast）
var numericGuardPattern = regexp.MustCompile(`^[+-]?([0-9]+\.?[0-9]*|\.[0-9]+)$`)

// permissionScopeMatches 组数据范围 × 记录值匹配：match=all 且（默认）/
// match=any 或；空条件 = 全部数据（S6）。fieldList 为当前字段清单（类型
// 分派事实源；条件字段已不在清单时按 NULL 语义收敛）。
func permissionScopeMatches(
	scope model.PermissionDataScopeSpec, fieldList []permissionFieldMeta, values map[string]any,
) bool {
	if len(scope.Conditions) == 0 {
		return true // S6：未配置 = 全部数据
	}
	index := permissionFieldIndex(fieldList)
	if scope.Match == model.PermissionScopeMatchAny {
		for i := range scope.Conditions {
			if permissionConditionMatches(&scope.Conditions[i], index, values) {
				return true
			}
		}
		return false
	}
	for i := range scope.Conditions {
		if !permissionConditionMatches(&scope.Conditions[i], index, values) {
			return false
		}
	}
	return true
}

// permissionConditionMatches 单条件判定（NULL 语义表）：
//   - eq/gt/gte/lt/lte/contains/in：NULL 不命中；
//   - ne：NULL 不命中（三值逻辑，禁止 COALESCE 放行）；
//   - empty：命中（NULL 即空）；not_empty：不命中；
//   - not_in：NULL 显式命中（模板已显式 V IS NULL OR …，非取反实现）。
func permissionConditionMatches(
	condition *model.PermissionDataCondition, index map[string]permissionFieldMeta, values map[string]any,
) bool {
	field, known := index[condition.Field]
	value, isNull := normalizeConditionValue(&field, known, values[condition.Field])
	switch condition.Operator {
	case "eq":
		return !isNull && compareConditionText(value.text, condition.Value) == 0
	case "ne":
		return !isNull && compareConditionText(value.text, condition.Value) != 0
	case "gt", "gte", "lt", "lte":
		if isNull {
			return false
		}
		order := compareConditionValue(&field, value, condition.Value)
		switch condition.Operator {
		case "gt":
			return order > 0
		case "gte":
			return order >= 0
		case "lt":
			return order < 0
		default:
			return order <= 0
		}
	case "contains":
		if isNull {
			return false
		}
		needle, _ := conditionString(condition.Value)
		if value.isArray {
			for _, element := range value.array {
				if text, ok := element.(string); ok && text == needle {
					return true
				}
			}
			return false
		}
		return strings.Contains(value.text, needle)
	case "in":
		if isNull {
			return false
		}
		if value.isArray {
			return arrayIntersects(value.array, condition.Value)
		}
		return conditionContainsText(condition.Value, value.text)
	case "not_in":
		if isNull {
			return true // 显式命中（非取反实现）
		}
		if value.isArray {
			return !arrayIntersects(value.array, condition.Value)
		}
		return !conditionContainsText(condition.Value, value.text)
	case "empty":
		return isNull || value.text == "" || (value.isArray && len(value.array) == 0)
	case "not_empty":
		return !(isNull || value.text == "" || (value.isArray && len(value.array) == 0))
	default:
		return false
	}
}

// normalizedFieldValue 归一取值结果：标量类经 V(F) 文本；多选类经数组。
type normalizedFieldValue struct {
	text    string
	array   []any
	isArray bool
}

// normalizeConditionValue 行值归一（V(F) 语义 + 类型类守卫）：
//   - 缺键 / JSON null → SQL NULL；
//   - 字段已不在当前清单（类未知）→ SQL NULL（发布阻塞使启用组不会出现该
//     情形，此分支为禁用组遗留条件的防御收敛）；
//   - 标量类：string 原样；数字类经守卫正则（float 值直接保留、字符串须过
//     正则）；其余形状（数组/对象/布尔/形状不符）→ SQL NULL（守卫不过）；
//   - 多选类：全字符串数组 → 数组；其余 → SQL NULL。
func normalizeConditionValue(field *permissionFieldMeta, known bool, raw any) (normalizedFieldValue, bool) {
	if raw == nil {
		return normalizedFieldValue{}, true
	}
	var class permissionFieldClass
	if known {
		class = permissionClassOfWidget(field.WidgetType)
	}
	switch class {
	case permFieldClassMultiOption:
		if array, ok := raw.([]any); ok {
			for _, element := range array {
				if _, isText := element.(string); !isText {
					return normalizedFieldValue{}, true
				}
			}
			return normalizedFieldValue{array: array, isArray: true}, false
		}
		return normalizedFieldValue{}, true
	case permFieldClassNumber:
		switch v := raw.(type) {
		case float64:
			return normalizedFieldValue{text: jsNumber(v)}, false
		case string:
			if numericGuardPattern.MatchString(v) {
				return normalizedFieldValue{text: v}, false
			}
		}
		return normalizedFieldValue{}, true
	case permFieldClassDateTime:
		if text, ok := raw.(string); ok {
			format := field.Format
			if format == "" {
				format = "datetime"
			}
			if pattern, ok := timeShapePatterns[format]; ok && pattern.MatchString(text) {
				return normalizedFieldValue{text: text}, false
			}
		}
		return normalizedFieldValue{}, true
	case permFieldClassText, permFieldClassSingleOption:
		switch v := raw.(type) {
		case string:
			return normalizedFieldValue{text: v}, false
		case float64:
			return normalizedFieldValue{text: jsNumber(v)}, false
		}
		return normalizedFieldValue{}, true
	default:
		// 类未知（字段已删除/类型不支持条件）：按 SQL NULL 收敛
		return normalizedFieldValue{}, true
	}
}

// compareConditionValue 比较 operator 的三态比较（-1/0/1）：数字类按数值，
// 日期类按定宽文本字典序（= 时间序）。
func compareConditionValue(field *permissionFieldMeta, value normalizedFieldValue, conditionValue []any) int {
	if permissionClassOfWidget(field.WidgetType) == permFieldClassNumber {
		left, err := strconv.ParseFloat(value.text, 64)
		if err != nil {
			return 0 // 守卫已过但解析失败（理论不可达）：视同相等不放大
		}
		right, ok := conditionFloat(conditionValue)
		if !ok {
			return 0
		}
		switch {
		case left > right:
			return 1
		case left < right:
			return -1
		default:
			return 0
		}
	}
	// 日期类：定宽零填充文本字典序 = 时间序，与数据库 TimeZone 无关
	right, _ := conditionString(conditionValue)
	return strings.Compare(value.text, right)
}

// compareConditionText 等值比较（eq/ne）：标量文本与条件值文本比较
func compareConditionText(text string, conditionValue []any) int {
	right, _ := conditionString(conditionValue)
	return strings.Compare(text, right)
}

// conditionString 取条件值的单元素文本
func conditionString(conditionValue []any) (string, bool) {
	if len(conditionValue) == 0 {
		return "", false
	}
	if text, ok := conditionValue[0].(string); ok {
		return text, true
	}
	if number, ok := conditionFloat(conditionValue); ok {
		return jsNumber(number), true
	}
	return "", false
}

// conditionFloat 取条件值的单元素数值
func conditionFloat(conditionValue []any) (float64, bool) {
	if len(conditionValue) == 0 {
		return 0, false
	}
	return jsonFloat(conditionValue[0])
}

// conditionContainsText 文本 ∈ 条件值集合
func conditionContainsText(conditionValue []any, text string) bool {
	for _, element := range conditionValue {
		if candidate, ok := element.(string); ok && candidate == text {
			return true
		}
	}
	return false
}

// arrayIntersects 多选数组 ∩ 条件值集合（`?|` 语义）
func arrayIntersects(array []any, conditionValue []any) bool {
	for _, element := range array {
		text, ok := element.(string)
		if !ok {
			continue
		}
		if conditionContainsText(conditionValue, text) {
			return true
		}
	}
	return false
}
