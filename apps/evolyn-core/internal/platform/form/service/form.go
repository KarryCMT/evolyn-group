package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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
	tx         TxManager
	repo       repository.FormRepository
	versions   repository.FormVersionRepository
	records    repository.FormRecordRepository
	quota      tenantservice.QuotaService
	audit      auditservice.Recorder
	access     AccessEvaluator
	apps       ApplicationDirectory
	menu       MenuMaintenance
	references ReferenceSource
}

// NewFormService 构造表单域服务（records 可为 nil：P1 未启记录提交路径；
// menu 为菜单维护窄端口，可为 nil：跳过节点维护，便于单测桩；references
// 为引用视图只读端口，可为 nil：引用视图返回空集）。
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

// UseReferenceSource 注入引用视图只读端口（装配期一次性调用，ADR-011）：
// 未注入时 ListReferences 返回空集，存量测试桩无需调整。
func (s *formService) UseReferenceSource(src ReferenceSource) {
	s.references = src
}

// FormReferenceSourceInjector 装配期注入能力（可选）。
type FormReferenceSourceInjector interface {
	UseReferenceSource(src ReferenceSource)
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
	if !req.FormType.Valid() {
		return nil, httpx.Wrap(apperrors.ErrFormTypeInvalid,
			fmt.Errorf("invalid form type %q", req.FormType))
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
	code, err := newFormCode()
	if err != nil {
		return nil, err
	}

	var created *model.Form
	if err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		var err error
		created, err = s.provision(tctx, tenantID, member, req.ApplicationID, code, name, req.FormType,
			emptyFormDocument, strings.TrimSpace(req.ParentEntryCode))
		return err
	}); err != nil {
		return nil, err
	}

	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "form", Action: "create", ResourceType: "form",
			ResourceID: created.Code,
			After: map[string]any{
				"name": created.Name, "formType": created.FormType,
				"applicationId": created.ApplicationID,
			},
		})
	}
	return detailOf(created), nil
}

// provision 事务内的创建主流程（Create 与 Copy 共用，ADR-011）：配额占位
// （CheckAndReserve 内先锁租户行再判限额），通过后写入资产行；任一步失败
// 随外层事务整体回滚。draftContent 由调用方给定（Create 传空协议文档，
// Copy 传源表单草稿全文）。parentEntryCode 为空挂应用根级菜单，非空挂指定
// 分组下（合法性由菜单维护端口校验，非法分组随整个创建事务回滚）。
func (s *formService) provision(
	ctx context.Context, tenantID uint, member *iammodel.User, applicationID uint,
	code, name string, formType model.FormType, draftContent model.JSONContent, parentEntryCode string,
) (*model.Form, error) {
	var created *model.Form
	err := s.quota.CheckAndReserve(ctx, tenantID, tenantmodel.QuotaForms, func(tctx context.Context) error {
		draft := &model.Form{
			ApplicationID:   applicationID,
			Code:            code,
			Name:            name,
			FormType:        formType,
			DraftContent:    draftContent,
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

func (s *formService) Get(ctx context.Context, member *iammodel.User, code string) (*model.FormDetail, error) {
	form, err := s.loadByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if !s.access.Permissions(ctx, member)["forms:get"] {
		return nil, httpx.Wrap(apperrors.ErrForbidden, fmt.Errorf("member cannot get form %s", code))
	}
	return detailOf(form), nil
}

// ---- 更新与删除 ----

func (s *formService) Update(ctx context.Context, member *iammodel.User, code string, req *model.UpdateFormRequest) (*model.FormDetail, error) {
	if !s.access.Permissions(ctx, member)["forms:patch"] {
		return nil, httpx.Wrap(apperrors.ErrForbidden, fmt.Errorf("member cannot update form %s", code))
	}
	form, err := s.loadByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	// 「修改名称和图标」是同一按钮的单一权限点（forms:patch）：名称在
	// forms 表、图标/颜色在菜单节点行，两段写经 MenuMaintenance 同事务
	// 同步（ADR-011），事务未注入（单测）时跳过展示属性同步。
	name := req.Name
	icon := req.Icon
	color := req.Color
	if name == nil && icon == nil && color == nil {
		return detailOf(form), nil
	}
	var auditBefore, auditAfter map[string]any
	if err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		auditBefore = map[string]any{}
		auditAfter = map[string]any{}
		if name != nil {
			newName := strings.TrimSpace(*name)
			if newName == "" || utf8.RuneCountInString(newName) > maxFormNameRunes {
				return httpx.Wrap(apperrors.ErrFormNameInvalid, fmt.Errorf("invalid form name length"))
			}
			if newName != form.Name {
				if err := s.repo.UpdateName(tctx, form.ID, newName); err != nil {
					return err
				}
				if s.menu != nil {
					if err := s.menu.SyncFormEntryName(tctx, form.ApplicationID, form.ID, newName); err != nil {
						return err
					}
				}
				auditBefore["name"] = form.Name
				auditAfter["name"] = newName
			}
		}
		// 图标/颜色是菜单节点展示属性：以本域为事实源，经端口同步（空串=清空，
		// 出网投影为 null）
		if icon != nil || color != nil {
			newIcon := ""
			newColor := ""
			if icon != nil {
				newIcon = strings.TrimSpace(*icon)
			}
			if color != nil {
				newColor = strings.TrimSpace(*color)
			}
			if utf8.RuneCountInString(newIcon) > 32 || utf8.RuneCountInString(newColor) > 32 {
				return httpx.Wrap(apperrors.ErrFormIconInvalid, fmt.Errorf("invalid icon/color key length"))
			}
			if s.menu != nil {
				if err := s.menu.SyncFormEntryAppearance(tctx, form.ApplicationID, form.ID, newIcon, newColor); err != nil {
					return err
				}
			}
			if icon != nil {
				auditAfter["icon"] = newIcon
			}
			if color != nil {
				auditAfter["color"] = newColor
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	if s.audit != nil && len(auditAfter) > 0 {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "form", Action: "update", ResourceType: "form",
			ResourceID: form.Code,
			Before:     auditBefore,
			After:      auditAfter,
		})
	}
	updated, err := s.loadByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return detailOf(updated), nil
}

// ---- 切换类型 / 复制 / 引用视图（ADR-011） ----

// SwitchType 切换表单类型：URL 门 POST→forms:create，动作键
// form-actions:switch-type 独立复核；standard↔workflow 互转，流程表单切
// 标准后原流程数据保留（仅不可再发起流程），草稿与发布快照不受影响。
func (s *formService) SwitchType(ctx context.Context, member *iammodel.User, code string, req *model.SwitchFormTypeRequest) (*model.FormDetail, error) {
	perms := s.access.Permissions(ctx, member)
	if !perms["forms:create"] || !perms["form-actions:switch-type"] {
		return nil, httpx.Wrap(apperrors.ErrForbidden,
			fmt.Errorf("member cannot switch form %s type", code))
	}
	if req == nil || !req.FormType.Valid() {
		return nil, httpx.Wrap(apperrors.ErrFormTypeInvalid,
			fmt.Errorf("invalid form type %v", req))
	}
	form, err := s.loadByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if req.FormType == form.FormType {
		return nil, httpx.Wrap(apperrors.ErrFormTypeUnchanged,
			fmt.Errorf("form %s is already %s", code, form.FormType))
	}
	before := form.FormType
	if err := s.repo.UpdateFormType(ctx, form.ID, req.FormType); err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "form", Action: "switch-type", ResourceType: "form",
			ResourceID: form.Code,
			Before:     map[string]any{"formType": before},
			After:      map[string]any{"formType": req.FormType},
		})
	}
	updated, err := s.loadByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return detailOf(updated), nil
}

// Copy 复制表单（ADR-011 双动作码）：目标应用为空或等于源应用走
// form-actions:copy-in-app，跨应用走 form-actions:copy-cross-app；复制
// 草稿全文与表单类型（不复制发布快照与记录），名称追加「（副本）」，
// 事务内占目标应用所属配额并挂目标应用菜单。
func (s *formService) Copy(ctx context.Context, member *iammodel.User, code string, req *model.CopyFormRequest) (*model.FormDetail, error) {
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("tenant context required")
	}
	if member == nil || member.ID == 0 || member.TenantID != tenantID {
		return nil, httpx.Wrap(apperrors.ErrForbidden,
			fmt.Errorf("member not in tenant %d", tenantID))
	}
	perms := s.access.Permissions(ctx, member)
	if !perms["forms:create"] {
		return nil, httpx.Wrap(apperrors.ErrForbidden,
			fmt.Errorf("member cannot copy form %s", code))
	}
	if req == nil {
		return nil, httpx.Wrap(apperrors.ErrFormAppInvalid, fmt.Errorf("copy request is nil"))
	}
	form, err := s.loadByCode(ctx, code)
	if err != nil {
		return nil, err
	}

	// 动作裁决：目标应用与源应用同/异决定走哪个动作码
	targetAppID := form.ApplicationID
	if req.TargetApplicationID != nil && *req.TargetApplicationID != 0 && *req.TargetApplicationID != form.ApplicationID {
		if !perms["form-actions:copy-cross-app"] {
			return nil, httpx.Wrap(apperrors.ErrForbidden,
				fmt.Errorf("member lacks form-actions:copy-cross-app"))
		}
		targetAppID = *req.TargetApplicationID
	} else if !perms["form-actions:copy-in-app"] {
		return nil, httpx.Wrap(apperrors.ErrForbidden,
			fmt.Errorf("member lacks form-actions:copy-in-app"))
	}

	// 目标应用复核（同应用复制同样要求目标应用可用，口径同 Create）
	targetApp, notFound, err := s.apps.ApplicationByID(ctx, targetAppID)
	if err != nil {
		return nil, err
	}
	if notFound || targetApp.Status != applicationStatusActive {
		return nil, httpx.Wrap(apperrors.ErrFormAppInvalid,
			fmt.Errorf("copy target application %d unavailable", targetAppID))
	}

	// 名称追加「（副本）」，超长按 rune 截断到 128
	name := form.Name + "（副本）"
	if utf8.RuneCountInString(name) > maxFormNameRunes {
		runes := []rune(name)
		name = string(runes[:maxFormNameRunes])
	}
	newCode, err := newFormCode()
	if err != nil {
		return nil, err
	}

	var created *model.Form
	if err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		var err error
		created, err = s.provision(tctx, tenantID, member, targetAppID, newCode, name, form.FormType,
			form.DraftContent, strings.TrimSpace(req.ParentEntryCode))
		return err
	}); err != nil {
		return nil, err
	}

	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "form", Action: "copy", ResourceType: "form",
			ResourceID: created.Code,
			After: map[string]any{
				"sourceCode":        form.Code,
				"sourceApplication": form.ApplicationID,
				"targetApplication": targetAppID,
				"formType":          created.FormType,
			},
		})
	}
	return detailOf(created), nil
}

// ListReferences 引用视图（ADR-011）：跨应用反查引用指定表单的菜单节点；
// 只读诊断信息，持 forms:get 即可读取（不做应用可见性裁剪）。
func (s *formService) ListReferences(ctx context.Context, member *iammodel.User, code string) ([]FormReference, error) {
	if !s.access.Permissions(ctx, member)["forms:get"] {
		return nil, httpx.Wrap(apperrors.ErrForbidden,
			fmt.Errorf("member cannot list references of form %s", code))
	}
	form, err := s.loadByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if s.references == nil {
		return []FormReference{}, nil
	}
	return s.references.ListFormReferences(ctx, form.ID)
}

// Delete 软删表单：发布版本行保留（历史记录可追溯），配额随软删释放。
func (s *formService) Delete(ctx context.Context, member *iammodel.User, code string) error {
	if !s.access.Permissions(ctx, member)["forms:delete"] {
		return httpx.Wrap(apperrors.ErrForbidden, fmt.Errorf("member cannot delete form %s", code))
	}
	form, err := s.loadByCode(ctx, code)
	if err != nil {
		return err
	}
	// 软删表单与菜单节点摘除同一事务（M2-资产-1）
	if err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		if err := s.repo.SoftDelete(tctx, form); err != nil {
			return err
		}
		if s.menu != nil {
			return s.menu.DetachFormEntry(tctx, form.ApplicationID, form.ID)
		}
		return nil
	}); err != nil {
		return err
	}
	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "form", Action: "delete", ResourceType: "form",
			ResourceID: form.Code,
			After:      map[string]any{"name": form.Name},
		})
	}
	return nil
}

// ---- 草稿保存 ----

// SaveDraft 保存草稿：先按字段字典严格校验（失败携带 JSON Path issues），再经乐观锁
// 条件更新；0 行影响即修订口令过期。草稿原文原样落库（未编辑属性不丢失）。
func (s *formService) SaveDraft(ctx context.Context, member *iammodel.User, code string, req *model.SaveDraftRequest) (*model.SaveDraftResult, error) {
	if !s.access.Permissions(ctx, member)["forms:update"] {
		return nil, httpx.Wrap(apperrors.ErrForbidden, fmt.Errorf("member cannot save form %s draft", code))
	}
	form, err := s.loadByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	issues := ValidateFormSchema([]byte(req.Content))
	if len(issues) > 0 {
		return nil, httpx.Wrap(apperrors.ErrSchemaInvalid.WithData(map[string]any{"issues": issues}),
			fmt.Errorf("form %s draft invalid: %s", code, issues[0].Path))
	}
	if req.DraftRevision != form.DraftRevision {
		return nil, httpx.Wrap(apperrors.ErrRevisionConflict,
			fmt.Errorf("form %s draft revision %d != %d", code, req.DraftRevision, form.DraftRevision))
	}
	updated, err := s.repo.UpdateDraft(ctx, form.ID, req.DraftRevision, req.Content)
	if err != nil {
		return nil, err
	}
	if !updated {
		return nil, httpx.Wrap(apperrors.ErrRevisionConflict,
			fmt.Errorf("form %s draft revision %d stale", code, req.DraftRevision))
	}
	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "form", Action: "update-draft", ResourceType: "form",
			ResourceID: form.Code,
			After:      map[string]any{"draftRevision": req.DraftRevision + 1},
		})
	}
	return &model.SaveDraftResult{DraftRevision: req.DraftRevision + 1}, nil
}

// ---- 助手 ----

// loadByCode 按稳定公开编码加载：跨租户表现为 NotFound（租户 Callback 过滤）；仅「确实无此行」
// 包装 FORM_NOT_FOUND，基础设施错误原样上抛。
func (s *formService) loadByCode(ctx context.Context, code string) (*model.Form, error) {
	form, err := s.repo.GetByCode(ctx, code)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.Wrap(apperrors.ErrFormNotFound, err)
		}
		return nil, err
	}
	return form, nil
}

// newFormCode 生成表单公开编码：form_ + 16 位随机 hex；租户内唯一索引兜底。
func newFormCode() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate form code: %w", err)
	}
	return "form_" + hex.EncodeToString(buf), nil
}

func detailOf(form *model.Form) *model.FormDetail {
	return &model.FormDetail{
		ApplicationID:    form.ApplicationID,
		Code:             form.Code,
		Name:             form.Name,
		FormType:         form.FormType,
		DraftRevision:    form.DraftRevision,
		PublishedVersion: form.PublishedVersion,
		Draft:            form.DraftContent,
		CreatedAt:        form.CreatedAt,
		UpdatedAt:        form.UpdatedAt,
	}
}

func summaryOf(form *model.Form) model.FormSummary {
	return model.FormSummary{
		ApplicationID:    form.ApplicationID,
		Code:             form.Code,
		Name:             form.Name,
		FormType:         form.FormType,
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
