package service

import (
	"context"

	"evolyn/internal/contextx"
	"evolyn/internal/platform/audit/model"
	"evolyn/internal/platform/audit/repository"

	"github.com/sirupsen/logrus"
)

// summaryMaxRunes 摘要列 varchar(1000) 的安全截断上限（按字符计）
const summaryMaxRunes = 1000

type auditService struct {
	repo       repository.AuditRepository
	actorNamer ActorNamer
}

// NewService 审计域服务工厂（ADR-007 域模块化）。actorNamer 为可选的
// 操作者显示名解析端口（企业日志展示快照，000036）： variadic 形式保持
// 既有装配/测试调用兼容
func NewService(repo repository.AuditRepository, actorNamer ...ActorNamer) Recorder {
	svc := &auditService{repo: repo}
	for _, namer := range actorNamer {
		if namer != nil {
			svc.actorNamer = namer
		}
	}
	return svc
}

// Record 落一条业务审计：身份/租户/请求元数据尽量从 ctx 自动解析，
// 企业日志展示投影（事件码/分类/快照/摘要）经事件注册表推导后一并固化；
// 写入失败仅记录错误日志——审计不可用不应阻断业务主流程
func (s *auditService) Record(ctx context.Context, e Entry) {
	log := &model.AuditLog{
		Module:       e.Module,
		Action:       e.Action,
		ResourceType: e.ResourceType,
		ResourceID:   e.ResourceID,
		TenantID:     e.TenantID,
		AccountID:    e.AccountID,
		MemberID:     e.MemberID,
	}

	// ctx 自动补全：租户 → 操作者 → 请求元数据
	if log.TenantID == 0 {
		if tenantID, ok := contextx.TenantIDFromContext(ctx); ok {
			log.TenantID = tenantID
		}
	}
	if actor, ok := contextx.ActorFromContext(ctx); ok {
		if log.AccountID == 0 {
			log.AccountID = actor.AccountID
		}
		if log.MemberID == 0 {
			log.MemberID = actor.MemberID
		}
	}
	meta := contextx.RequestMetaFromContext(ctx)
	log.IP = meta.IP
	log.UserAgent = meta.UserAgent
	log.RequestID = meta.RequestID

	before, err := model.MarshalJSONText(e.Before)
	if err != nil {
		logrus.Warnf("audit before snapshot marshal failed: %v", err)
		return
	}
	after, err := model.MarshalJSONText(e.After)
	if err != nil {
		logrus.Warnf("audit after snapshot marshal failed: %v", err)
		return
	}
	log.BeforeData = before
	log.AfterData = after

	// 企业日志展示投影（000036）：目标名 → 事件码/分类/摘要 → 操作人快照
	targetName := e.TargetName
	if targetName == "" {
		targetName = displayNameFromSnapshot(e.After, e.Before)
	}
	log.EventCode, log.CategoryCode, log.Summary = projectEvent(e, targetName)
	log.TargetNameSnapshot = truncateRunes(targetName, 256)

	// 产品日志应用维度快照（000064）：应用内操作写时固化，0 视为非应用内操作
	if e.ApplicationID != 0 {
		appID := e.ApplicationID
		log.ApplicationID = &appID
		log.ApplicationCode = truncateRunes(e.ApplicationCode, 128)
		log.ApplicationNameSnapshot = truncateRunes(e.ApplicationName, 256)
	}

	log.ActorNameSnapshot = e.ActorName
	if log.ActorNameSnapshot == "" && log.MemberID != 0 && s.actorNamer != nil {
		log.ActorNameSnapshot = truncateRunes(s.actorNamer.MemberDisplayName(ctx, log.MemberID), 128)
	}

	if err := s.repo.Create(ctx, log); err != nil {
		logrus.Warnf("audit record persist failed: module=%s action=%s resource=%s/%s: %v",
			e.Module, e.Action, e.ResourceType, e.ResourceID, err)
	}
}

// projectEvent 推导企业日志展示投影三元组：
//  1. Entry 显式 EventCode 优先，分类缺省按事件码回查注册表；
//  2. 未显式指定事件码时，按 module+resourceType 查注册表机械拼接
//     （未登记的资源保持空投影，读取侧降级「历史操作记录」）；
//  3. 摘要缺省时由注册表模板生成（动词+资源标签+目标名），超长截断
func projectEvent(e Entry, targetName string) (eventCode, category, summary string) {
	eventCode, category, summary = e.EventCode, e.CategoryCode, e.Summary
	if eventCode == "" {
		if meta, ok := resourceRegistry[e.Module+"/"+e.ResourceType]; ok {
			eventCode = EventCodeOf(e.Module, e.ResourceType, e.Action)
			category = meta.Category
		}
	} else if category == "" {
		if _, _, meta, _, ok := splitEventCode(eventCode); ok {
			category = meta.Category
		}
	}
	if summary == "" && eventCode != "" {
		summary = buildSummaryFromCode(eventCode, targetName)
	}
	return eventCode, category, truncateRunes(summary, summaryMaxRunes)
}

// buildSummaryFromCode 按事件码生成摘要（事件码须已登记，否则返回空串）
func buildSummaryFromCode(code, targetName string) string {
	module, resourceType, _, action, ok := splitEventCode(code)
	if !ok {
		return ""
	}
	return BuildSummary(module, resourceType, action, targetName)
}

// displayNameFromSnapshot 从变更快照提取目标展示名：仅取 name/nickname
// 展示级键（脱敏口径：快照其余字段绝不进摘要/快照出网）；After 优先
// （新建场景名称只在 After）
func displayNameFromSnapshot(snapshots ...interface{}) string {
	for _, snapshot := range snapshots {
		switch m := snapshot.(type) {
		case map[string]string:
			if name := firstNonEmpty(m["name"], m["nickname"]); name != "" {
				return name
			}
		case map[string]interface{}:
			if name := firstNonEmpty(stringOf(m["name"]), stringOf(m["nickname"])); name != "" {
				return name
			}
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// stringOf 任意展示值的字符串化；nil/非字符串标量返回空串（保守脱敏）
func stringOf(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	default:
		return ""
	}
}

// truncateRunes 按字符数安全截断（列宽 varchar(n) 语义）
func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}
