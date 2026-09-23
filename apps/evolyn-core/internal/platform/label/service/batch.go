package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	enginelabel "evolyn/internal/engine/label"
	"evolyn/internal/platform/httpx"
	iammodel "evolyn/internal/platform/iam/model"
	labelapp "evolyn/internal/platform/label"
	"evolyn/internal/platform/label/model"

	"gorm.io/gorm"
)

const maxBatchRecords = 1000

func (s *templateService) BatchRender(ctx context.Context, member *iammodel.User, req *model.BatchRenderRequest) (*model.BatchRenderCreated, error) {
	tenantID, err := s.authorize(ctx, member, iammodel.LabelResource, "create")
	if err != nil {
		return nil, err
	}
	if s.tasks == nil || s.queue == nil || s.members == nil || !s.queue.Available() {
		return nil, labelapp.ErrTaskQueueUnavailable
	}
	if s.artifacts == nil || !s.artifacts.Available() {
		return nil, labelapp.ErrStorageUnavailable
	}
	if req == nil || len(req.RecordIDs) == 0 || len(req.RecordIDs) > maxBatchRecords {
		return nil, labelapp.ErrBatchInvalid
	}
	format := strings.ToLower(strings.TrimSpace(req.Format))
	if format == "" {
		format = "pdf"
	}
	if format != "pdf" {
		return nil, labelapp.ErrFormatUnsupported
	}
	template, err := s.load(ctx, strings.TrimSpace(req.TemplateCode))
	if err != nil {
		return nil, err
	}
	if template.LatestVersionID == nil || template.PublishedVersion == 0 || template.Status != model.StatusPublished {
		return nil, labelapp.ErrTemplateNotPublished
	}
	recordIDs, err := normalizeBatchRecordIDs(req.RecordIDs)
	if err != nil {
		return nil, err
	}
	taskCode, err := newRenderTaskCode()
	if err != nil {
		return nil, err
	}
	task := &model.RenderTask{
		TenantID: tenantID, Code: taskCode, TemplateID: template.ID, AppID: template.AppID, FormID: template.FormID,
		TemplateVersionID: *template.LatestVersionID, TemplateVersionNo: template.PublishedVersion,
		OutputFormat: format, Status: model.RenderTaskPending, TotalCount: len(recordIDs),
		RequestedByMemberID: member.ID,
	}
	items := make([]model.RenderTaskItem, len(recordIDs))
	for index, recordID := range recordIDs {
		items[index] = model.RenderTaskItem{RecordID: recordID, SequenceNo: index + 1, Status: model.RenderItemPending}
	}
	if err := s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		return s.tasks.Create(txCtx, task, items)
	}); err != nil {
		return nil, err
	}
	// 任务行提交后才入队，Worker 永远不会观察到未提交的 task/items。
	if err := s.queue.Enqueue(ctx, tenantID, task.Code); err != nil {
		_ = s.tasks.Finish(ctx, task.ID, model.RenderTaskFailed, "", labelapp.ErrTaskQueueUnavailable.Code, "渲染队列暂不可用", 0, 0)
		return nil, httpx.Wrap(labelapp.ErrTaskQueueUnavailable, err)
	}
	return &model.BatchRenderCreated{TaskID: task.Code, Status: task.Status}, nil
}

func (s *templateService) GetRenderTask(ctx context.Context, member *iammodel.User, taskCode string) (*model.RenderTaskDetail, error) {
	if _, err := s.authorize(ctx, member, iammodel.LabelResource, "get"); err != nil {
		return nil, err
	}
	if s.tasks == nil || !validRenderTaskCode(taskCode) {
		return nil, labelapp.ErrTaskNotFound
	}
	task, err := s.tasks.GetByCode(ctx, taskCode)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, labelapp.ErrTaskNotFound
	}
	if err != nil {
		return nil, err
	}
	items, err := s.tasks.ListItems(ctx, task.ID)
	if err != nil {
		return nil, err
	}
	details := make([]model.RenderTaskItemDetail, len(items))
	for index := range items {
		item := &items[index]
		details[index] = model.RenderTaskItemDetail{
			RecordID: strconv.FormatUint(uint64(item.RecordID), 10), SequenceNo: item.SequenceNo,
			Status: item.Status, ErrorCode: item.ErrorCode, ErrorMessage: item.ErrorMessage,
			StartedAt: item.StartedAt, FinishedAt: item.FinishedAt,
			CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
		}
	}
	return &model.RenderTaskDetail{RenderTask: *task, Items: details}, nil
}

func (s *templateService) DownloadRenderTask(ctx context.Context, member *iammodel.User, taskCode string) (*ArtifactDownload, error) {
	if _, err := s.authorize(ctx, member, iammodel.LabelResource, "get"); err != nil {
		return nil, err
	}
	if s.tasks == nil || s.artifacts == nil || !validRenderTaskCode(taskCode) {
		return nil, labelapp.ErrTaskNotFound
	}
	task, err := s.tasks.GetByCode(ctx, taskCode)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, labelapp.ErrTaskNotFound
	}
	if err != nil {
		return nil, err
	}
	if (task.Status != model.RenderTaskSuccess && task.Status != model.RenderTaskPartialSuccess) || task.FileCode == "" {
		return nil, labelapp.ErrTaskNotReady
	}
	result, err := s.artifacts.Download(ctx, member, task.FileCode)
	if err != nil {
		return nil, httpx.Wrap(labelapp.ErrStorageUnavailable, err)
	}
	return result, nil
}

// ProcessBatch 是 Asynq Handler 调用的内部入口。业务拒绝只失败对应条目；
// 数据库、渲染器基础设施和 RustFS 故障返回错误交给队列重试。
func (s *templateService) ProcessBatch(ctx context.Context, taskCode string) (runErr error) {
	if s.tasks == nil || s.artifacts == nil || s.members == nil {
		return fmt.Errorf("label batch dependencies are not configured")
	}
	claimed, err := s.tasks.Claim(ctx, taskCode)
	if err != nil || !claimed {
		return err
	}
	task, err := s.tasks.GetByCode(ctx, taskCode)
	if err != nil {
		return err
	}
	defer func() {
		if runErr != nil {
			_ = s.tasks.ResetPending(ctx, task.ID, labelapp.ErrRender.Code, "任务执行失败，等待重试")
		}
	}()
	version, err := s.versions.GetByID(ctx, task.TemplateVersionID)
	if err != nil {
		return err
	}
	schema, issues, err := decodeAndValidate(version.SchemaSnapshot)
	if err != nil || len(issues) > 0 {
		return s.tasks.Finish(ctx, task.ID, model.RenderTaskFailed, "", labelapp.ErrSchemaInvalid.Code, "模板发布快照无效", 0, task.TotalCount)
	}
	member, err := s.members.MemberByID(ctx, task.RequestedByMemberID)
	if err != nil {
		return err
	}
	items, err := s.tasks.ListItems(ctx, task.ID)
	if err != nil {
		return err
	}
	pages := make([]enginelabel.RenderData, 0, len(items))
	successCount, failedCount := 0, 0
	for index := range items {
		item := &items[index]
		if err := s.tasks.MarkItemRunning(ctx, item.ID); err != nil {
			return err
		}
		data, itemErr := s.batchRecordData(ctx, member, task.AppID, task.FormID, task.TemplateID, schema, item.RecordID)
		if itemErr != nil {
			code, message, handled := safeBatchItemError(itemErr)
			if !handled {
				return itemErr
			}
			failedCount++
			if err := s.tasks.MarkItemFinished(ctx, item.ID, model.RenderItemFailed, code, message); err != nil {
				return err
			}
		} else {
			successCount++
			pages = append(pages, data)
			if err := s.tasks.MarkItemFinished(ctx, item.ID, model.RenderItemSuccess, "", ""); err != nil {
				return err
			}
		}
		progress := ((index + 1) * 99) / len(items)
		if err := s.tasks.UpdateProgress(ctx, task.ID, successCount, failedCount, progress); err != nil {
			return err
		}
	}
	if len(pages) == 0 {
		return s.tasks.Finish(ctx, task.ID, model.RenderTaskFailed, "", labelapp.ErrRender.Code, "所有记录均渲染失败", 0, failedCount)
	}
	result, err := s.renderer.RenderBatch(ctx, *schema, pages)
	if err != nil {
		return err
	}
	createdAt := time.Time(task.CreatedAt)
	relativePath := fmt.Sprintf("label/%04d/%02d/%s/result.pdf", createdAt.Year(), createdAt.Month(), task.Code)
	fileCode, err := s.artifacts.StorePDF(ctx, member, task.Code+".pdf", relativePath, result.Content)
	if err != nil {
		return err
	}
	status := model.RenderTaskSuccess
	if failedCount > 0 {
		status = model.RenderTaskPartialSuccess
	}
	return s.tasks.Finish(ctx, task.ID, status, fileCode, "", "", successCount, failedCount)
}

// FailBatch 在 Asynq 重试耗尽后收口任务，避免任务永久停留在 pending。
// cause 只用于服务端诊断，不写入面向用户的错误摘要。
func (s *templateService) FailBatch(ctx context.Context, taskCode string, cause error) error {
	if s.tasks == nil {
		return cause
	}
	task, err := s.tasks.GetByCode(ctx, taskCode)
	if err != nil {
		return err
	}
	return s.tasks.Finish(ctx, task.ID, model.RenderTaskFailed, "", labelapp.ErrRender.Code, "任务执行失败，请重试", task.SuccessCount, task.FailedCount)
}

func (s *templateService) batchRecordData(ctx context.Context, member *iammodel.User, appID, formID, templateID uint, schema *enginelabel.Schema, recordID uint) (enginelabel.RenderData, error) {
	record, err := s.records.GetRecord(ctx, member, formID, recordID)
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
	for _, fieldID := range fieldReferences(schema) {
		if _, allowed := record.Fields[fieldID]; !allowed {
			return enginelabel.RenderData{}, httpx.Wrap(labelapp.ErrFieldNoPermission, fmt.Errorf("field %s denied", fieldID))
		}
	}
	if usesScanToken(schema) {
		if err := s.attachQRURL(ctx, member, appID, formID, templateID, recordID, record.System); err != nil {
			return enginelabel.RenderData{}, err
		}
	}
	data := enginelabel.RenderData{Fields: record.Fields, System: record.System}
	// 单页预渲染在逐项边界捕获二维码或字体问题，避免一条坏数据拖垮整批。
	if _, err := s.renderer.Render(ctx, enginelabel.RenderRequest{Schema: *schema, Format: "pdf", Data: data}); err != nil {
		if errors.Is(err, enginelabel.ErrQRGenerate) {
			return enginelabel.RenderData{}, httpx.Wrap(labelapp.ErrQRGenerate, err)
		}
		return enginelabel.RenderData{}, httpx.Wrap(labelapp.ErrRender, err)
	}
	return data, nil
}

func normalizeBatchRecordIDs(values []string) ([]uint, error) {
	result := make([]uint, 0, len(values))
	seen := make(map[uint]struct{}, len(values))
	for _, value := range values {
		parsed, err := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
		if err != nil || parsed == 0 {
			return nil, labelapp.ErrBatchInvalid
		}
		id := uint(parsed)
		if _, duplicate := seen[id]; duplicate {
			return nil, labelapp.ErrBatchInvalid
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result, nil
}

func safeBatchItemError(err error) (string, string, bool) {
	var biz *httpx.BizError
	if !errors.As(err, &biz) {
		return "", "", false
	}
	switch biz.Code {
	case labelapp.ErrRecordNotFound.Code, labelapp.ErrRecordNoPermission.Code,
		labelapp.ErrFieldNoPermission.Code, labelapp.ErrQRGenerate.Code, labelapp.ErrRender.Code:
		return biz.Code, biz.Msg, true
	default:
		return "", "", false
	}
}

func newRenderTaskCode() (string, error) {
	buffer := make([]byte, 12)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return "lrt_" + hex.EncodeToString(buffer), nil
}

func validRenderTaskCode(value string) bool {
	value = strings.TrimSpace(value)
	return strings.HasPrefix(value, "lrt_") && len(value) == 28
}
