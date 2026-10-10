package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	enginelabel "evolyn/internal/engine/label"
	"evolyn/internal/platform/httpx"
	iammodel "evolyn/internal/platform/iam/model"
	labelapp "evolyn/internal/platform/label"
	"evolyn/internal/platform/label/model"

	"gorm.io/gorm"
)

const batchPrintOperation = "batch_print"

func (s *templateService) runtimeTemplate(ctx context.Context, member *iammodel.User, formCode string) (*model.Template, bool, error) {
	if _, err := s.ensureMember(ctx, member); err != nil {
		return nil, false, err
	}
	form, notFound, err := s.forms.FormByCode(ctx, strings.TrimSpace(formCode))
	if err != nil {
		return nil, false, err
	}
	if notFound {
		return nil, false, labelapp.ErrFormInvalid
	}
	access, ok := s.records.(OperationAccessResolver)
	if !ok {
		return nil, false, fmt.Errorf("label operation access resolver is not configured")
	}
	allowed, err := access.CanUseOperation(ctx, member, form.ID, batchPrintOperation)
	if err != nil {
		return nil, false, err
	}
	if !allowed {
		return nil, false, httpx.Wrap(labelapp.ErrForbidden, fmt.Errorf("member cannot batch print form %s", form.Code))
	}
	template, err := s.templates.GetByFormCode(ctx, form.Code)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if template.LatestVersionID == nil || template.PublishedVersion == 0 || template.Status != model.StatusPublished {
		return template, false, nil
	}
	return template, true, nil
}

func (s *templateService) RuntimeProfile(ctx context.Context, member *iammodel.User, formCode string) (*model.RuntimeProfile, error) {
	template, available, err := s.runtimeTemplate(ctx, member, formCode)
	if err != nil {
		return nil, err
	}
	profile := &model.RuntimeProfile{
		Available: available, OutputPresets: []model.RuntimeOutputPreset{},
		CanManageTemplate: s.permissions(ctx, member)[iammodel.LabelTemplateResource+":update"],
	}
	if template == nil {
		return profile, nil
	}
	profile.TemplateCode = template.Code
	profile.TemplateName = template.Name
	profile.PublishedVersion = template.PublishedVersion
	if !available {
		return profile, nil
	}
	version, err := s.versions.GetByID(ctx, *template.LatestVersionID)
	if err != nil {
		return nil, err
	}
	schema, issues, err := decodeAndValidate(version.SchemaSnapshot)
	if err != nil || len(issues) > 0 {
		return nil, httpx.Wrap(labelapp.ErrSchemaInvalid, fmt.Errorf("invalid published label snapshot"))
	}
	for _, preset := range enginelabel.OutputPresets(schema) {
		pixelWidth, pixelHeight := enginelabel.PixelSize(preset)
		profile.OutputPresets = append(profile.OutputPresets, model.RuntimeOutputPreset{
			ID: preset.ID, Name: preset.Name, Width: preset.Width, Height: preset.Height,
			Unit: preset.Unit, DPI: preset.DPI, PixelWidth: pixelWidth, PixelHeight: pixelHeight,
		})
	}
	return profile, nil
}

func (s *templateService) RuntimePreview(ctx context.Context, member *iammodel.User, formCode string, req *model.RuntimePreviewRequest) (*enginelabel.RenderResult, error) {
	template, available, err := s.runtimeTemplate(ctx, member, formCode)
	if err != nil {
		return nil, err
	}
	if !available || template == nil || template.LatestVersionID == nil {
		return nil, labelapp.ErrTemplateNotPublished
	}
	recordID, err := strconv.ParseUint(strings.TrimSpace(req.RecordID), 10, 64)
	if err != nil || recordID == 0 {
		return nil, labelapp.ErrRecordNotFound
	}
	version, err := s.versions.GetByID(ctx, *template.LatestVersionID)
	if err != nil {
		return nil, err
	}
	schema, issues, err := decodeAndValidate(version.SchemaSnapshot)
	if err != nil || len(issues) > 0 {
		return nil, httpx.Wrap(labelapp.ErrSchemaInvalid, fmt.Errorf("invalid published label snapshot"))
	}
	scaled, _, err := enginelabel.WithOutputPreset(*schema, strings.TrimSpace(req.OutputPresetID))
	if errors.Is(err, enginelabel.ErrOutputPresetNotFound) {
		return nil, labelapp.ErrOutputPresetInvalid
	}
	if err != nil {
		return nil, err
	}
	data, err := s.runtimeRecordData(ctx, member, template, &scaled, uint(recordID))
	if err != nil {
		return nil, err
	}
	result, err := s.renderer.Render(ctx, enginelabel.RenderRequest{Schema: scaled, Format: "svg", Data: data})
	if err != nil {
		return nil, httpx.Wrap(labelapp.ErrRender, err)
	}
	return result, nil
}

func (s *templateService) runtimeRecordData(ctx context.Context, member *iammodel.User, template *model.Template, schema *enginelabel.Schema, recordID uint) (enginelabel.RenderData, error) {
	resolver, ok := s.records.(OperationRecordResolver)
	if !ok {
		return enginelabel.RenderData{}, fmt.Errorf("label operation record resolver is not configured")
	}
	record, err := resolver.GetRecordForOperation(ctx, member, template.FormID, recordID, batchPrintOperation)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return enginelabel.RenderData{}, httpx.Wrap(labelapp.ErrRecordNotFound, err)
		}
		var biz *httpx.BizError
		if errors.As(err, &biz) && biz.HTTP == 403 {
			return enginelabel.RenderData{}, httpx.Wrap(labelapp.ErrRecordNoPermission, err)
		}
		return enginelabel.RenderData{}, err
	}
	if err := validateRuntimeRecord(record, schema); err != nil {
		return enginelabel.RenderData{}, err
	}
	if usesScanToken(schema) {
		if err := s.attachQRURL(ctx, member, template.AppID, template.FormID, template.ID, recordID, record.System); err != nil {
			return enginelabel.RenderData{}, err
		}
	}
	return enginelabel.RenderData{Fields: record.Fields, System: record.System}, nil
}

func validateRuntimeRecord(record *RecordView, schema *enginelabel.Schema) error {
	if record == nil {
		return labelapp.ErrRecordNoPermission
	}
	for _, fieldID := range fieldReferences(schema) {
		if _, allowed := record.Fields[fieldID]; !allowed {
			return httpx.Wrap(labelapp.ErrFieldNoPermission, fmt.Errorf("field %s denied", fieldID))
		}
	}
	return nil
}
