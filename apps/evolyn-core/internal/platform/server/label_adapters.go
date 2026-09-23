package server

import (
	"context"
	"encoding/json"
	"errors"

	formrepository "evolyn/internal/platform/form/repository"
	formservice "evolyn/internal/platform/form/service"
	iammodel "evolyn/internal/platform/iam/model"
	labelservice "evolyn/internal/platform/label/service"

	"gorm.io/gorm"
)

// labelFormDirectory 只投影标签发布校验所需的表单归属和冻结字段身份。
type labelFormDirectory struct {
	forms    formrepository.FormRepository
	versions formrepository.FormVersionRepository
}

func (d labelFormDirectory) FormByCode(ctx context.Context, code string) (labelservice.FormView, bool, error) {
	form, err := d.forms.GetByCode(ctx, code)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return labelservice.FormView{}, true, nil
		}
		return labelservice.FormView{}, false, err
	}
	return labelservice.FormView{
		ID: form.ID, AppID: form.AppID, Code: form.Code, Name: form.Name,
		Published: form.LatestVersionID != nil,
	}, false, nil
}

func (d labelFormDirectory) PublishedForm(ctx context.Context, id uint) (labelservice.FormView, bool, error) {
	form, err := d.forms.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return labelservice.FormView{}, true, nil
		}
		return labelservice.FormView{}, false, err
	}
	view := labelservice.FormView{ID: form.ID, AppID: form.AppID, Code: form.Code, Name: form.Name, Fields: map[string]bool{}}
	if form.LatestVersionID == nil {
		return view, false, nil
	}
	version, err := d.versions.GetByID(ctx, *form.LatestVersionID)
	if err != nil {
		return labelservice.FormView{}, false, err
	}
	mappings := make([]formservice.SnapshotFieldMapping, 0)
	if len(version.FieldMappings) > 0 && string(version.FieldMappings) != "[]" {
		if err := json.Unmarshal(version.FieldMappings, &mappings); err != nil {
			return labelservice.FormView{}, false, err
		}
	} else {
		content := make(map[string]any)
		if err := json.Unmarshal(version.Content, &content); err != nil {
			return labelservice.FormView{}, false, err
		}
		mappings = formservice.ExtractSnapshotFieldMappings(content)
	}
	for _, mapping := range mappings {
		view.Fields[mapping.WidgetName] = true
		if mapping.FieldID != "" {
			view.Fields[mapping.FieldID] = true
		}
	}
	view.Published = true
	return view, false, nil
}

// labelRecordResolver 桥接表单域已完成权限裁剪的记录读模型。
type labelRecordResolver struct {
	source formservice.LabelRecordSource
}

func (r labelRecordResolver) GetRecord(ctx context.Context, member *iammodel.User, formID, recordID uint) (*labelservice.RecordView, error) {
	record, err := r.source.ReadLabelRecord(ctx, member, formID, recordID)
	if err != nil {
		return nil, err
	}
	return &labelservice.RecordView{FormID: record.FormID, Fields: record.Fields, System: record.System}, nil
}
