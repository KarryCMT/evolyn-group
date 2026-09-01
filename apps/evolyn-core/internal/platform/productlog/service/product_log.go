package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"evolyn/internal/contextx"
	auditservice "evolyn/internal/platform/audit/service"
	apperrors "evolyn/internal/platform/productlog"
	"evolyn/internal/platform/productlog/model"
	"evolyn/internal/platform/productlog/repository"

	kernel "evolyn/internal/model"

	"gorm.io/gorm"
)

// dateLayout 日期入参口径：yyyy-MM-dd 按东八区解析为自然日零点（与个人
// 登录日志自查同口径）
const dateLayout = "2006-01-02"

// legacyDisplay 存量历史行（事件码/分类/摘要为空）的统一降级文案
const legacyDisplay = "历史操作记录"

type productLogService struct {
	repo    repository.Repository
	members MemberDirectory
	apps    ApplicationDirectory
	audit   auditservice.Recorder
}

// normalizePage 分页参数规范化：页码 1 起、页大小默认 20、上限 100
func normalizePage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}
	return page, pageSize
}

// parseDateRange 日期闭区间 → 半开区间：EndDate 换算为次日零点开区间上界；
// start > end 视为参数冲突（与「未知分类/事件码」同级的稳定错误）
func parseDateRange(startDate, endDate string) (repository.LogTimeRange, error) {
	var rng repository.LogTimeRange
	if startDate != "" {
		parsed, err := time.ParseInLocation(dateLayout, startDate, kernel.CSTLocation())
		if err != nil {
			return rng, apperrors.ErrDateInvalid
		}
		rng.Start = parsed
	}
	if endDate != "" {
		parsed, err := time.ParseInLocation(dateLayout, endDate, kernel.CSTLocation())
		if err != nil {
			return rng, apperrors.ErrDateInvalid
		}
		rng.EndExcl = parsed.AddDate(0, 0, 1)
	}
	if !rng.Start.IsZero() && !rng.EndExcl.IsZero() && !rng.Start.Before(rng.EndExcl) {
		return rng, apperrors.ErrTimeRangeInvalid
	}
	return rng, nil
}

// validateQuery 列表/导出共用的筛选规范化与归属校验：日期/成员/应用/
// 分类/事件码全量校验（未知码直接拒绝，避免拼错参数静默返回空列表）
func (s *productLogService) validateQuery(ctx context.Context, tenantID uint, categoryCode, eventCode string, memberID, applicationID uint, startDate, endDate string) (repository.LogTimeRange, error) {
	rng, err := parseDateRange(startDate, endDate)
	if err != nil {
		return rng, err
	}
	if categoryCode != "" && !auditservice.KnownProductCategory(categoryCode) {
		return rng, apperrors.ErrCategoryUnknown
	}
	if eventCode != "" && !auditservice.KnownProductEvent(eventCode) {
		return rng, apperrors.ErrEventUnknown
	}
	if memberID != 0 && s.members != nil {
		if err := s.members.ValidateMember(ctx, tenantID, memberID); err != nil {
			return rng, err
		}
	}
	// application_id 只是租户内的进一步筛选维度：必须先校验归属，不可
	// 替代租户过滤（跨租户应用 ID 与无效同义）
	if applicationID != 0 && s.apps != nil {
		if err := s.apps.ValidateApplication(ctx, tenantID, applicationID); err != nil {
			return rng, err
		}
	}
	return rng, nil
}

func (s *productLogService) List(ctx context.Context, tenantID uint, q model.ProductLogQuery) (*model.ProductLogPage, error) {
	page, pageSize := normalizePage(q.Page, q.PageSize)
	rng, err := s.validateQuery(ctx, tenantID, q.CategoryCode, q.EventCode, q.MemberID, q.ApplicationID, q.StartDate, q.EndDate)
	if err != nil {
		return nil, err
	}

	rows, total, err := s.repo.ListProductLogs(ctx, repository.ProductLogFilter{
		TenantID:      tenantID,
		MemberID:      q.MemberID,
		CategoryCode:  q.CategoryCode,
		EventCode:     q.EventCode,
		ApplicationID: q.ApplicationID,
		Keyword:       q.Keyword,
		Range:         rng,
		// 产品分类白名单：产品日志只读应用内操作，与企业日志范围互斥
		Categories: auditservice.ProductCategoryCodes(),
	}, (page-1)*pageSize, pageSize)
	if err != nil {
		return nil, err
	}

	items := make([]model.ProductLogItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, model.ProductLogItem{
			ActorName:       actorDisplayName(row.ActorNameSnapshot, row.DisplayName),
			OperatedAt:      kernel.JSONTime(row.CreatedAt),
			CategoryCode:    row.CategoryCode,
			CategoryName:    categoryDisplay(row.CategoryCode),
			EventName:       eventDisplay(row.EventCode),
			ApplicationName: row.ApplicationNameSnapshot,
			TargetName:      targetDisplay(row.TargetNameSnapshot),
			Summary:         summaryDisplay(row.Summary),
			IP:              row.IP,
		})
	}
	return &model.ProductLogPage{Items: items, Total: total}, nil
}

func (s *productLogService) Options(ctx context.Context, tenantID uint) (*model.ProductLogOptions, error) {
	// 分类与事件码：审计注册表产品目录（唯一事实源，前端不硬编码）
	catalog := auditservice.CatalogProductCategories()
	categories := make([]model.CategoryOption, 0, len(catalog))
	for _, category := range catalog {
		events := make([]model.EventOption, 0, len(category.Events))
		for _, event := range category.Events {
			events = append(events, model.EventOption{Code: event.Code, Name: event.Name})
		}
		categories = append(categories, model.CategoryOption{
			Code:   category.Code,
			Name:   category.Name,
			Events: events,
		})
	}

	options := &model.ProductLogOptions{
		Categories:   categories,
		Members:      []model.MemberOption{},
		Applications: []model.ApplicationOption{},
	}
	// 目录适配器可空（单测/降级）：跳过对应筛选项
	if s.members != nil {
		members, err := s.members.ListMembers(ctx, tenantID)
		if err != nil {
			return nil, err
		}
		options.Members = members
	}
	if s.apps != nil {
		applications, err := s.apps.ListApplications(ctx, tenantID)
		if err != nil {
			return nil, err
		}
		options.Applications = applications
	}
	return options, nil
}

// categoryDisplay 日志范围展示名；存量历史行（分类为空）降级展示
func categoryDisplay(code string) string {
	if code == "" {
		return legacyDisplay
	}
	return auditservice.CategoryName(code)
}

// eventDisplay 操作类型展示名；存量历史行降级展示，未知事件码原样透出
// （注册表演进期间不吞掉新事件）
func eventDisplay(code string) string {
	if code == "" {
		return legacyDisplay
	}
	return auditservice.EventName(code)
}

// targetDisplay 操作对象展示；快照为空时降级占位（前端渲染「—」）
func targetDisplay(name string) string {
	return name
}

// summaryDisplay 操作详情展示；存量历史行无摘要时降级展示
func summaryDisplay(summary string) string {
	if summary == "" {
		return legacyDisplay
	}
	return summary
}

// actorDisplayName 展示快照优先，存量历史行回查当前昵称兜底
func actorDisplayName(snapshot, fallback string) string {
	if snapshot != "" {
		return snapshot
	}
	return fallback
}

// CreateExport 创建导出任务（一期同步生成）：校验与列表完全同口径 → 计数
// 限流 → 落 pending 任务 → keyset 扫描生成 CSV → 就绪回写 → 提交后
// best-effort 记导出行为审计（企业治理类，不记录导出文件内容）
func (s *productLogService) CreateExport(ctx context.Context, tenantID uint, req model.CreateExportRequest) (*model.ExportTaskView, error) {
	// 与列表一致的筛选规范化（日期/成员/应用/分类/事件码全量校验）
	rng, err := s.validateQuery(ctx, tenantID, req.CategoryCode, req.EventCode, req.MemberID, req.ApplicationID, req.StartDate, req.EndDate)
	if err != nil {
		return nil, err
	}
	filters := repository.ProductLogFilter{
		TenantID:      tenantID,
		MemberID:      req.MemberID,
		CategoryCode:  req.CategoryCode,
		EventCode:     req.EventCode,
		ApplicationID: req.ApplicationID,
		Keyword:       req.Keyword,
		Range:         rng,
		Categories:    auditservice.ProductCategoryCodes(),
	}

	// 数据量预检：超过单次导出上限即拒绝（提示缩小范围）
	_, total, err := s.repo.ListProductLogs(ctx, filters, 0, 1)
	if err != nil {
		return nil, err
	}
	if total > MaxExportRows {
		return nil, apperrors.ErrExportTooLarge
	}

	// 固化任务：申请人从操作者上下文解析（无会话上下文的系统路径不产生任务）
	actor, _ := contextx.ActorFromContext(ctx)
	now := time.Now()
	expiresAt := kernel.JSONTime(now.Add(ExportFileTTL))
	rawFilters, err := json.Marshal(model.ExportFilters{
		MemberID:      req.MemberID,
		CategoryCode:  req.CategoryCode,
		EventCode:     req.EventCode,
		ApplicationID: req.ApplicationID,
		Keyword:       req.Keyword,
		StartDate:     req.StartDate,
		EndDate:       req.EndDate,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal export filters: %w", err)
	}
	task := &model.ExportTask{
		TenantID:  tenantID,
		AccountID: actor.AccountID,
		MemberID:  actor.MemberID,
		Filters:   model.FiltersJSON(rawFilters),
		Total:     total,
		Status:    model.ExportStatusPending,
		ExpiresAt: &expiresAt,
	}
	if err := s.repo.CreateExportTask(ctx, task); err != nil {
		return nil, err
	}

	// 同步生成（一期）：生成失败落 failed 状态并回传业务错误
	content, err := s.buildExportContent(ctx, filters, int(total))
	if err != nil {
		task.Status = model.ExportStatusFailed
		_ = s.repo.UpdateExportTask(ctx, task)
		return nil, err
	}
	task.Status = model.ExportStatusReady
	task.FileName = fmt.Sprintf("产品日志-%s.csv", now.Format("20060102150405"))
	task.FileData = content
	if err := s.repo.UpdateExportTask(ctx, task); err != nil {
		return nil, err
	}

	// 导出行为审计（企业治理类·日志导出分类）：记录筛选时间范围与导出条数，
	// 不记录导出文件内容（best-effort）
	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "productlog", Action: "create", ResourceType: "export",
			ResourceID: fmt.Sprintf("%d", task.ID), TenantID: tenantID,
			EventCode:  "productlog.export.create",
			TargetName: "产品日志",
			Summary:    exportAuditSummary(req.StartDate, req.EndDate, total),
		})
	}
	return exportTaskView(task), nil
}

// exportAuditSummary 导出行为审计摘要（仅时间范围/条数等展示级字段）
func exportAuditSummary(startDate, endDate string, total int64) string {
	if startDate == "" && endDate == "" {
		return fmt.Sprintf("导出产品日志，共 %d 条", total)
	}
	if startDate == "" {
		return fmt.Sprintf("导出产品日志（截至 %s），共 %d 条", endDate, total)
	}
	if endDate == "" {
		return fmt.Sprintf("导出产品日志（%s 起），共 %d 条", startDate, total)
	}
	return fmt.Sprintf("导出产品日志（%s 至 %s），共 %d 条", startDate, endDate, total)
}

// buildExportContent 生成 CSV：按 (created_at,id) keyset 游标分批扫描组装，
// 行数以预检计数封顶（防御扫描期间新增数据导致超限）
func (s *productLogService) buildExportContent(ctx context.Context, filters repository.ProductLogFilter, total int) (string, error) {
	remaining := total
	var rows [][]string
	err := s.repo.ScanProductLogs(ctx, filters, exportScanBatch, func(batch []model.ProductLogRow) error {
		if remaining <= 0 {
			return errExportBudgetExhausted
		}
		remaining -= len(batch)
		rows = append(rows, productCSVRows(batch)...)
		return nil
	})
	if err != nil && !errors.Is(err, errExportBudgetExhausted) {
		return "", err
	}
	return buildCSV(productCSVHeader, rows), nil
}

// errExportBudgetExhausted 导出行数预算用尽：中止扫描但保留已生成内容
// （预算按预检计数设定，正常不会触发）
var errExportBudgetExhausted = errors.New("product log export budget exhausted")

// exportScanBatch keyset 扫描批大小
const exportScanBatch = 500

func (s *productLogService) GetExport(ctx context.Context, tenantID, taskID uint) (*model.ExportTaskView, error) {
	task, err := s.loadTask(ctx, tenantID, taskID)
	if err != nil {
		return nil, err
	}
	return exportTaskView(task), nil
}

func (s *productLogService) ExportFile(ctx context.Context, tenantID, taskID uint) (*model.ExportFileContent, error) {
	task, err := s.loadTask(ctx, tenantID, taskID)
	if err != nil {
		return nil, err
	}
	if task.Status != model.ExportStatusReady {
		return nil, apperrors.ErrExportNotReady
	}
	if isExpired(task) {
		return nil, apperrors.ErrExportExpired
	}
	return &model.ExportFileContent{
		FileName:    task.FileName,
		ContentType: "text/csv; charset=utf-8",
		Data:        []byte(task.FileData),
	}, nil
}

// loadTask 租户归属复核后的任务加载（仓储已按 tenant_id 过滤，
// 不存在/跨租户统一映射 ErrExportNotFound）
func (s *productLogService) loadTask(ctx context.Context, tenantID, taskID uint) (*model.ExportTask, error) {
	task, err := s.repo.GetExportTask(ctx, tenantID, taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrExportNotFound
		}
		return nil, err
	}
	return task, nil
}

// isExpired 任务是否已过有效期（expires_at 为空视为永不过期）
func isExpired(task *model.ExportTask) bool {
	return task.ExpiresAt != nil && task.ExpiresAt.Time().Before(time.Now())
}

// exportTaskView 任务出网视图：ready 且已过期的任务读时投影为 expired
// （创建响应一定是 ready，expired 只出现在后续状态查询）
func exportTaskView(task *model.ExportTask) *model.ExportTaskView {
	status := task.Status
	if status == model.ExportStatusReady && isExpired(task) {
		status = model.ExportStatusExpired
	}
	var expiresAt kernel.JSONTime
	if task.ExpiresAt != nil {
		expiresAt = *task.ExpiresAt
	}
	filters := model.ExportFilters{}
	_ = json.Unmarshal([]byte(task.Filters), &filters)
	return &model.ExportTaskView{
		ID:        task.ID,
		Filters:   filters,
		Total:     task.Total,
		Status:    status,
		FileName:  task.FileName,
		ExpiresAt: expiresAt,
		CreatedAt: task.CreatedAt,
	}
}
