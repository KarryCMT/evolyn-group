package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"evolyn/internal/contextx"
	auditservice "evolyn/internal/platform/audit/service"
	apperrors "evolyn/internal/platform/form"
	"evolyn/internal/platform/form/model"
	"evolyn/internal/platform/form/repository"
	"evolyn/internal/platform/httpx"
	iammodel "evolyn/internal/platform/iam/model"
	tenantmodel "evolyn/internal/platform/tenant/model"
	tenantservice "evolyn/internal/platform/tenant/service"

	"gorm.io/gorm"
)

const (
	defaultListLimit = 20
	maxListLimit     = 100
	maxFormNameRunes = 128

	// applicationStatusActive 应用可见状态（与应用域 model.ApplicationStatusActive 一致；
	// 经窄端口传递字符串，避免跨域模型依赖）
	applicationStatusActive = "active"
)

// TxManager 事务边界抽象（FIX-021）：与 application 域同形，具体实现在 infrastructure。
type TxManager interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

// emptyFormDocument 空协议文档（新表单草稿初值）。
var emptyFormDocument = model.JSONContent(`{"content":{"type":"form","items":[]}}`)

// formService 表单资产服务实现。
type formService struct {
	tx       TxManager
	repo     repository.FormRepository
	versions repository.FormVersionRepository
	records  repository.FormRecordRepository
	quota    tenantservice.QuotaService
	audit    auditservice.Recorder
	access   AccessEvaluator
	apps     ApplicationDirectory
	menu     MenuMaintenance
}

// NewFormService 构造表单域服务（records 可为 nil：P1 未启记录提交路径；
// menu 为菜单维护窄端口，可为 nil：跳过节点维护，便于单测桩）。
func NewFormService(
	tx TxManager,
	repo repository.FormRepository,
	versions repository.FormVersionRepository,
	records repository.FormRecordRepository,
	quota tenantservice.QuotaService,
	audit auditservice.Recorder,
	access AccessEvaluator,
	apps ApplicationDirectory,
	menu MenuMaintenance,
) FormService {
	return &formService{
		tx:       tx,
		repo:     repo,
		versions: versions,
		records:  records,
		quota:    quota,
		audit:    audit,
		access:   access,
		apps:     apps,
		menu:     menu,
	}
}

// ---- 创建 ----

// Create 创建表单：校验成员/应用归属 → 事务内配额占位 + 资产落库 → 提交后审计。
func (s *formService) Create(ctx context.Context, member *iammodel.User, req *model.CreateFormRequest) (*model.FormDetail, error) {
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("tenant context required")
	}
	// 操作者确属当前租户（禁止裸 ID 绑定，§9.3 口径）
	if member == nil || member.ID == 0 || member.TenantID != tenantID {
		return nil, httpx.Wrap(apperrors.ErrForbidden,
			fmt.Errorf("member not in tenant %d", tenantID))
	}
	// 与路由层 POST /forms（verb=create）同口径的 Service 复核
	if !s.access.Permissions(ctx, member)["forms:create"] {
		return nil, httpx.Wrap(apperrors.ErrForbidden,
			fmt.Errorf("member %d cannot create form", member.ID))
	}

	name := strings.TrimSpace(req.Name)
	if name == "" || utf8.RuneCountInString(name) > maxFormNameRunes {
		return nil, httpx.Wrap(apperrors.ErrFormNameInvalid, fmt.Errorf("invalid form name length"))
	}
	app, notFound, err := s.apps.ApplicationByID(ctx, req.ApplicationID)
	if err != nil {
		return nil, err
	}
	if notFound {
		return nil, httpx.Wrap(apperrors.ErrFormAppInvalid, fmt.Errorf("application %d not found", req.ApplicationID))
	}
	if app.Status != applicationStatusActive {
		return nil, httpx.Wrap(apperrors.ErrFormAppInvalid, fmt.Errorf("application %d status %s", app.ID, app.Status))
	}

	var created *model.Form
	if err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		var err error
		created, err = s.provision(tctx, tenantID, member, req.ApplicationID, name,
			strings.TrimSpace(req.ParentEntryCode))
		return err
	}); err != nil {
		return nil, err
	}

	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "form", Action: "create", ResourceType: "form",
			ResourceID: strconv.FormatUint(uint64(created.ID), 10),
			After:      map[string]any{"name": created.Name, "applicationId": created.ApplicationID},
		})
	}
	return detailOf(created), nil
}

// provision 事务内的创建主流程：配额占位（CheckAndReserve 内先锁租户行再判限额），
// 通过后写入资产行；任一步失败随外层事务整体回滚。parentEntryCode 为空挂
// 应用根级菜单，非空挂指定分组下（合法性由菜单维护端口校验，非法分组随
// 整个创建事务回滚）。
func (s *formService) provision(
	ctx context.Context, tenantID uint, member *iammodel.User, applicationID uint, name, parentEntryCode string,
) (*model.Form, error) {
	var created *model.Form
	err := s.quota.CheckAndReserve(ctx, tenantID, tenantmodel.QuotaForms, func(tctx context.Context) error {
		draft := &model.Form{
			ApplicationID:   applicationID,
			Name:            name,
			DraftContent:    emptyFormDocument,
			DraftRevision:   1,
			ProtocolVersion: 1,
			CreatorMemberID: member.ID,
		}
		draft.TenantID = tenantID
		form, cerr := s.repo.Create(tctx, draft)
		if cerr != nil {
			return cerr
		}
		created = form
		// M2-资产-1：同事务挂菜单节点（form 类型、target 指向本表单，
		// menu_revision 随之递增）；端口未注入（单测）时跳过。
		if s.menu != nil {
			return s.menu.AttachFormEntry(tctx, applicationID, form.ID, name, parentEntryCode)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

// ---- 查询 ----

func (s *formService) List(ctx context.Context, member *iammodel.User, query model.ListFormsQuery) (*model.FormPage, error) {
	if !s.access.Permissions(ctx, member)["forms:list"] {
		return nil, httpx.Wrap(apperrors.ErrForbidden, fmt.Errorf("member cannot list forms"))
	}
	if query.ApplicationID == 0 {
		return nil, httpx.Wrap(apperrors.ErrFormAppInvalid, fmt.Errorf("applicationId required"))
	}
	limit := query.Limit
	if limit <= 0 {
		limit = defaultListLimit
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}
	afterID, hasCursor, err := decodeListCursor(query.Cursor)
	if err != nil {
		return nil, httpx.Wrap(apperrors.ErrFormAppInvalid, err)
	}

	forms, hasMore, err := s.repo.List(ctx, repository.ListParams{
		ApplicationID: query.ApplicationID,
		Limit:         limit,
		HasCursor:     hasCursor,
		AfterID:       afterID,
	})
	if err != nil {
		return nil, err
	}
	page := &model.FormPage{Items: make([]model.FormSummary, 0, len(forms)), HasMore: hasMore}
	for i := range forms {
		page.Items = append(page.Items, summaryOf(&forms[i]))
	}
	if hasMore && len(forms) > 0 {
		page.NextCursor = encodeListCursor(forms[len(forms)-1].ID)
	}
	return page, nil
}

func (s *formService) Get(ctx context.Context, member *iammodel.User, id uint) (*model.FormDetail, error) {
	form, err := s.load(ctx, id)
	if err != nil {
		return nil, err
	}
	if !s.access.Permissions(ctx, member)["forms:get"] {
		return nil, httpx.Wrap(apperrors.ErrForbidden, fmt.Errorf("member cannot get form %d", id))
	}
	return detailOf(form), nil
}

// ---- 更新与删除 ----

func (s *formService) Update(ctx context.Context, member *iammodel.User, id uint, req *model.UpdateFormRequest) (*model.FormDetail, error) {
	if !s.access.Permissions(ctx, member)["forms:patch"] {
		return nil, httpx.Wrap(apperrors.ErrForbidden, fmt.Errorf("member cannot update form %d", id))
	}
	form, err := s.load(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" || utf8.RuneCountInString(name) > maxFormNameRunes {
			return nil, httpx.Wrap(apperrors.ErrFormNameInvalid, fmt.Errorf("invalid form name length"))
		}
		if name != form.Name {
			before := form.Name
			// 改名与菜单节点名同步同一事务（M2-资产-1）
			if err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
				if err := s.repo.UpdateName(tctx, id, name); err != nil {
					return err
				}
				if s.menu != nil {
					return s.menu.SyncFormEntryName(tctx, form.ApplicationID, id, name)
				}
				return nil
			}); err != nil {
				return nil, err
			}
			if s.audit != nil {
				s.audit.Record(ctx, auditservice.Entry{
					Module: "form", Action: "update", ResourceType: "form",
					ResourceID: strconv.FormatUint(uint64(id), 10),
					Before:     map[string]any{"name": before},
					After:      map[string]any{"name": name},
				})
			}
		}
	}
	updated, err := s.load(ctx, id)
	if err != nil {
		return nil, err
	}
	return detailOf(updated), nil
}

// Delete 软删表单：发布版本行保留（历史记录可追溯），配额随软删释放。
func (s *formService) Delete(ctx context.Context, member *iammodel.User, id uint) error {
	if !s.access.Permissions(ctx, member)["forms:delete"] {
		return httpx.Wrap(apperrors.ErrForbidden, fmt.Errorf("member cannot delete form %d", id))
	}
	form, err := s.load(ctx, id)
	if err != nil {
		return err
	}
	// 软删表单与菜单节点摘除同一事务（M2-资产-1）
	if err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		if err := s.repo.SoftDelete(tctx, form); err != nil {
			return err
		}
		if s.menu != nil {
			return s.menu.DetachFormEntry(tctx, form.ApplicationID, id)
		}
		return nil
	}); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "form", Action: "delete", ResourceType: "form",
			ResourceID: strconv.FormatUint(uint64(id), 10),
			After:      map[string]any{"name": form.Name},
		})
	}
	return nil
}

// ---- 草稿保存 ----

// SaveDraft 保存草稿：先按字段字典严格校验（失败携带 JSON Path issues），再经乐观锁
// 条件更新；0 行影响即修订口令过期。草稿原文原样落库（未编辑属性不丢失）。
func (s *formService) SaveDraft(ctx context.Context, member *iammodel.User, id uint, req *model.SaveDraftRequest) (*model.SaveDraftResult, error) {
	if !s.access.Permissions(ctx, member)["forms:update"] {
		return nil, httpx.Wrap(apperrors.ErrForbidden, fmt.Errorf("member cannot save form %d draft", id))
	}
	form, err := s.load(ctx, id)
	if err != nil {
		return nil, err
	}
	issues := ValidateFormSchema([]byte(req.Content))
	if len(issues) > 0 {
		return nil, httpx.Wrap(apperrors.ErrSchemaInvalid.WithData(map[string]any{"issues": issues}),
			fmt.Errorf("form %d draft invalid: %s", id, issues[0].Path))
	}
	if req.DraftRevision != form.DraftRevision {
		return nil, httpx.Wrap(apperrors.ErrRevisionConflict,
			fmt.Errorf("form %d draft revision %d != %d", id, req.DraftRevision, form.DraftRevision))
	}
	updated, err := s.repo.UpdateDraft(ctx, id, req.DraftRevision, req.Content)
	if err != nil {
		return nil, err
	}
	if !updated {
		return nil, httpx.Wrap(apperrors.ErrRevisionConflict,
			fmt.Errorf("form %d draft revision %d stale", id, req.DraftRevision))
	}
	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "form", Action: "update-draft", ResourceType: "form",
			ResourceID: strconv.FormatUint(uint64(id), 10),
			After:      map[string]any{"draftRevision": req.DraftRevision + 1},
		})
	}
	return &model.SaveDraftResult{DraftRevision: req.DraftRevision + 1}, nil
}

// ---- 助手 ----

// load 按 ID 加载：跨租户表现为 NotFound（租户 Callback 过滤）；仅「确实无此行」
// 包装 FORM_NOT_FOUND，基础设施错误原样上抛。
func (s *formService) load(ctx context.Context, id uint) (*model.Form, error) {
	form, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.Wrap(apperrors.ErrFormNotFound, err)
		}
		return nil, err
	}
	return form, nil
}

func detailOf(form *model.Form) *model.FormDetail {
	return &model.FormDetail{
		ID:               form.ID,
		ApplicationID:    form.ApplicationID,
		Name:             form.Name,
		DraftRevision:    form.DraftRevision,
		PublishedVersion: form.PublishedVersion,
		Draft:            form.DraftContent,
		CreatedAt:        form.CreatedAt,
		UpdatedAt:        form.UpdatedAt,
	}
}

func summaryOf(form *model.Form) model.FormSummary {
	return model.FormSummary{
		ID:               form.ID,
		ApplicationID:    form.ApplicationID,
		Name:             form.Name,
		PublishedVersion: form.PublishedVersion,
		UpdatedAt:        form.UpdatedAt,
	}
}

// ---- 游标编解码（id 倒序，客户端不透明） ----

type listCursor struct {
	ID uint `json:"i"`
}

func encodeListCursor(id uint) string {
	b, _ := json.Marshal(listCursor{ID: id})
	return base64.RawURLEncoding.EncodeToString(b)
}

func decodeListCursor(cursor string) (uint, bool, error) {
	if cursor == "" {
		return 0, false, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return 0, false, fmt.Errorf("invalid list cursor")
	}
	var c listCursor
	if err := json.Unmarshal(raw, &c); err != nil || c.ID == 0 {
		return 0, false, fmt.Errorf("invalid list cursor")
	}
	return c.ID, true, nil
}
