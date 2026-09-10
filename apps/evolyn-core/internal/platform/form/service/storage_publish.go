// 物理表存储发布编排（方案 §4.2/§7/§8）：表名分配、快照→物理模型构建、
// Diff 与异步 DDL Job 创建（202）、Job 查询/重试。发布时序：
//
//	事务内：锁定 form/storage → 校验草稿 → 创建不可变 FormVersion
//	→ 后端从快照生成 StorageModel，与已应用模型 Diff
//	→ 写 SchemaVersion + DDL Job，Storage=PUBLISHING → 返回 202 + jobId
//
// 不涉及物理变更的发布同步完成（MarkPublished 直推）；是否同步由服务端
// 生成的 Diff 决定，不接受客户端声明。
package service

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"

	"evolyn/internal/contextx"
	storagepkg "evolyn/internal/engine/data/storage"
	kernel "evolyn/internal/model"
	auditservice "evolyn/internal/platform/audit/service"
	apperrors "evolyn/internal/platform/form"
	"evolyn/internal/platform/form/model"
	"evolyn/internal/platform/httpx"
	iammodel "evolyn/internal/platform/iam/model"

	"gorm.io/gorm"
)

// tableNameAttempts 表名防重重试上限（方案 §7：先插入元数据，唯一冲突则
// 重试生成，最多五次；绝不先查 PostgreSQL catalog 再建表）。
const tableNameAttempts = 5

// base36Alphabet 表名随机段与 app4 段共用的字符集（小写 base36）。
const base36Alphabet = "0123456789abcdefghijklmnopqrstuvwxyz"

// newPhysicalTableName 分配动态物理表名：tn_fd_<app4>_<rand6> /
// tn_fc_<app4>_<rand6>。app4 是应用内部编号 base36 末四位（仅供运维识别，
// 绝不作为唯一性或归属判断）；rand6 为密码学安全随机源生成的小写字母
// 数字串（组合空间 ≈21.8 亿）。表名生成后永久不变。
func newPhysicalTableName(prefix string, appID uint) (string, error) {
	app4 := big.NewInt(int64(appID)).Text(36)
	for len(app4) < 4 {
		app4 = "0" + app4
	}
	if len(app4) > 4 {
		app4 = app4[len(app4)-4:]
	}
	randChars := make([]byte, 6)
	for i := range randChars {
		index, err := rand.Int(rand.Reader, big.NewInt(int64(len(base36Alphabet))))
		if err != nil {
			return "", fmt.Errorf("generate table name: %w", err)
		}
		randChars[i] = base36Alphabet[index.Int64()]
	}
	return prefix + app4 + "_" + string(randChars), nil
}

// attachPhysicalStorage 创建表单事务内建立 physical 存储绑定（新建表单
// 固定 physical，不接收任何存储模式入参）。绑定就绪即 READY——物理表本体
// 在首次发布时由 DDL Job 创建。
func (s *formService) attachPhysicalStorage(ctx context.Context, tenantID uint, form *model.Form, appID uint) error {
	if s.storages == nil {
		return fmt.Errorf("physical storage repository is not configured")
	}
	var lastErr error
	for attempt := 0; attempt < tableNameAttempts; attempt++ {
		name, err := newPhysicalTableName("tn_fd_", appID)
		if err != nil {
			return err
		}
		binding := &model.FormStorage{
			FormID:        form.ID,
			Backend:       model.StorageBackendPhysical,
			PhysicalTable: name,
			State:         model.StorageStateReady,
			StorageMetaColumns: model.StorageMetaColumns{
				TenantID: tenantID,
			},
		}
		created, conflict, err := s.storages.Create(ctx, binding)
		if err != nil {
			return err
		}
		if created {
			return nil
		}
		if conflict {
			lastErr = fmt.Errorf("table name %s already taken", name)
			continue
		}
	}
	return fmt.Errorf("allocate physical table name: %w", lastErr)
}

// loadStorageByForm 加载存储绑定；无绑定（存量 JSONB 表单）返回 nil。
func (s *formService) loadStorageByForm(ctx context.Context, formID uint) (*model.FormStorage, error) {
	if s.storages == nil {
		return nil, nil
	}
	binding, err := s.storages.GetByFormID(ctx, formID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return binding, nil
}

// loadAppliedModel 加载已应用物理模型（无已应用版本返回 nil）。
func (s *formService) loadAppliedModel(ctx context.Context, binding *model.FormStorage) (*storagepkg.StorageModel, error) {
	if binding == nil || binding.AppliedSchemaVersionID == nil {
		return nil, nil
	}
	version, err := s.schemaVersions.GetByID(ctx, *binding.AppliedSchemaVersionID)
	if err != nil {
		return nil, err
	}
	var parsed storagepkg.StorageModel
	if err := decodeJSONB(version.Model, &parsed); err != nil {
		return nil, fmt.Errorf("applied storage model decode: %w", err)
	}
	if err := parsed.Validate(); err != nil {
		return nil, fmt.Errorf("applied storage model invalid: %w", err)
	}
	return &parsed, nil
}

// buildTargetStorageModel 从发布快照构建目标物理模型。appliedChildren 提供
// 既有子表的 fieldId→表名继承（表名永不变更）；新子表经 namer 分配。
// 返回的 issues 为发布期必须拒绝的控件清单（FORM_STORAGE_UNSUPPORTED_FIELD）。
func buildTargetStorageModel(
	content map[string]any,
	parentTable string,
	appliedChildren map[string]string,
	namer func(fieldID string) (string, error),
) (*storagepkg.StorageModel, []SchemaIssue, error) {
	itemsAny, ok := documentItems(content)
	if !ok {
		return nil, nil, fmt.Errorf("快照缺少 items")
	}
	target := &storagepkg.StorageModel{TableName: parentTable}
	issues := make([]SchemaIssue, 0)
	appendColumn := func(scope string, widget map[string]any, itemPath string) (storagepkg.ColumnSpec, bool) {
		fieldID, _ := widget["fieldId"].(string)
		name, _ := widget["widgetName"].(string)
		widgetType, _ := widget["type"].(string)
		if err := storagepkg.ValidateFieldID(fieldID); err != nil {
			issues = append(issues, SchemaIssue{Path: itemPath + ".widget.fieldId", Message: "字段缺少不可变标识 fieldId"})
			return storagepkg.ColumnSpec{}, false
		}
		format, _ := widget["format"].(string)
		kind, supported := storagepkg.KindOf(widgetType, format)
		if !supported {
			issues = append(issues, SchemaIssue{
				Path:    itemPath + ".widget.type",
				Message: fmt.Sprintf("控件「%s」尚不支持物理存储，请先移除或等待能力开放", widgetType),
			})
			return storagepkg.ColumnSpec{}, false
		}
		spec := storagepkg.ColumnSpec{
			FieldID:    fieldID,
			WidgetName: name,
			WidgetType: widgetType,
			Kind:       kind,
			Type:       storagepkg.ColumnTypeOf(kind),
		}
		if kind == storagepkg.KindDecimal {
			// 数值字段族：物理列 NUMERIC(p,s)，显式配置优先、缺省按类型
			// 默认解析（numeric_field.go，与提交终审/TS 镜像共用）；精度
			// 修饰纳入类型冲突比较，发布后不可变。
			spec.Precision, spec.Scale = resolveNumericColumnSpec(widgetType, widget)
		}
		return spec, true
	}
	for itemIndex, rawItem := range itemsAny {
		item, ok := rawItem.(map[string]any)
		if !ok {
			continue
		}
		widget, _ := item["widget"].(map[string]any)
		widgetType, _ := widget["type"].(string)
		itemPath := fmt.Sprintf("content.items[%d]", itemIndex)
		if storagepkg.NoColumnWidgetType(widgetType) {
			continue
		}
		if widgetType == "subform" {
			child, childIssues, err := buildChildTableSpec(widget, itemPath, appliedChildren, namer, appendColumn)
			if err != nil {
				return nil, nil, err
			}
			issues = append(issues, childIssues...)
			if child != nil {
				target.Children = append(target.Children, *child)
			}
			continue
		}
		column, ok := appendColumn("parent", widget, itemPath)
		if !ok {
			continue
		}
		target.Columns = append(target.Columns, column)
	}
	if len(issues) > 0 {
		return nil, issues, nil
	}
	if err := target.Validate(); err != nil {
		return nil, nil, fmt.Errorf("目标物理模型非法: %w", err)
	}
	return target, nil, nil
}

// buildChildTableSpec 子表单物理子表规格：子字段列由子字段各自 fieldId 推导，
// 不与父表共享可变名称。
func buildChildTableSpec(
	widget map[string]any,
	itemPath string,
	appliedChildren map[string]string,
	namer func(fieldID string) (string, error),
	appendColumn func(scope string, widget map[string]any, itemPath string) (storagepkg.ColumnSpec, bool),
) (*storagepkg.ChildTableSpec, []SchemaIssue, error) {
	fieldID, _ := widget["fieldId"].(string)
	name, _ := widget["widgetName"].(string)
	if err := storagepkg.ValidateFieldID(fieldID); err != nil {
		//nolint:nilerr // 非法输入按 issues 出网（发布期拒绝），不是执行错误
		return nil, []SchemaIssue{{Path: itemPath + ".widget.fieldId", Message: "子表单缺少不可变标识 fieldId"}}, nil
	}
	tableName, inherited := appliedChildren[fieldID]
	if !inherited {
		assigned, err := namer(fieldID)
		if err != nil {
			return nil, nil, err
		}
		tableName = assigned
	}
	child := &storagepkg.ChildTableSpec{FieldID: fieldID, WidgetName: name, TableName: tableName}
	issues := make([]SchemaIssue, 0)
	childrenAny, _ := widget["items"].([]any)
	for childIndex, rawChild := range childrenAny {
		childItem, ok := rawChild.(map[string]any)
		if !ok {
			continue
		}
		childWidget, _ := childItem["widget"].(map[string]any)
		childType, _ := childWidget["type"].(string)
		childPath := fmt.Sprintf("%s.widget.items[%d]", itemPath, childIndex)
		if storagepkg.NoColumnWidgetType(childType) {
			continue
		}
		column, ok := appendColumn("child", childWidget, childPath)
		if !ok {
			continue
		}
		child.Columns = append(child.Columns, column)
	}
	return child, issues, nil
}

// publishStorageModel 发布事务内的物理模型编排：BUSY 检查 → 目标模型构建 →
// Diff → 同步推进或异步 Job。返回非 nil JobID 表示异步发布（202）；nil 表示
// 无物理变更，由调用方同步推进发布指针。
func (s *formService) publishStorageModel(
	tctx context.Context,
	form *model.Form,
	binding *model.FormStorage,
	content map[string]any,
	created *model.FormVersion,
) (*uint, error) {
	tenantID, ok := contextx.TenantIDFromContext(tctx)
	if !ok {
		return nil, fmt.Errorf("tenant context required")
	}
	// 同一表单任一时刻只能有一个待执行/执行中的物理模型 Job
	active, err := s.jobs.HasActiveByStorage(tctx, binding.ID)
	if err != nil {
		return nil, err
	}
	if active {
		return nil, httpx.Wrap(apperrors.ErrStorageBusy,
			fmt.Errorf("form %s has an active storage ddl job", form.Code))
	}
	// physical 表单必须以 v8+ 协议发布（fieldId 契约冻结前置）
	if created.ProtocolVersion < model.FieldIdentityProtocolVersion {
		return nil, httpx.Wrap(apperrors.ErrStorageModelInvalid,
			fmt.Errorf("form %s protocol %d lacks fieldId contract", form.Code, created.ProtocolVersion))
	}

	applied, err := s.loadAppliedModel(tctx, binding)
	if err != nil {
		return nil, err
	}
	appliedChildren := map[string]string{}
	if applied != nil {
		for _, child := range applied.Children {
			appliedChildren[child.FieldID] = child.TableName
		}
	}
	// 新子表名分配：优先以已持久化映射（000071）继承——同一 fieldId 的映射行
	// 在首次分配时落库，此后发布（含失败重发布）永久复用；全新 fieldId 经
	//「生成名 → 插映射行 → 唯一冲突换名重试」防重，绝不先查 catalog 再建表。
	if s.storageChildren != nil {
		persisted, err := s.storageChildren.ListByStorage(tctx, binding.ID)
		if err != nil {
			return nil, err
		}
		for _, child := range persisted {
			if _, exists := appliedChildren[child.ParentFieldID]; !exists {
				appliedChildren[child.ParentFieldID] = child.PhysicalTable
			}
		}
	}
	appID := form.ApplicationID
	namer := func(fieldID string) (string, error) {
		if s.storageChildren == nil {
			// 映射仓储未装配（单测桩）：退化为纯随机名（无防重，仅测试语义）
			return newPhysicalTableName("tn_fc_", appID)
		}
		var lastErr error
		for attempt := 0; attempt < tableNameAttempts; attempt++ {
			name, err := newPhysicalTableName("tn_fc_", appID)
			if err != nil {
				return "", err
			}
			created, conflict, err := s.storageChildren.Create(tctx, &model.FormStorageChild{
				StorageID:     binding.ID,
				ParentFieldID: fieldID,
				PhysicalTable: name,
				StorageMetaColumns: model.StorageMetaColumns{
					TenantID: tenantID,
				},
			})
			if err != nil {
				return "", err
			}
			if created {
				return name, nil
			}
			if conflict {
				lastErr = fmt.Errorf("child table name %s already taken", name)
				continue
			}
		}
		return "", fmt.Errorf("allocate child table name: %w", lastErr)
	}
	target, issues, err := buildTargetStorageModel(content, binding.PhysicalTable, appliedChildren, namer)
	if err != nil {
		return nil, httpx.Wrap(apperrors.ErrStorageModelInvalid, err)
	}
	if len(issues) > 0 {
		return nil, httpx.Wrap(apperrors.ErrStorageUnsupportedField.WithData(map[string]any{"issues": issues}),
			fmt.Errorf("form %s publish blocked: %d unsupported physical field(s)", form.Code, len(issues)))
	}
	diff, err := storagepkg.Diff(applied, target)
	if err != nil {
		return nil, httpx.Wrap(apperrors.ErrStorageModelInvalid, err)
	}
	if diff.TypeConflict != nil {
		fields := strings.Split(diff.TypeConflict.FieldID, ":")
		widgetName := fields[0]
		for _, column := range target.Columns {
			if column.FieldID == fields[0] {
				widgetName = column.WidgetName
			}
		}
		return nil, httpx.Wrap(apperrors.ErrStorageTypeChangeUnsupported.WithData(map[string]any{"fields": []string{widgetName}}),
			diff.TypeConflict)
	}
	modelJSON, planJSON, err := marshalStorageArtifacts(&diff.MergedModel, diff.Plan)
	if err != nil {
		return nil, err
	}
	schemaVersion := &model.FormStorageSchemaVersion{
		StorageID:     binding.ID,
		FormVersionID: created.ID,
		Model:         modelJSON,
		Plan:          planJSON,
		Checksum:      diff.Plan.Checksum,
		State:         model.SchemaVersionPending,
		StorageMetaColumns: model.StorageMetaColumns{
			TenantID: tenantID,
		},
	}
	if _, err := s.schemaVersions.Create(tctx, schemaVersion); err != nil {
		return nil, err
	}
	if diff.Plan.Empty {
		// 无物理变更：版本直接置 APPLIED 并同步推进发布指针（200 路径）
		if err := s.schemaVersions.MarkApplied(tctx, schemaVersion.ID); err != nil {
			return nil, err
		}
		if err := s.storages.MarkReady(tctx, binding.ID, schemaVersion.ID); err != nil {
			return nil, err
		}
		return nil, nil
	}
	job := &model.FormDDLJob{
		StorageSchemaVersionID: schemaVersion.ID,
		Status:                 model.DDLJobPending,
		NextAttemptAt:          kernel.JSONTime(time.Now()),
		StorageMetaColumns: model.StorageMetaColumns{
			TenantID: tenantID,
		},
	}
	createdJob, err := s.jobs.Create(tctx, job)
	if err != nil {
		return nil, err
	}
	if err := s.storages.MarkPublishing(tctx, binding.ID); err != nil {
		return nil, err
	}
	jobID := createdJob.ID
	return &jobID, nil
}

// lockStorageByForm 发布事务内锁定存储绑定行（SELECT FOR UPDATE，防同表单
// 并发发布的 BUSY 竞态窗口）；无绑定（存量 JSONB 表单/未装配）返回 nil。
func (s *formService) lockStorageByForm(ctx context.Context, form *model.Form) (*model.FormStorage, error) {
	if s.storages == nil {
		return nil, nil
	}
	binding, err := s.storages.LockByForm(ctx, form.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return binding, nil
}

// marshalStorageArtifacts 模型与计划序列化为 JSONB 持久化形态。
func marshalStorageArtifacts(target *storagepkg.StorageModel, plan storagepkg.Plan) (model.JSONContent, model.JSONContent, error) {
	modelBytes, err := json.Marshal(target)
	if err != nil {
		return nil, nil, fmt.Errorf("storage model marshal: %w", err)
	}
	planBytes, err := json.Marshal(plan)
	if err != nil {
		return nil, nil, fmt.Errorf("storage plan marshal: %w", err)
	}
	return model.JSONContent(modelBytes), model.JSONContent(planBytes), nil
}

// ---- DDL Job 查询与重试（§14 API） ----

// GetStorageJob 查询 DDL 发布状态与受控失败信息（沿用表单管理权限）。
func (s *formService) GetStorageJob(ctx context.Context, member *iammodel.User, code string, jobID uint) (*model.StorageJobDetail, error) {
	if !s.access.Permissions(ctx, member)["forms:get"] {
		return nil, httpx.Wrap(apperrors.ErrForbidden, fmt.Errorf("member cannot view storage job of form %s", code))
	}
	form, err := s.loadByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	job, version, err := s.loadStorageJobChain(ctx, form, jobID)
	if err != nil {
		return nil, err
	}
	detail := &model.StorageJobDetail{
		JobID:            job.ID,
		Status:           string(job.Status),
		RetryCount:       job.RetryCount,
		NextAttemptAt:    job.NextAttemptAt,
		LastErrorCode:    job.LastErrorCode,
		PublishedVersion: version.VersionNo,
		SchemaRevision:   fmt.Sprintf("%d", version.SchemaRevision),
	}
	if job.StartedAt != nil {
		started := *job.StartedAt
		detail.StartedAt = &started
	}
	if job.FinishedAt != nil {
		finished := *job.FinishedAt
		detail.FinishedAt = &finished
	}
	return detail, nil
}

// RetryStorageJob 管理员重试失败 Job：仅接受 FAILED→PENDING 复位，不接受
// 新模型；重试成功后由 Worker 推进发布指针。
func (s *formService) RetryStorageJob(ctx context.Context, member *iammodel.User, code string, jobID uint) (*model.StorageJobDetail, error) {
	if !s.access.Permissions(ctx, member)["forms:update"] {
		return nil, httpx.Wrap(apperrors.ErrForbidden, fmt.Errorf("member cannot retry storage job of form %s", code))
	}
	form, err := s.loadByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	job, _, err := s.loadStorageJobChain(ctx, form, jobID)
	if err != nil {
		return nil, err
	}
	if job.Status != model.DDLJobFailed {
		return nil, httpx.Wrap(apperrors.ErrStorageDDLFailed,
			fmt.Errorf("storage job %d is %s, only FAILED can be retried", jobID, job.Status))
	}
	// 复位链：Job 回 PENDING + SchemaVersion 回 PENDING + Storage 回 PUBLISHING，
	// 与 Worker 失败路径的状态镜像对称（同一事务）。
	var retried bool
	if err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		if err := s.jobs.ResetForRetry(tctx, job.ID, time.Now()); err != nil {
			return err
		}
		if err := s.schemaVersions.ResetForRetry(tctx, job.StorageSchemaVersionID); err != nil {
			return err
		}
		binding, err := s.storages.GetByFormID(tctx, form.ID)
		if err != nil {
			return err
		}
		if err := s.storages.MarkPublishing(tctx, binding.ID); err != nil {
			return err
		}
		retried = true
		return nil
	}); err != nil {
		return nil, err
	}
	if !retried {
		return nil, httpx.Wrap(apperrors.ErrStorageDDLFailed, fmt.Errorf("storage job %d retry did not apply", jobID))
	}
	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "form", Action: "retry-storage-ddl", ResourceType: "form",
			ResourceID: form.Code,
			After:      map[string]any{"jobId": jobID},
			TargetName: form.Name,
		})
	}
	return s.GetStorageJob(ctx, member, code, jobID)
}

// loadStorageJobChain 加载 Job 并复核归属表单（经 SchemaVersion → FormVersion）。
func (s *formService) loadStorageJobChain(ctx context.Context, form *model.Form, jobID uint) (*model.FormDDLJob, *model.FormVersion, error) {
	job, err := s.jobs.GetByID(ctx, jobID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, httpx.Wrap(apperrors.ErrStorageJobNotFound, err)
		}
		return nil, nil, err
	}
	version, err := s.schemaVersions.GetByID(ctx, job.StorageSchemaVersionID)
	if err != nil {
		return nil, nil, err
	}
	if version.FormVersionID == 0 {
		return nil, nil, httpx.Wrap(apperrors.ErrStorageJobNotFound, fmt.Errorf("broken job chain %d", jobID))
	}
	formVersion, err := s.versions.GetByID(ctx, version.FormVersionID)
	if err != nil {
		return nil, nil, err
	}
	if formVersion.FormID != form.ID {
		return nil, nil, httpx.Wrap(apperrors.ErrStorageJobNotFound,
			fmt.Errorf("storage job %d belongs to another form", jobID))
	}
	return job, formVersion, nil
}

// decodeJSONB 统一 JSONB 列解码助手。
func decodeJSONB(raw model.JSONContent, target any) error {
	if len(raw) == 0 {
		return nil
	}
	return json.Unmarshal([]byte(raw), target)
}

// validateFieldIdentityFrozen 字段身份冻结校验（方案 §4.1 契约冻结）：比对
// 新草稿与最新已发布快照——fieldId 与 widgetName 的双向绑定不可变（同
// fieldId 不得换 widgetName，同 widgetName 不得换 fieldId）；字段消失合法
// （删除=弃用），新字段合法。快照协议 < v8 时不校验（历史快照无 fieldId，
// 首次 v8 发布建立物理身份）。
func (s *formService) validateFieldIdentityFrozen(ctx context.Context, form *model.Form, draft model.JSONContent) error {
	current, err := s.versions.GetByID(ctx, *form.LatestVersionID)
	if err != nil {
		return err
	}
	if current.ProtocolVersion < model.FieldIdentityProtocolVersion {
		return nil
	}
	currentContent := make(map[string]any)
	if err := json.Unmarshal([]byte(current.Content), &currentContent); err != nil {
		return fmt.Errorf("published snapshot decode: %w", err)
	}
	draftContent := make(map[string]any)
	if err := json.Unmarshal([]byte(draft), &draftContent); err != nil {
		return fmt.Errorf("draft decode: %w", err)
	}
	publishedIDByName, publishedNameByID, err := fieldIdentityIndex(currentContent)
	if err != nil {
		return err
	}
	draftIDByName, draftNameByID, err := fieldIdentityIndex(draftContent)
	if err != nil {
		return err
	}
	conflicts := make([]string, 0)
	for fieldID, name := range publishedNameByID {
		if draftName, exists := draftNameByID[fieldID]; exists && draftName != name {
			conflicts = append(conflicts, name)
		}
	}
	for name, fieldID := range publishedIDByName {
		if draftID, exists := draftIDByName[name]; exists && draftID != fieldID {
			if !containsStringIn(conflicts, name) {
				conflicts = append(conflicts, name)
			}
		}
	}
	if len(conflicts) > 0 {
		sort.Strings(conflicts)
		return httpx.Wrap(apperrors.ErrFieldIdentityFrozen.WithData(map[string]any{"fields": conflicts}),
			fmt.Errorf("form %s field identity changed: %v", form.Code, conflicts))
	}
	return nil
}

// fieldIdentityIndex 提取快照的 widgetName↔fieldId 双向索引（顶层与子表单
// 子项全部参与；布局/按钮无 fieldId 跳过）。
func fieldIdentityIndex(content map[string]any) (map[string]string, map[string]string, error) {
	itemsAny, ok := documentItems(content)
	if !ok {
		return nil, nil, fmt.Errorf("快照缺少 items")
	}
	byName := make(map[string]string)
	ByID := make(map[string]string)
	// 先声明后赋值以支持闭包递归（子表单子项递归收集）
	var collect func(items []any) error
	collect = func(items []any) error {
		for _, rawItem := range items {
			item, ok := rawItem.(map[string]any)
			if !ok {
				continue
			}
			widget, _ := item["widget"].(map[string]any)
			name, _ := widget["widgetName"].(string)
			widgetType, _ := widget["type"].(string)
			fieldID, _ := widget["fieldId"].(string)
			if storagepkg.NoColumnWidgetType(widgetType) {
				continue
			}
			if name == "" || fieldID == "" {
				continue
			}
			byName[name] = fieldID
			ByID[fieldID] = name
			if widgetType == "subform" {
				children, _ := widget["items"].([]any)
				if err := collect(children); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := collect(itemsAny); err != nil {
		return nil, nil, err
	}
	return byName, ByID, nil
}

func containsStringIn(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

// WorkflowProjectionRecalibrator 投影校准执行端口（装配层由 workflow 侧
// FormProjector 适配；form 域不依赖 workflow 实现）。
type WorkflowProjectionRecalibrator interface {
	RecalibrateByForm(ctx context.Context, formID uint) (int, error)
}

// UseProjectionRecalibrator 注入校准端口（装配期一次性调用）。
func (s *formService) UseProjectionRecalibrator(recalibrator WorkflowProjectionRecalibrator) {
	s.projectionRecalibrator = recalibrator
}

// ProjectionRecalibratorInjector 装配期注入能力（可选）。
type ProjectionRecalibratorInjector interface {
	UseProjectionRecalibrator(recalibrator WorkflowProjectionRecalibrator)
}

// RecalibrateWorkflowProjection 管理员校准表单记录的流程状态投影（方案
// §10.2）：以 wf_instance 为源回填单号/状态/更新时间，仅修复投影、不反向
// 修改流程实例；用于上线回填、故障修复与一致性巡检。
func (s *formService) RecalibrateWorkflowProjection(ctx context.Context, member *iammodel.User, code string) (*model.WorkflowProjectionRecalibrateResult, error) {
	if !s.access.Permissions(ctx, member)["forms:update"] {
		return nil, httpx.Wrap(apperrors.ErrForbidden,
			fmt.Errorf("member cannot recalibrate workflow projection of form %s", code))
	}
	form, err := s.loadByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if s.projectionRecalibrator == nil {
		return nil, httpx.Wrap(apperrors.ErrWorkflowProjectionInvalid,
			fmt.Errorf("projection recalibrator is not configured"))
	}
	count, err := s.projectionRecalibrator.RecalibrateByForm(ctx, form.ID)
	if err != nil {
		return nil, err
	}
	if s.audit != nil {
		appID, appCode, appName := s.appSnapshot(ctx, form.ApplicationID)
		s.audit.Record(ctx, auditservice.Entry{
			Module: "form", Action: "recalibrate-projection", ResourceType: "form",
			ResourceID:      form.Code,
			After:           map[string]any{"instances": count},
			TargetName:      form.Name,
			ApplicationID:   appID,
			ApplicationCode: appCode,
			ApplicationName: appName,
		})
	}
	return &model.WorkflowProjectionRecalibrateResult{Instances: count}, nil
}
