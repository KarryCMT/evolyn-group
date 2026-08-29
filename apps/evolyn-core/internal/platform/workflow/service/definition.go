package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"
	"unicode/utf8"

	"evolyn/internal/contextx"
	kernel "evolyn/internal/model"
	auditservice "evolyn/internal/platform/audit/service"
	"evolyn/internal/platform/httpx"
	iammodel "evolyn/internal/platform/iam/model"
	wfapp "evolyn/internal/platform/workflow"
	"evolyn/internal/platform/workflow/model"
	"evolyn/internal/platform/workflow/repository"

	enginedefinition "evolyn/internal/engine/workflow/definition"
	enginemodel "evolyn/internal/engine/workflow/model"

	"gorm.io/gorm"
)

const (
	defaultListLimit    = 20
	maxListLimit        = 100
	maxNameRunes        = 128
	maxDescriptionRunes = 512
)

// TxManager 事务边界抽象（FIX-021）：与 form 域同形，具体实现在 infrastructure。
type TxManager interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

// definitionService 流程定义服务实现。
type definitionService struct {
	tx       TxManager
	repo     repository.DefinitionRepository
	versions repository.VersionRepository
	access   AccessEvaluator
	audit    auditservice.Recorder
	// validator 引擎严格校验器（无状态，构造一次复用）；发布前的 Expr
	// 预编译经 enginedefinition.Compile 独立把关（双保险：校验 + 编译）
	validator *enginedefinition.Validator
}

// NewDefinitionService 构造流程定义服务（audit 可为 nil：跳过审计记录，便于单测桩）。
func NewDefinitionService(
	tx TxManager,
	repo repository.DefinitionRepository,
	versions repository.VersionRepository,
	access AccessEvaluator,
	audit auditservice.Recorder,
) DefinitionService {
	return &definitionService{
		tx:        tx,
		repo:      repo,
		versions:  versions,
		access:    access,
		audit:     audit,
		validator: enginedefinition.NewValidator(nil),
	}
}

// minimalDraft 最小合法 DSL（新定义草稿初值）：start → end，开箱即可发布。
func minimalDraft() model.DSLContent {
	return model.DSLContent(`{"schemaVersion":"1.0","nodes":[{"key":"start","type":"start","name":"发起"},{"key":"end","type":"end","name":"结束"}],"edges":[{"key":"e_start_end","source":"start","target":"end"}],"settings":{}}`)
}

// newWorkflowCode 生成流程公开编码：wf_ + 16 位随机 hex；租户内唯一索引兜底。
func newWorkflowCode() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate workflow code: %w", err)
	}
	return "wf_" + hex.EncodeToString(buf), nil
}

// validateName / validateDescription 白名单字段校验（trim 后长度约束）。
func validateName(name string) (string, bool) {
	trimmed := trimSpace(name)
	return trimmed, utf8.RuneCountInString(trimmed) >= 1 && utf8.RuneCountInString(trimmed) <= maxNameRunes
}

func validateDescription(description string) (string, bool) {
	trimmed := trimSpace(description)
	return trimmed, utf8.RuneCountInString(trimmed) <= maxDescriptionRunes
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && isSpaceByte(s[start]) {
		start++
	}
	for end > start && isSpaceByte(s[end-1]) {
		end--
	}
	return s[start:end]
}

func isSpaceByte(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\v' || b == '\f'
}

// loadByCode 按 code 加载定义（NotFound 统一口径；租户过滤由 ctx 承载）。
func (s *definitionService) loadByCode(ctx context.Context, code string) (*model.WfDefinition, error) {
	def, err := s.repo.GetByCode(ctx, code)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("workflow %s: %w", code, wfapp.ErrWorkflowNotFound)
		}
		return nil, err
	}
	return def, nil
}

// permissions 取当前成员权限集（nil 成员视为空集）。
func (s *definitionService) permissions(ctx context.Context, member *iammodel.User) map[string]bool {
	if member == nil {
		return map[string]bool{}
	}
	return s.access.Permissions(ctx, member)
}

// validateDSL 将草稿字节解码为引擎协议文档并执行严格校验；存在问题时返回
// issues 负载（path/code/message），由调用方包装为 WORKFLOW_DEFINITION_INVALID。
func (s *definitionService) validateDSL(raw []byte) ([]map[string]string, *enginemodel.Document, error) {
	doc := new(enginemodel.Document)
	if err := json.Unmarshal(raw, doc); err != nil {
		return nil, nil, fmt.Errorf("DSL 文档不是合法 JSON: %w", err)
	}
	issues := make([]map[string]string, 0)
	for _, e := range s.validator.Validate(doc) {
		issues = append(issues, map[string]string{"path": e.Path, "code": e.Code, "message": e.Message})
	}
	if len(issues) > 0 {
		return issues, nil, nil
	}
	return nil, doc, nil
}

// ---- CRUD ----

func (s *definitionService) Create(ctx context.Context, member *iammodel.User, req *model.CreateWorkflowRequest) (*model.WorkflowDetail, error) {
	if !s.permissions(ctx, member)["workflows:create"] {
		return nil, httpx.Wrap(wfapp.ErrForbidden, fmt.Errorf("member cannot create workflow"))
	}
	name, ok := validateName(req.Name)
	if !ok {
		return nil, wfapp.ErrWorkflowNameInvalid
	}
	description, ok := validateDescription(req.Description)
	if !ok {
		return nil, wfapp.ErrWorkflowDescriptionInvalid
	}
	tenantID, okTenant := contextx.TenantIDFromContext(ctx)
	if !okTenant {
		return nil, fmt.Errorf("tenant context required")
	}
	if member == nil || member.ID == 0 || member.TenantID != tenantID {
		return nil, httpx.Wrap(wfapp.ErrForbidden, fmt.Errorf("member not in tenant %d", tenantID))
	}
	code, err := newWorkflowCode()
	if err != nil {
		return nil, err
	}

	def := &model.WfDefinition{
		Code:            code,
		Name:            name,
		Description:     description,
		DraftContent:    minimalDraft(),
		DraftRevision:   1,
		CreatorMemberID: member.ID,
	}
	def.TenantID = tenantID
	created, err := s.repo.Create(ctx, def)
	if err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "workflow", Action: "create", ResourceType: "workflow",
			ResourceID: created.Code,
			After:      map[string]any{"name": created.Name},
		})
	}
	return toDetail(created), nil
}

func (s *definitionService) List(ctx context.Context, member *iammodel.User, query model.ListWorkflowsQuery) (*model.WorkflowPage, error) {
	if !s.permissions(ctx, member)["workflows:get"] {
		return nil, httpx.Wrap(wfapp.ErrForbidden, fmt.Errorf("member cannot list workflows"))
	}
	limit := query.Limit
	if limit <= 0 {
		limit = defaultListLimit
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}
	afterID, hasCursor, err := repository.ParseCursor(query.Cursor)
	if err != nil {
		return nil, fmt.Errorf("无效的分页游标: %w", err)
	}
	rows, hasMore, err := s.repo.List(ctx, repository.ListParams{Limit: limit, HasCursor: hasCursor, AfterID: afterID})
	if err != nil {
		return nil, err
	}
	items := make([]model.WorkflowSummary, 0, len(rows))
	for i := range rows {
		items = append(items, toSummary(&rows[i]))
	}
	page := &model.WorkflowPage{Items: items}
	if hasMore && len(rows) > 0 {
		page.NextCursor = strconv.FormatUint(uint64(rows[len(rows)-1].ID), 10)
	}
	return page, nil
}

func (s *definitionService) Get(ctx context.Context, member *iammodel.User, code string) (*model.WorkflowDetail, error) {
	if !s.permissions(ctx, member)["workflows:get"] {
		return nil, httpx.Wrap(wfapp.ErrForbidden, fmt.Errorf("member cannot read workflow %s", code))
	}
	def, err := s.loadByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return toDetail(def), nil
}

func (s *definitionService) Update(ctx context.Context, member *iammodel.User, code string, req *model.UpdateWorkflowRequest) (*model.WorkflowDetail, error) {
	if !s.permissions(ctx, member)["workflows:patch"] {
		return nil, httpx.Wrap(wfapp.ErrForbidden, fmt.Errorf("member cannot update workflow %s", code))
	}
	def, err := s.loadByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	name := def.Name
	if req.Name != nil {
		updated, ok := validateName(*req.Name)
		if !ok {
			return nil, wfapp.ErrWorkflowNameInvalid
		}
		name = updated
	}
	description := def.Description
	if req.Description != nil {
		updated, ok := validateDescription(*req.Description)
		if !ok {
			return nil, wfapp.ErrWorkflowDescriptionInvalid
		}
		description = updated
	}
	if err := s.repo.UpdateMeta(ctx, def.ID, name, description); err != nil {
		return nil, err
	}
	return s.Get(ctx, member, code)
}

// SaveDraft 保存草稿：严格校验（失败携带 issues）→ 乐观锁条件更新。
// DSL 以引擎严格校验器为唯一事实源，校验通过与表单域「保存前校验」同口径。
func (s *definitionService) SaveDraft(ctx context.Context, member *iammodel.User, code string, req *model.SaveDraftRequest) (*model.SaveDraftResult, error) {
	if !s.permissions(ctx, member)["workflows:update"] {
		return nil, httpx.Wrap(wfapp.ErrForbidden, fmt.Errorf("member cannot save workflow %s draft", code))
	}
	def, err := s.loadByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if req.DraftRevision != def.DraftRevision {
		return nil, httpx.Wrap(wfapp.ErrRevisionConflict,
			fmt.Errorf("workflow %s draft revision %d != %d", code, req.DraftRevision, def.DraftRevision))
	}
	issues, _, err := s.validateDSL(req.Draft)
	if err != nil {
		return nil, err
	}
	if len(issues) > 0 {
		return nil, httpx.Wrap(wfapp.ErrDefinitionInvalid.WithData(map[string]any{"issues": issues}),
			fmt.Errorf("workflow %s draft invalid: %s", code, issues[0]["path"]))
	}
	saved, err := s.repo.SaveDraft(ctx, def.ID, req.DraftRevision, model.DSLContent(req.Draft))
	if err != nil {
		return nil, err
	}
	if !saved {
		return nil, httpx.Wrap(wfapp.ErrRevisionConflict,
			fmt.Errorf("workflow %s draft revision %d stale", code, req.DraftRevision))
	}
	return &model.SaveDraftResult{DraftRevision: req.DraftRevision + 1}, nil
}

// Delete 软删定义：发布版本行保留（运行态历史不随设计态删除，V1.1 §8.1）。
// 运行中实例守卫自 Phase 2（Instance Repository 落地后）接入，当前无实例可存在。
func (s *definitionService) Delete(ctx context.Context, member *iammodel.User, code string) error {
	if !s.permissions(ctx, member)["workflows:delete"] {
		return httpx.Wrap(wfapp.ErrForbidden, fmt.Errorf("member cannot delete workflow %s", code))
	}
	def, err := s.loadByCode(ctx, code)
	if err != nil {
		return err
	}
	if err := s.repo.SoftDelete(ctx, def); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "workflow", Action: "delete", ResourceType: "workflow",
			ResourceID: def.Code,
			Before:     map[string]any{"name": def.Name, "publishedVersion": def.PublishedVersion},
		})
	}
	return nil
}

// ---- 发布与版本 ----

// Publish 发布：口令复核 → 严格校验 → Expr 预编译 → 事务内创建不可变快照并
// 回写 latest_version_id/published_version。历史版本不被触碰。
func (s *definitionService) Publish(ctx context.Context, member *iammodel.User, code string, req *model.PublishRequest) (*model.PublishResult, error) {
	// 发布复用 workflows:create 动词（POST URL 门同口径，与表单域一致）
	if !s.permissions(ctx, member)["workflows:create"] {
		return nil, httpx.Wrap(wfapp.ErrForbidden, fmt.Errorf("member cannot publish workflow %s", code))
	}
	def, err := s.loadByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if req.DraftRevision != def.DraftRevision {
		return nil, httpx.Wrap(wfapp.ErrRevisionConflict,
			fmt.Errorf("workflow %s draft revision %d != %d on publish", code, req.DraftRevision, def.DraftRevision))
	}
	issues, doc, err := s.validateDSL([]byte(def.DraftContent))
	if err != nil {
		return nil, err
	}
	if len(issues) > 0 {
		return nil, httpx.Wrap(wfapp.ErrDefinitionInvalid.WithData(map[string]any{"issues": issues}),
			fmt.Errorf("workflow %s publish blocked: %s", code, issues[0]["path"]))
	}
	// Expr 预编译兜底：即使校验器与编译器规则未来出现漂移，编译失败即拒绝发布
	if _, err := enginedefinition.Compile(doc, nil); err != nil {
		return nil, httpx.Wrap(wfapp.ErrDefinitionInvalid.WithData(map[string]any{"issues": []map[string]string{
			{"path": "$.edges", "code": "EXPRESSION_INVALID", "message": err.Error()},
		}}), fmt.Errorf("workflow %s publish precompile failed: %w", code, err))
	}
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("tenant context required")
	}
	if member == nil || member.ID == 0 || member.TenantID != tenantID {
		return nil, httpx.Wrap(wfapp.ErrForbidden, fmt.Errorf("member not in tenant %d", tenantID))
	}

	var result *model.PublishResult
	if err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		// 发布号在事务内取 max+1；(definition_id, version_no) 唯一约束兜底并发发布
		nextNo, err := s.versions.MaxVersionNo(tctx, def.ID)
		if err != nil {
			return err
		}
		nextNo++
		version := &model.WfDefinitionVersion{
			DefinitionID:        def.ID,
			VersionNo:           nextNo,
			DSLSnapshot:         def.DraftContent,
			PublishedByMemberID: member.ID,
			PublishedAt:         kernel.JSONTime(time.Now()),
		}
		version.TenantID = tenantID
		created, err := s.versions.Create(tctx, version)
		if err != nil {
			return err
		}
		// 回写定义行最新发布指针（草稿不被覆盖）
		if err := s.repo.MarkPublished(tctx, def.ID, created.ID, nextNo); err != nil {
			return err
		}
		result = &model.PublishResult{VersionNo: nextNo}
		return nil
	}); err != nil {
		return nil, err
	}

	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "workflow", Action: "publish", ResourceType: "workflow",
			ResourceID: code,
			After:      map[string]any{"versionNo": result.VersionNo},
		})
	}
	return result, nil
}

func (s *definitionService) ListVersions(ctx context.Context, member *iammodel.User, code string) ([]model.VersionSummary, error) {
	if !s.permissions(ctx, member)["workflows:get"] {
		return nil, httpx.Wrap(wfapp.ErrForbidden, fmt.Errorf("member cannot list workflow %s versions", code))
	}
	def, err := s.loadByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	rows, err := s.versions.ListByDefinition(ctx, def.ID)
	if err != nil {
		return nil, err
	}
	summaries := make([]model.VersionSummary, 0, len(rows))
	for i := range rows {
		summaries = append(summaries, model.VersionSummary{
			VersionNo:           rows[i].VersionNo,
			PublishedByMemberID: rows[i].PublishedByMemberID,
			PublishedAt:         rows[i].PublishedAt,
		})
	}
	return summaries, nil
}

// GetVersion 指定版本详情：快照全文出网（LogicFlow 只读预览的协议来源）。
func (s *definitionService) GetVersion(ctx context.Context, member *iammodel.User, code string, versionNo int) (*model.VersionDetail, error) {
	if !s.permissions(ctx, member)["workflows:get"] {
		return nil, httpx.Wrap(wfapp.ErrForbidden, fmt.Errorf("member cannot read workflow %s version %d", code, versionNo))
	}
	def, err := s.loadByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	version, err := s.versions.GetByDefinitionAndVersionNo(ctx, def.ID, versionNo)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("workflow %s version %d: %w", code, versionNo, wfapp.ErrVersionNotFound)
		}
		return nil, err
	}
	return &model.VersionDetail{
		VersionSummary: model.VersionSummary{
			VersionNo:           version.VersionNo,
			PublishedByMemberID: version.PublishedByMemberID,
			PublishedAt:         version.PublishedAt,
		},
		DSL: json.RawMessage(version.DSLSnapshot),
	}, nil
}

// ---- 投影 ----

func toSummary(def *model.WfDefinition) model.WorkflowSummary {
	return model.WorkflowSummary{
		Code:             def.Code,
		Name:             def.Name,
		Description:      def.Description,
		PublishedVersion: def.PublishedVersion,
		DraftRevision:    def.DraftRevision,
		CreatorMemberID:  def.CreatorMemberID,
		CreatedAt:        def.CreatedAt,
		UpdatedAt:        def.UpdatedAt,
	}
}

func toDetail(def *model.WfDefinition) *model.WorkflowDetail {
	return &model.WorkflowDetail{
		WorkflowSummary: toSummary(def),
		Draft:           json.RawMessage(def.DraftContent),
	}
}
