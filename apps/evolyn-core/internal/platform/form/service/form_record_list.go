package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"evolyn/internal/contextx"
	storagepkg "evolyn/internal/engine/data/storage"
	apperrors "evolyn/internal/platform/form"
	"evolyn/internal/platform/form/model"
	"evolyn/internal/platform/form/repository"
	"evolyn/internal/platform/httpx"
	iammodel "evolyn/internal/platform/iam/model"

	"gorm.io/gorm"
)

const (
	defaultRecordListPageSize = 20
	maxRecordListPageSize     = 100
)

// ListRecords applies the user Query DSL and the member's record-level view
// scopes as one database predicate before Count/Offset pagination. The later
// field projection is intentionally a second, per-record permission decision:
// data scopes decide which rows exist; field matrices decide what each row shows.
func (s *formService) ListRecords(ctx context.Context, member *iammodel.User, code string, query model.RecordQueryDocument) (*model.FormRecordPage, error) {
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("tenant context required")
	}
	if member == nil || member.ID == 0 || member.TenantID != tenantID {
		return nil, httpx.Wrap(apperrors.ErrForbidden, fmt.Errorf("member not in tenant %d", tenantID))
	}
	if !s.access.Permissions(ctx, member)["form-records:get"] {
		return nil, httpx.Wrap(apperrors.ErrForbidden, fmt.Errorf("member cannot list form records"))
	}
	form, err := s.loadByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if form.LatestVersionID == nil {
		return nil, httpx.Wrap(apperrors.ErrNotPublished, fmt.Errorf("form %s not published", code))
	}
	version, err := s.versions.GetByID(ctx, *form.LatestVersionID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, httpx.Wrap(apperrors.ErrNotPublished, err)
		}
		return nil, err
	}
	// 000065 之前已存在的不可变发布快照不会被回写 field_mappings；这类
	// 快照只能从其自身的 Content 推导一次只读映射，绝不能根据草稿或记录
	// values 猜字段。新快照始终以冻结列为唯一事实源。
	mappings, err := snapshotFieldMappings(version)
	if err != nil {
		return nil, err
	}
	content := make(map[string]any)
	if err := json.Unmarshal(version.Content, &content); err != nil {
		return nil, fmt.Errorf("published schema decode: %w", err)
	}
	fieldList, err := buildPermissionFieldList(content)
	if err != nil {
		return nil, fmt.Errorf("published field list: %w", err)
	}

	page, pageSize := normalizeRecordListPaging(query.Paging)
	// 物理存储分派（方案 §11）：physical 表单的用户字段谓词解析为物理列
	// 表达式（JOIN tn_fd_* d），系统字段挂信封别名 r；列名唯一事实源是
	// 已应用存储模型。存量 JSONB 表单保持 values JSONB 编译路径。
	opts, listBinding, err := s.physicalListBinding(ctx, form)
	if err != nil {
		return nil, err
	}
	userFilter, err := CompileRecordListQuery(query, mappings, fieldList, opts)
	if err != nil {
		return nil, httpx.Wrap(apperrors.ErrRecordQueryInvalid, err)
	}
	keywordFilter, err := CompileRecordKeyword(query.Keyword, mappings, fieldList, opts)
	if err != nil {
		return nil, httpx.Wrap(apperrors.ErrRecordQueryInvalid, err)
	}
	predicates := []CompiledRecordQuery{userFilter, keywordFilter}
	resolved, err := s.evaluatePermissions(ctx, member, form.ID)
	if err != nil {
		return nil, err
	}
	if resolved != nil && !resolved.Admin && !resolved.Baseline {
		scopes := make([]CompiledRecordQuery, 0, len(resolved.Matched))
		for _, group := range resolved.Matched {
			if !group.Operations[model.PermissionOpView] {
				continue
			}
			scope, cerr := CompilePermissionScopeSQL(group.DataScope, mappings, fieldList, opts)
			if cerr != nil {
				return nil, fmt.Errorf("compile view scope for group %s: %w", group.Code, cerr)
			}
			scopes = append(scopes, scope)
		}
		if len(scopes) == 0 {
			return &model.FormRecordPage{Items: []model.FormRecordDTO{}, Page: page, PageSize: pageSize}, nil
		}
		predicates = append(predicates, joinCompiled(scopes, " OR "))
	}
	predicate := joinCompiled(predicates, " AND ")
	// 排序先行编译：仅系统字段可排序（CompileRecordListSorts 白名单），非法
	// 排序与非法筛选同样以 FORM_RECORD_QUERY_INVALID 拒绝。
	orderBy, err := CompileRecordListSorts(query.Sorts, opts)
	if err != nil {
		return nil, httpx.Wrap(apperrors.ErrRecordQueryInvalid, err)
	}
	params := repository.RecordListParams{
		FormID: form.ID, Page: page, PageSize: pageSize,
		Where: predicate.Where, Args: predicate.Args, OrderBy: orderBy,
	}

	// physical 查询在只读事务内执行：动态表 RLS 消费事务级 SET LOCAL
	// app.current_tenant（方案 §12），非事务会话将按 fail-closed 不可见。
	var (
		items []model.FormRecordDTO
		total int64
	)
	if listBinding != nil {
		params.TenantID = tenantID
		err = s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
			rows, t, lerr := s.physical.ListJoinControlled(tctx, params, *listBinding)
			if lerr != nil {
				return lerr
			}
			if lerr = s.hydratePhysicalListChildren(tctx, rows, listBinding.Children); lerr != nil {
				return lerr
			}
			total = t
			items = assembleRecordItems(rowsOfPhysical(rows), mappings, resolved)
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		records, t, err := s.records.ListControlled(ctx, params)
		if err != nil {
			return nil, err
		}
		total = t
		items = assembleRecordItems(rowsOfLegacy(records), mappings, resolved)
	}
	if err := s.hydrateMemberReferences(ctx, items, content); err != nil {
		return nil, err
	}
	return &model.FormRecordPage{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

// physicalListBinding 解析物理模式编译选项与 JOIN 绑定（列名唯一事实源是
// 已应用存储模型）；legacy 表单返回零值选项与 nil 绑定。
func (s *formService) physicalListBinding(ctx context.Context, form *model.Form) (RecordQueryCompileOptions, *repository.PhysicalListBinding, error) {
	binding, err := s.loadStorageByForm(ctx, form.ID)
	if err != nil {
		return RecordQueryCompileOptions{}, nil, err
	}
	if binding == nil {
		return RecordQueryCompileOptions{}, nil, nil
	}
	if s.physical == nil || s.schemaVersions == nil {
		return RecordQueryCompileOptions{}, nil, fmt.Errorf("physical storage pipeline is not configured")
	}
	// 与 resolvePhysicalContext 同口径：结构变更在途（PUBLISHING/FAILED）按
	// 已应用模型继续服务（物理结构是历次已应用字段的并集，列表投影/谓词
	// 使用的列集合 ⊆ 已应用模型列）；仅从未应用过模型（首次发布 DDL 在
	// 途，物理表尚不存在）才拒绝列表。
	applied, err := s.loadAppliedModel(ctx, binding)
	if err != nil {
		return RecordQueryCompileOptions{}, nil, err
	}
	if applied == nil {
		return RecordQueryCompileOptions{}, nil, httpx.Wrap(apperrors.ErrStorageNotReady,
			fmt.Errorf("form %s has no applied storage model", form.Code))
	}
	physicalColumns := make(map[string]string, len(applied.Columns))
	columns := make([]repository.PhysicalColumn, 0, len(applied.Columns))
	for _, column := range activeColumns(applied.Columns) {
		physicalColumns[column.WidgetName] = column.ColumnName()
		columns = append(columns, repository.PhysicalColumn{WidgetName: column.WidgetName, Column: column})
	}
	children := make([]storagepkg.ChildTableSpec, 0, len(applied.Children))
	for _, child := range applied.Children {
		if !child.Deprecated {
			children = append(children, child)
		}
	}
	return RecordQueryCompileOptions{Physical: true, PhysicalColumns: physicalColumns},
		&repository.PhysicalListBinding{
			TableName: binding.PhysicalTable,
			Columns:   columns,
			Children:  children,
		}, nil
}

// hydratePhysicalListChildren 在父记录分页完成后，以「每张子表一次查询」批量
// 水合子表单值。子表绝不直接 JOIN 到父记录分页 SQL，避免一对多 JOIN 让一条
// 父记录被拆成多行、破坏 count/offset 以及数据管理的合并表头行组。
func (s *formService) hydratePhysicalListChildren(ctx context.Context, rows []repository.PhysicalRecordRow, children []storagepkg.ChildTableSpec) error {
	if len(rows) == 0 || len(children) == 0 {
		return nil
	}
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return fmt.Errorf("tenant context required")
	}
	parentRecordIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		parentRecordIDs = append(parentRecordIDs, row.Record.ID)
	}
	for _, child := range children {
		byParent, err := s.physical.ReadChildRowsByParentIDs(
			ctx, child.TableName, tenantID, parentRecordIDs, child.Columns,
		)
		if err != nil {
			return err
		}
		for rowIndex := range rows {
			childRows := byParent[rows[rowIndex].Record.ID]
			items := make([]any, 0, len(childRows))
			for _, childRow := range childRows {
				items = append(items, childRow)
			}
			rows[rowIndex].Values[child.WidgetName] = items
		}
	}
	return nil
}

// envelopeRow 列表行的统一读取视图（legacy/physical 两路同构组装）。
type envelopeRow struct {
	record model.FormRecord
	values map[string]any
}

func rowsOfPhysical(rows []repository.PhysicalRecordRow) []envelopeRow {
	out := make([]envelopeRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, envelopeRow{record: row.Record, values: row.Values})
	}
	return out
}

func rowsOfLegacy(records []model.FormRecord) []envelopeRow {
	out := make([]envelopeRow, 0, len(records))
	for _, record := range records {
		values := make(map[string]any)
		if len(record.Values) > 0 && string(record.Values) != "null" {
			if err := json.Unmarshal(record.Values, &values); err != nil {
				values = map[string]any{}
			}
		}
		out = append(out, envelopeRow{record: record, values: values})
	}
	return out
}

// assembleRecordRows 出网逐行二段裁决（两路共用）：行级 view 命中后，
// 字段矩阵裁剪 + 快照映射白名单 + 系统字段直出。
func assembleRecordItems(rows []envelopeRow, mappings []SnapshotFieldMapping, resolved *ResolvedFormPermission) []model.FormRecordDTO {
	items := make([]model.FormRecordDTO, 0, len(rows))
	allowed := make(map[string]bool, len(mappings))
	for _, mapping := range mappings {
		allowed[mapping.WidgetName] = true
	}
	for _, row := range rows {
		values := row.values
		if resolved != nil {
			fields := resolved.FieldsFor(model.PermissionOpView, values)
			for key := range allowed {
				permission, exists := fields[key]
				if !exists || !permission.Visible {
					delete(values, key)
				}
			}
		}
		for key := range values {
			if !allowed[key] {
				delete(values, key)
			}
		}
		// 系统字段不参与字段矩阵裁剪：行级可见即可见；提交人快照为空（昵称
		// 未设置的边缘态或 000067 前未回填命中）回落固定文案保证可读。
		submittedByName := row.record.SubmittedByName
		if strings.TrimSpace(submittedByName) == "" {
			submittedByName = "成员"
		}
		items = append(items, model.FormRecordDTO{
			WorkflowInstanceNo:  row.record.WorkflowInstanceNo,
			WorkflowStatus:      row.record.WorkflowStatus,
			WorkflowUpdatedAt:   row.record.WorkflowUpdatedAt,
			ID:                  row.record.ID,
			Values:              values,
			SubmittedByMemberID: row.record.SubmittedByMemberID,
			SubmittedByName:     submittedByName,
			SubmittedAt:         row.record.SubmittedAt,
			UpdatedAt:           row.record.UpdatedAt,
			UpdatedByMemberID:   row.record.UpdatedByMemberID,
			UpdatedByName:       strings.TrimSpace(row.record.UpdatedByName),
		})
	}
	return items
}

func snapshotFieldMappings(version *model.FormVersion) ([]SnapshotFieldMapping, error) {
	mappings := make([]SnapshotFieldMapping, 0)
	if len(version.FieldMappings) > 0 && string(version.FieldMappings) != "[]" {
		if err := json.Unmarshal(version.FieldMappings, &mappings); err != nil {
			return nil, fmt.Errorf("published field mappings decode: %w", err)
		}
		return mappings, nil
	}
	content := make(map[string]any)
	if err := json.Unmarshal(version.Content, &content); err != nil {
		return nil, fmt.Errorf("legacy published schema decode: %w", err)
	}
	return ExtractSnapshotFieldMappings(content), nil
}

func normalizeRecordListPaging(paging model.RecordQueryPaging) (int, int) {
	page, pageSize := paging.Page, paging.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultRecordListPageSize
	}
	if pageSize > maxRecordListPageSize {
		pageSize = maxRecordListPageSize
	}
	return page, pageSize
}
