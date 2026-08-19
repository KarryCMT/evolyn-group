package service

import (
	"context"

	"evolyn/internal/contextx"
	"evolyn/internal/platform/audit/model"
	"evolyn/internal/platform/audit/repository"

	"github.com/sirupsen/logrus"
)

type auditService struct {
	repo repository.AuditRepository
}

// NewService 审计域服务工厂（ADR-007 域模块化）
func NewService(repo repository.AuditRepository) Recorder {
	return &auditService{repo: repo}
}

// Record 落一条业务审计：身份/租户/请求元数据尽量从 ctx 自动解析，
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

	if err := s.repo.Create(ctx, log); err != nil {
		logrus.Warnf("audit record persist failed: module=%s action=%s resource=%s/%s: %v",
			e.Module, e.Action, e.ResourceType, e.ResourceID, err)
	}
}
