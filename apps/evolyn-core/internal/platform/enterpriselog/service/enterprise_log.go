package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"evolyn/internal/contextx"
	auditservice "evolyn/internal/platform/audit/service"
	apperrors "evolyn/internal/platform/enterpriselog"
	"evolyn/internal/platform/enterpriselog/model"
	"evolyn/internal/platform/enterpriselog/repository"

	kernel "evolyn/internal/model"

	"gorm.io/gorm"
)

// dateLayout 日期入参口径：yyyy-MM-dd 按东八区解析为自然日零点（与个人
// 登录日志自查同口径）
const dateLayout = "2006-01-02"

// legacyDisplay 存量历史行（事件码/分类/摘要为空）的统一降级文案
const legacyDisplay = "历史操作记录"

type enterpriseLogService struct {
	repo    repository.Repository
	members MemberDirectory
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

// validateMember 成员筛选归属校验（可空的成员目录适配器下跳过）
func (s *enterpriseLogService) validateMember(ctx context.Context, tenantID, memberID uint) error {
	if memberID == 0 || s.members == nil {
		return nil
	}
	return s.members.ValidateMember(ctx, tenantID, memberID)
}

func (s *enterpriseLogService) ListLoginLogs(ctx context.Context, tenantID uint, q model.LoginLogQuery) (*model.LoginLogPage, error) {
	page, pageSize := normalizePage(q.Page, q.PageSize)
	rng, err := parseDateRange(q.StartDate, q.EndDate)
	if err != nil {
		return nil, err
	}
	if err := s.validateMember(ctx, tenantID, q.MemberID); err != nil {
		return nil, err
	}

	rows, total, err := s.repo.ListLoginLogs(ctx, repository.LoginLogFilter{
		TenantID: tenantID,
		MemberID: q.MemberID,
		Range:    rng,
	}, (page-1)*pageSize, pageSize)
	if err != nil {
		return nil, err
	}

	items := make([]model.LoginLogItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, model.LoginLogItem{
			ActorName: actorDisplayName(row.ActorNameSnapshot, row.DisplayName),
			LoggedAt:  kernel.JSONTime(row.CreatedAt),
			Location:  row.Location,
			Client:    row.Client,
			IP:        row.IP,
		})
	}
	return &model.LoginLogPage{Items: items, Total: total}, nil
}

func (s *enterpriseLogService) ListOperationLogs(ctx context.Context, tenantID uint, q model.OperationLogQuery) (*model.OperationLogPage, error) {
	page, pageSize := normalizePage(q.Page, q.PageSize)
	rng, err := parseDateRange(q.StartDate, q.EndDate)
	if err != nil {
		return nil, err
	}
	// 未知日志范围/事件码直接拒绝：避免拼错参数静默返回空列表
	if q.CategoryCode != "" && !auditservice.KnownCategory(q.CategoryCode) {
		return nil, apperrors.ErrCategoryUnknown
	}
	if q.EventCode != "" && !auditservice.KnownEvent(q.EventCode) {
		return nil, apperrors.ErrEventUnknown
	}
	if err := s.validateMember(ctx, tenantID, q.MemberID); err != nil {
		return nil, err
	}

	rows, total, err := s.repo.ListAuditLogs(ctx, repository.AuditLogFilter{
		TenantID:          tenantID,
		MemberID:          q.MemberID,
		CategoryCode:      q.CategoryCode,
		EventCode:         q.EventCode,
		Range:             rng,
		ExcludeCategories: auditservice.ProductCategoryCodes(),
	}, (page-1)*pageSize, pageSize)
	if err != nil {
		return nil, err
	}

	items := make([]model.OperationLogItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, model.OperationLogItem{
			ActorName:    actorDisplayName(row.ActorNameSnapshot, row.DisplayName),
			OperatedAt:   kernel.JSONTime(row.CreatedAt),
			CategoryName: categoryDisplay(row.CategoryCode),
			EventName:    eventDisplay(row.EventCode),
			Summary:      summaryDisplay(row.Summary),
			IP:           row.IP,
		})
	}
	return &model.OperationLogPage{Items: items, Total: total}, nil
}

func (s *enterpriseLogService) ListCategories() []model.CategoryOption {
	catalog := auditservice.CatalogCategories()
	options := make([]model.CategoryOption, 0, len(catalog))
	for _, category := range catalog {
		events := make([]model.EventOption, 0, len(category.Events))
		for _, event := range category.Events {
			events = append(events, model.EventOption{Code: event.Code, Name: event.Name})
		}
		options = append(options, model.CategoryOption{
			Code:   category.Code,
			Name:   category.Name,
			Events: events,
		})
	}
	return options
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

// summaryDisplay 操作详情展示；存量历史行无摘要时降级展示
func summaryDisplay(summary string) string {
	if summary == "" {
		return legacyDisplay
	}
	return summary
}

// CreateExport 创建导出任务（一期同步生成）：校验与列表完全同口径 → 计数
// 限流 → 落 pending 任务 → keyset 扫描生成 CSV → 就绪回写 → 提交后
// best-effort 记导出行为审计（不记录导出文件内容）
func (s *enterpriseLogService) CreateExport(ctx context.Context, tenantID uint, req model.CreateExportRequest) (*model.ExportTaskView, error) {
	if req.Kind != model.ExportKindLogin && req.Kind != model.ExportKindOperation {
		return nil, apperrors.ErrExportKindInvalid
	}

	// 与列表一致的筛选规范化（日期/成员/分类/事件码全量校验）
	rng, err := parseDateRange(req.StartDate, req.EndDate)
	if err != nil {
		return nil, err
	}
	if req.CategoryCode != "" && !auditservice.KnownCategory(req.CategoryCode) {
		return nil, apperrors.ErrCategoryUnknown
	}
	if req.EventCode != "" && !auditservice.KnownEvent(req.EventCode) {
		return nil, apperrors.ErrEventUnknown
	}
	if err := s.validateMember(ctx, tenantID, req.MemberID); err != nil {
		return nil, err
	}

	filters := model.ExportFilters{
		MemberID:     req.MemberID,
		CategoryCode: req.CategoryCode,
		EventCode:    req.EventCode,
		StartDate:    req.StartDate,
		EndDate:      req.EndDate,
	}

	// 数据量预检：超过单次导出上限即拒绝（提示缩小范围）
	var total int64
	if req.Kind == model.ExportKindLogin {
		_, total, err = s.repo.ListLoginLogs(ctx, repository.LoginLogFilter{
			TenantID: tenantID, MemberID: req.MemberID, Range: rng,
		}, 0, 1)
	} else {
		_, total, err = s.repo.ListAuditLogs(ctx, repository.AuditLogFilter{
			TenantID: tenantID, MemberID: req.MemberID,
			CategoryCode: req.CategoryCode, EventCode: req.EventCode, Range: rng,
			ExcludeCategories: auditservice.ProductCategoryCodes(),
		}, 0, 1)
	}
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
	rawFilters, err := json.Marshal(filters)
	if err != nil {
		return nil, fmt.Errorf("marshal export filters: %w", err)
	}
	task := &model.ExportTask{
		TenantID:  tenantID,
		AccountID: actor.AccountID,
		MemberID:  actor.MemberID,
		Kind:      req.Kind,
		Filters:   model.FiltersJSON(rawFilters),
		Total:     total,
		Status:    model.ExportStatusPending,
		ExpiresAt: &expiresAt,
	}
	if err := s.repo.CreateExportTask(ctx, task); err != nil {
		return nil, err
	}

	// 同步生成（一期）：生成失败落 failed 状态并回传业务错误
	content, err := s.buildExportContent(ctx, tenantID, req.Kind, repository.LoginLogFilter{
		TenantID: tenantID, MemberID: req.MemberID, Range: rng,
	}, repository.AuditLogFilter{
		TenantID: tenantID, MemberID: req.MemberID,
		CategoryCode: req.CategoryCode, EventCode: req.EventCode, Range: rng,
		ExcludeCategories: auditservice.ProductCategoryCodes(),
	}, int(total))
	if err != nil {
		task.Status = model.ExportStatusFailed
		_ = s.repo.UpdateExportTask(ctx, task)
		return nil, err
	}
	task.Status = model.ExportStatusReady
	task.FileName = fmt.Sprintf("企业日志-%s-%s.csv", model.ExportKindLabel(req.Kind), now.Format("20060102150405"))
	task.FileData = content
	if err := s.repo.UpdateExportTask(ctx, task); err != nil {
		return nil, err
	}

	// 导出行为审计：记录日志类型/时间范围/条数，不记录文件内容（best-effort）
	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "enterpriselog", Action: "create", ResourceType: "export",
			ResourceID: fmt.Sprintf("%d", task.ID), TenantID: tenantID,
			EventCode:  "enterpriselog.export.create",
			TargetName: model.ExportKindLabel(req.Kind),
			Summary:    exportAuditSummary(req.Kind, req.StartDate, req.EndDate, total),
		})
	}
	return exportTaskView(task), nil
}

// exportAuditSummary 导出行为审计摘要（仅日志类型/时间范围/条数等展示级字段）
func exportAuditSummary(kind, startDate, endDate string, total int64) string {
	kindLabel := model.ExportKindLabel(kind)
	if startDate == "" && endDate == "" {
		return fmt.Sprintf("导出%s，共 %d 条", kindLabel, total)
	}
	if startDate == "" {
		return fmt.Sprintf("导出%s（截至 %s），共 %d 条", kindLabel, endDate, total)
	}
	if endDate == "" {
		return fmt.Sprintf("导出%s（%s 起），共 %d 条", kindLabel, startDate, total)
	}
	return fmt.Sprintf("导出%s（%s 至 %s），共 %d 条", kindLabel, startDate, endDate, total)
}

// buildExportContent 生成 CSV：按 (created_at,id) keyset 游标分批扫描组装，
// 行数以预检计数封顶（防御扫描期间新增数据导致超限）
func (s *enterpriseLogService) buildExportContent(ctx context.Context, tenantID uint, kind string, loginFilter repository.LoginLogFilter, auditFilter repository.AuditLogFilter, total int) (string, error) {
	remaining := total
	var b builder
	if kind == model.ExportKindLogin {
		b.header = loginCSVHeader
		err := s.repo.ScanLoginLogs(ctx, loginFilter, exportScanBatch, func(rows []model.LoginLogRow) error {
			if remaining <= 0 {
				return errExportBudgetExhausted
			}
			remaining -= len(rows)
			b.rows = append(b.rows, loginCSVRows(rows)...)
			return nil
		})
		if err != nil && !errors.Is(err, errExportBudgetExhausted) {
			return "", err
		}
	} else {
		b.header = operationCSVHeader
		err := s.repo.ScanAuditLogs(ctx, auditFilter, exportScanBatch, func(rows []model.AuditLogRow) error {
			if remaining <= 0 {
				return errExportBudgetExhausted
			}
			remaining -= len(rows)
			b.rows = append(b.rows, operationCSVRows(rows)...)
			return nil
		})
		if err != nil && !errors.Is(err, errExportBudgetExhausted) {
			return "", err
		}
	}
	return buildCSV(b.header, b.rows), nil
}

// errExportBudgetExhausted 导出行数预算用尽：中止扫描但保留已生成内容
// （预算按预检计数设定，正常不会触发）
var errExportBudgetExhausted = errors.New("enterprise log export budget exhausted")

// exportScanBatch keyset 扫描批大小
const exportScanBatch = 500

// builder CSV 组装中间态（header + 已累积数据行）
type builder struct {
	header []string
	rows   [][]string
}

func (s *enterpriseLogService) GetExport(ctx context.Context, tenantID, taskID uint) (*model.ExportTaskView, error) {
	task, err := s.loadTask(ctx, tenantID, taskID)
	if err != nil {
		return nil, err
	}
	return exportTaskView(task), nil
}

func (s *enterpriseLogService) ExportFile(ctx context.Context, tenantID, taskID uint) (*model.ExportFileContent, error) {
	task, err := s.loadTask(ctx, tenantID, taskID)
	if err != nil {
		return nil, err
	}
	if task.Status == model.ExportStatusFailed {
		return nil, apperrors.ErrExportNotReady
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
func (s *enterpriseLogService) loadTask(ctx context.Context, tenantID, taskID uint) (*model.ExportTask, error) {
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
		Kind:      task.Kind,
		KindLabel: model.ExportKindLabel(task.Kind),
		Filters:   filters,
		Total:     task.Total,
		Status:    status,
		FileName:  task.FileName,
		ExpiresAt: expiresAt,
		CreatedAt: task.CreatedAt,
	}
}
