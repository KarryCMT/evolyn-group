package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"evolyn/internal/contextx"
	enginelabel "evolyn/internal/engine/label"
	kernel "evolyn/internal/model"
	auditservice "evolyn/internal/platform/audit/service"
	"evolyn/internal/platform/httpx"
	iammodel "evolyn/internal/platform/iam/model"
	labelapp "evolyn/internal/platform/label"
	"evolyn/internal/platform/label/model"
	"evolyn/internal/platform/label/repository"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

const (
	defaultListLimit = 20
	maxListLimit     = 100
)

// bizWithData 克隆稳定错误定义后再挂安全数据，避免修改包级错误常量产生并发串扰。
func bizWithData(base *httpx.BizError, data any) *httpx.BizError {
	return httpx.NewBiz(base.Code, base.Msg, base.HTTP).WithData(data)
}

type templateService struct {
	tx        TxManager
	templates repository.TemplateRepository
	versions  repository.VersionRepository
	forms     FormDirectory
	records   RecordResolver
	access    AccessEvaluator
	audit     auditservice.Recorder
	renderer  *enginelabel.Renderer
}

func NewTemplateService(
	tx TxManager,
	templates repository.TemplateRepository,
	versions repository.VersionRepository,
	forms FormDirectory,
	records RecordResolver,
	access AccessEvaluator,
	audit auditservice.Recorder,
) TemplateService {
	return &templateService{
		tx: tx, templates: templates, versions: versions, forms: forms,
		records: records, access: access, audit: audit, renderer: enginelabel.NewRenderer(nil),
	}
}

func (s *templateService) permissions(ctx context.Context, member *iammodel.User) map[string]bool {
	if member == nil {
		return map[string]bool{}
	}
	return s.access.Permissions(ctx, member)
}

func (s *templateService) ensureMember(ctx context.Context, member *iammodel.User) (uint, error) {
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return 0, fmt.Errorf("tenant context required")
	}
	if member == nil || member.ID == 0 || member.TenantID != tenantID {
		return 0, httpx.Wrap(labelapp.ErrForbidden, fmt.Errorf("member not in tenant %d", tenantID))
	}
	return tenantID, nil
}

// authorize 先确认成员确实属于当前租户，再检查资源操作权限。所有公开服务
// 方法统一走这里，避免只依赖路由中间件或权限适配器而遗漏跨租户成员校验。
func (s *templateService) authorize(ctx context.Context, member *iammodel.User, resource, operation string) (uint, error) {
	tenantID, err := s.ensureMember(ctx, member)
	if err != nil {
		return 0, err
	}
	if !s.permissions(ctx, member)[resource+":"+operation] {
		return 0, httpx.Wrap(labelapp.ErrForbidden, fmt.Errorf("member cannot %s %s", operation, resource))
	}
	return tenantID, nil
}

func newCode() (string, error) {
	buffer := make([]byte, 8)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return "label_" + hex.EncodeToString(buffer), nil
}

func defaultSchema(name, formCode string, width, height float64, unit string, dpi int) model.SchemaContent {
	schema := enginelabel.Schema{
		SchemaVersion: "1.0", Name: name,
		Page:     enginelabel.Page{Width: width, Height: height, Unit: unit, DPI: dpi, Background: "#ffffff"},
		Source:   &enginelabel.Source{Type: "form", FormID: formCode},
		Elements: []enginelabel.Element{},
		Settings: enginelabel.Settings{SnapToGrid: true, GridSize: 1, ShowGrid: false},
	}
	encoded, _ := json.Marshal(schema)
	return model.SchemaContent(encoded)
}

func normalizePage(width, height float64, unit string, dpi int) (float64, float64, string, int) {
	if width <= 0 {
		width = 90
	}
	if height <= 0 {
		height = 60
	}
	if unit == "" {
		unit = "mm"
	}
	if dpi == 0 {
		dpi = 300
	}
	return width, height, unit, dpi
}

func (s *templateService) Create(ctx context.Context, member *iammodel.User, req *model.CreateTemplateRequest) (*model.TemplateDetail, error) {
	tenantID, err := s.authorize(ctx, member, iammodel.LabelTemplateResource, "create")
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.Name)
	if utf8.RuneCountInString(name) < 1 || utf8.RuneCountInString(name) > 128 {
		return nil, labelapp.ErrNameInvalid
	}
	form, notFound, err := s.forms.FormByCode(ctx, strings.TrimSpace(req.FormCode))
	if err != nil {
		return nil, err
	}
	if notFound {
		return nil, labelapp.ErrFormInvalid
	}
	width, height, unit, dpi := normalizePage(req.Width, req.Height, req.Unit, req.DPI)
	draft := defaultSchema(name, form.Code, width, height, unit, dpi)
	var schema enginelabel.Schema
	if err := json.Unmarshal(draft, &schema); err != nil || len(enginelabel.Validate(&schema)) > 0 {
		return nil, labelapp.ErrSchemaInvalid
	}
	code, err := newCode()
	if err != nil {
		return nil, err
	}
	template := &model.Template{
		Code: code, Name: name, Description: strings.TrimSpace(req.Description),
		AppID: form.AppID, FormID: form.ID, FormCode: form.Code, Status: model.StatusDraft,
		DraftSchema: draft, DraftRevision: 1,
		Width: width, Height: height, Unit: unit, DPI: dpi, CreatorMemberID: member.ID,
	}
	template.TenantID = tenantID
	created, err := s.templates.Create(ctx, template)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "uk_tn_label_templates_form" {
			return nil, httpx.Wrap(labelapp.ErrFormAlreadyBound, err)
		}
		return nil, err
	}
	s.recordAudit(ctx, "create", created, map[string]any{"name": created.Name})
	return toDetail(created), nil
}

func (s *templateService) List(ctx context.Context, member *iammodel.User, query model.ListTemplatesQuery) (*model.TemplatePage, error) {
	if _, err := s.authorize(ctx, member, iammodel.LabelTemplateResource, "get"); err != nil {
		return nil, err
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
		return nil, fmt.Errorf("invalid label cursor: %w", err)
	}
	rows, hasMore, err := s.templates.List(ctx, repository.ListParams{
		Limit: limit, AfterID: afterID, HasCursor: hasCursor,
		Keyword: strings.TrimSpace(query.Keyword), Status: strings.TrimSpace(query.Status), FormCode: strings.TrimSpace(query.FormCode),
	})
	if err != nil {
		return nil, err
	}
	items := make([]model.TemplateSummary, 0, len(rows))
	for index := range rows {
		items = append(items, toSummary(&rows[index]))
	}
	page := &model.TemplatePage{Items: items}
	if hasMore && len(rows) > 0 {
		page.NextCursor = strconv.FormatUint(uint64(rows[len(rows)-1].ID), 10)
	}
	return page, nil
}

func (s *templateService) load(ctx context.Context, code string) (*model.Template, error) {
	template, err := s.templates.GetByCode(ctx, code)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.Wrap(labelapp.ErrTemplateNotFound, err)
		}
		return nil, err
	}
	// Repository callback 是主隔离边界；服务层再核对返回模型的租户归属，
	// 防止未来适配器漏挂 callback 时通过公开编码探测其他租户模板。
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok || template.TenantID != tenantID {
		return nil, httpx.Wrap(labelapp.ErrTemplateNotFound, fmt.Errorf("label template tenant mismatch"))
	}
	return template, nil
}

func (s *templateService) Get(ctx context.Context, member *iammodel.User, code string) (*model.TemplateDetail, error) {
	if _, err := s.authorize(ctx, member, iammodel.LabelTemplateResource, "get"); err != nil {
		return nil, err
	}
	template, err := s.load(ctx, code)
	if err != nil {
		return nil, err
	}
	return toDetail(template), nil
}

func decodeAndValidate(raw []byte) (*enginelabel.Schema, []enginelabel.Issue, error) {
	var schema enginelabel.Schema
	if err := json.Unmarshal(raw, &schema); err != nil {
		return nil, nil, err
	}
	issues := enginelabel.Validate(&schema)
	return &schema, issues, nil
}

func (s *templateService) SaveDraft(ctx context.Context, member *iammodel.User, code string, req *model.SaveDraftRequest) (*model.SaveDraftResult, error) {
	if _, err := s.authorize(ctx, member, iammodel.LabelTemplateResource, "update"); err != nil {
		return nil, err
	}
	template, err := s.load(ctx, code)
	if err != nil {
		return nil, err
	}
	if req.DraftRevision != template.DraftRevision {
		return nil, httpx.Wrap(labelapp.ErrRevisionConflict, fmt.Errorf("label template revision mismatch"))
	}
	schema, issues, err := decodeAndValidate(req.Schema)
	if err != nil {
		return nil, httpx.Wrap(labelapp.ErrSchemaInvalid, err)
	}
	if len(issues) > 0 {
		return nil, httpx.Wrap(
			bizWithData(labelapp.ErrSchemaInvalid, map[string]any{"issues": issues}),
			fmt.Errorf("invalid schema at %s", issues[0].Path),
		)
	}
	if schema.Source == nil || schema.Source.Type != "form" || schema.Source.FormID != template.FormCode {
		return nil, httpx.Wrap(labelapp.ErrSchemaInvalid, fmt.Errorf("schema source must bind form %s", template.FormCode))
	}
	saved, err := s.templates.SaveDraft(ctx, template.ID, req.DraftRevision, model.SchemaContent(req.Schema), schema.Page.Width, schema.Page.Height, schema.Page.Unit, schema.Page.DPI)
	if err != nil {
		return nil, err
	}
	if !saved {
		return nil, httpx.Wrap(labelapp.ErrRevisionConflict, fmt.Errorf("stale label template revision"))
	}
	s.recordAudit(ctx, "update-draft", template, map[string]any{"draftRevision": req.DraftRevision + 1})
	return &model.SaveDraftResult{DraftRevision: req.DraftRevision + 1}, nil
}

func fieldReferences(schema *enginelabel.Schema) []string {
	seen := map[string]bool{}
	fields := make([]string, 0)
	for index := range schema.Elements {
		value := schema.Elements[index].Value
		if value != nil && value.Type == "field" && !seen[value.FieldID] {
			seen[value.FieldID] = true
			fields = append(fields, value.FieldID)
		}
	}
	return fields
}

func (s *templateService) Publish(ctx context.Context, member *iammodel.User, code string, req *model.PublishRequest) (*model.PublishResult, error) {
	if _, err := s.authorize(ctx, member, iammodel.LabelTemplateResource, "create"); err != nil {
		return nil, err
	}
	var template *model.Template
	var versionNo int
	err := s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		locked, err := s.templates.GetByCodeForUpdate(txCtx, code)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return httpx.Wrap(labelapp.ErrTemplateNotFound, err)
			}
			return err
		}
		template = locked
		if req.DraftRevision != template.DraftRevision {
			return httpx.Wrap(labelapp.ErrRevisionConflict, fmt.Errorf("label template revision mismatch on publish"))
		}
		if template.PreviewedDraftRevision != template.DraftRevision {
			return httpx.Wrap(labelapp.ErrRealPreviewRequired, fmt.Errorf("draft revision %d has not passed real-data preview", template.DraftRevision))
		}
		if template.PublishedDraftRevision == template.DraftRevision && template.PublishedVersion > 0 {
			versionNo = template.PublishedVersion
			return nil
		}
		schema, issues, err := decodeAndValidate(template.DraftSchema)
		if err != nil {
			return httpx.Wrap(labelapp.ErrSchemaInvalid, err)
		}
		if len(issues) > 0 {
			return httpx.Wrap(
				bizWithData(labelapp.ErrSchemaInvalid, map[string]any{"issues": issues}),
				fmt.Errorf("invalid schema at %s", issues[0].Path),
			)
		}
		form, notFound, err := s.forms.PublishedForm(txCtx, template.FormID)
		if err != nil {
			return err
		}
		if notFound || !form.Published {
			return labelapp.ErrFormInvalid
		}
		for _, fieldID := range fieldReferences(schema) {
			if !form.Fields[fieldID] {
				return httpx.Wrap(
					bizWithData(labelapp.ErrFieldNotFound, map[string]any{"fieldId": fieldID}),
					fmt.Errorf("label field %s not found", fieldID),
				)
			}
		}
		for _, element := range schema.Elements {
			if element.Value != nil && element.Value.Type == "expression" {
				return httpx.Wrap(labelapp.ErrExpression, fmt.Errorf("label expression resolver is not configured"))
			}
		}
		maxVersion, err := s.versions.MaxVersionNo(txCtx, template.ID)
		if err != nil {
			return err
		}
		versionNo = maxVersion + 1
		now := time.Now()
		version, err := s.versions.Create(txCtx, &model.TemplateVersion{
			TemplateID: template.ID, VersionNo: versionNo, SchemaVersion: schema.SchemaVersion,
			SchemaSnapshot: model.SchemaContent(template.DraftSchema), PublishedByMemberID: member.ID,
			PublishedAt: kernel.JSONTime(now), TenantID: template.TenantID,
		})
		if err != nil {
			return err
		}
		return s.templates.MarkPublished(txCtx, template.ID, version.ID, versionNo, template.DraftRevision)
	})
	if err != nil {
		return nil, err
	}
	s.recordAudit(ctx, "publish", template, map[string]any{"versionNo": versionNo})
	return &model.PublishResult{VersionNo: versionNo}, nil
}

func (s *templateService) Delete(ctx context.Context, member *iammodel.User, code string) error {
	if _, err := s.authorize(ctx, member, iammodel.LabelTemplateResource, "delete"); err != nil {
		return err
	}
	template, err := s.load(ctx, code)
	if err != nil {
		return err
	}
	if err := s.templates.SoftDelete(ctx, template); err != nil {
		return err
	}
	s.recordAudit(ctx, "delete", template, nil)
	return nil
}

func (s *templateService) Preview(ctx context.Context, member *iammodel.User, code string, req *model.PreviewRequest) (*enginelabel.RenderResult, error) {
	if _, err := s.authorize(ctx, member, iammodel.LabelTemplateResource, "create"); err != nil {
		return nil, err
	}
	template, err := s.load(ctx, code)
	if err != nil {
		return nil, err
	}
	result, err := s.render(ctx, member, template, template.DraftSchema, req.RecordID, req.Format)
	if err != nil {
		return nil, err
	}
	marked, err := s.templates.MarkPreviewed(ctx, template.ID, template.DraftRevision)
	if err != nil {
		return nil, err
	}
	if !marked {
		return nil, httpx.Wrap(labelapp.ErrRevisionConflict, fmt.Errorf("draft changed while real-data preview was rendering"))
	}
	return result, nil
}

func (s *templateService) Render(ctx context.Context, member *iammodel.User, req *model.RenderRequest) (*enginelabel.RenderResult, error) {
	if _, err := s.authorize(ctx, member, iammodel.LabelResource, "create"); err != nil {
		return nil, err
	}
	template, err := s.load(ctx, req.TemplateCode)
	if err != nil {
		return nil, err
	}
	if template.LatestVersionID == nil || template.PublishedVersion == 0 || template.Status != model.StatusPublished {
		return nil, labelapp.ErrTemplateNotPublished
	}
	version, err := s.versions.GetByID(ctx, *template.LatestVersionID)
	if err != nil {
		return nil, err
	}
	return s.render(ctx, member, template, version.SchemaSnapshot, req.RecordID, req.Format)
}

func (s *templateService) render(ctx context.Context, member *iammodel.User, template *model.Template, raw []byte, recordID, format string) (*enginelabel.RenderResult, error) {
	if format == "" {
		format = "svg"
	}
	format = strings.ToLower(strings.TrimSpace(format))
	if format != "svg" && format != "png" && format != "pdf" {
		return nil, labelapp.ErrFormatUnsupported
	}
	parsedID, err := strconv.ParseUint(strings.TrimSpace(recordID), 10, 64)
	if err != nil || parsedID == 0 {
		return nil, labelapp.ErrRecordNotFound
	}
	record, err := s.records.GetRecord(ctx, member, template.FormID, uint(parsedID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.Wrap(labelapp.ErrRecordNotFound, err)
		}
		var biz *httpx.BizError
		if errors.As(err, &biz) && biz.HTTP == 403 {
			return nil, httpx.Wrap(labelapp.ErrRecordNoPermission, err)
		}
		return nil, err
	}
	schema, issues, err := decodeAndValidate(raw)
	if err != nil {
		return nil, httpx.Wrap(labelapp.ErrSchemaInvalid, err)
	}
	if len(issues) > 0 {
		return nil, httpx.Wrap(
			bizWithData(labelapp.ErrSchemaInvalid, map[string]any{"issues": issues}),
			fmt.Errorf("invalid schema at %s", issues[0].Path),
		)
	}
	// 表单域对不可见字段不出键；区别于“字段存在但值为空”，正式渲染必须明确拒绝越权字段。
	for _, fieldID := range fieldReferences(schema) {
		if _, allowed := record.Fields[fieldID]; !allowed {
			return nil, httpx.Wrap(
				bizWithData(labelapp.ErrFieldNoPermission, map[string]any{"fieldId": fieldID}),
				fmt.Errorf("member cannot read label field %s", fieldID),
			)
		}
	}
	result, err := s.renderer.Render(ctx, enginelabel.RenderRequest{
		Schema: *schema, Format: format,
		Data: enginelabel.RenderData{Fields: record.Fields, System: record.System},
	})
	if err != nil {
		if errors.Is(err, enginelabel.ErrQRGenerate) {
			return nil, httpx.Wrap(labelapp.ErrQRGenerate, err)
		}
		return nil, httpx.Wrap(labelapp.ErrRender, err)
	}
	return result, nil
}

func toSummary(template *model.Template) model.TemplateSummary {
	return model.TemplateSummary{
		Code: template.Code, Name: template.Name, Description: template.Description,
		AppID: template.AppID, FormCode: template.FormCode, Status: template.Status,
		PublishedVersion: template.PublishedVersion, DraftRevision: template.DraftRevision,
		PreviewedDraftRevision: template.PreviewedDraftRevision, PublishedDraftRevision: template.PublishedDraftRevision,
		Width: template.Width, Height: template.Height, Unit: template.Unit, DPI: template.DPI,
		CreatorMemberID: template.CreatorMemberID, CreatedAt: template.CreatedAt, UpdatedAt: template.UpdatedAt,
	}
}

func toDetail(template *model.Template) *model.TemplateDetail {
	return &model.TemplateDetail{TemplateSummary: toSummary(template), Draft: json.RawMessage(template.DraftSchema)}
}

func (s *templateService) recordAudit(ctx context.Context, action string, template *model.Template, after map[string]any) {
	if s.audit == nil {
		return
	}
	s.audit.Record(ctx, auditservice.Entry{
		Module: "label", Action: action, ResourceType: "label-template", ResourceID: template.Code,
		After: after, TargetName: template.Name, AppID: template.AppID,
	})
}
