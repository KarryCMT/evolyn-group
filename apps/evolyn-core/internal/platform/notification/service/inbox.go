package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"time"

	apperrors "evolyn/internal/platform/notification"
	"evolyn/internal/platform/notification/model"
	"evolyn/internal/platform/notification/repository"

	kernel "evolyn/internal/model"
)

// 游标与分页默认值
const (
	defaultInboxLimit      = 20
	maxInboxLimit          = 100
	defaultRetentionMonths = 6
)

type inboxService struct {
	tx              TxManager
	messages        repository.MessageRepository
	retentionMonths int
}

func newInboxService(tx TxManager, messages repository.MessageRepository, retentionMonths int) *inboxService {
	if retentionMonths <= 0 {
		retentionMonths = defaultRetentionMonths
	}
	return &inboxService{tx: tx, messages: messages, retentionMonths: retentionMonths}
}

// UnreadSummary 未读摘要：分类计数由 SQL 排除过期消息（不依赖清理任务准时性），
// 只返回未读数大于 0 的分类
func (s *inboxService) UnreadSummary(
	ctx context.Context, tenantID, memberID uint,
) (*model.UnreadSummaryView, error) {
	rows, err := s.messages.CountUnreadByCategory(ctx, tenantID, memberID)
	if err != nil {
		return nil, err
	}
	view := &model.UnreadSummaryView{
		Categories:  rows,
		GeneratedAt: kernel.JSONTime(time.Now()),
	}
	for _, row := range rows {
		view.UnreadTotal += row.UnreadCount
	}
	return view, nil
}

// ListInbox 分类游标分页：校验分类/事件归属与游标合法性后经仓储读取；
// nextCursor 固化最后一条 (occurred_at, id)
func (s *inboxService) ListInbox(
	ctx context.Context, tenantID, memberID uint, q model.InboxQuery,
) (*model.InboxPageView, error) {
	if _, ok := LookupCategory(q.CategoryID); !ok {
		return nil, apperrors.ErrCategoryUnknown
	}
	if q.EventCode != "" {
		def, ok := LookupEvent(q.EventCode)
		if !ok || def.Category != q.CategoryID {
			return nil, apperrors.ErrEventUnknown
		}
	}
	if q.Limit <= 0 {
		q.Limit = defaultInboxLimit
	}
	if q.Limit > maxInboxLimit {
		q.Limit = maxInboxLimit
	}

	var (
		cursor    model.CursorPayload
		hasCursor bool
	)
	if q.Cursor != "" {
		decoded, err := decodeCursor(q.Cursor)
		if err != nil {
			return nil, apperrors.ErrCursorInvalid
		}
		cursor, hasCursor = decoded, true
	}

	rows, hasMore, err := s.messages.ListInbox(ctx, tenantID, memberID, q, cursor, hasCursor)
	if err != nil {
		return nil, err
	}

	page := &model.InboxPageView{
		Items:           make([]model.InboxItemView, 0, len(rows)),
		HasMore:         hasMore,
		RetentionMonths: s.retentionMonths,
		ServerTime:      kernel.JSONTime(time.Now()),
	}
	for _, row := range rows {
		readAt := kernel.JSONTime{}
		if row.ReadAt != nil {
			readAt = kernel.JSONTime(*row.ReadAt)
		}
		page.Items = append(page.Items, model.InboxItemView{
			ID:         row.InboxID,
			CategoryID: row.CategoryCode,
			EventCode:  row.EventCode,
			EventLabel: EventLabel(row.EventCode),
			Severity:   row.Severity,
			Title:      row.Title,
			Content:    row.Content,
			CreatedAt:  kernel.JSONTime(row.CreatedAt),
			Read:       row.ReadAt != nil,
			ReadAt:     readAt,
			Action:     normalizedAction(row.Action),
		})
	}
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		page.NextCursor = encodeCursor(model.CursorPayload{
			OccurredAtNano: last.OccurredAt.UnixNano(),
			InboxID:        last.InboxID,
		})
	}
	return page, nil
}

// MarkRead 幂等单条已读：不存在的行返回 404（不泄露他人消息存在性）；
// 已读行直接返回摘要，首次已读落 read_at 后重算摘要
func (s *inboxService) MarkRead(
	ctx context.Context, tenantID, memberID, inboxID uint,
) (*model.UnreadSummaryView, error) {
	readAt, exists, err := s.messages.GetInboxReadState(ctx, tenantID, memberID, inboxID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, apperrors.ErrNotFound
	}
	if readAt != nil {
		// 已读幂等：首次 read_at 不被改写，直接返回最新摘要
		return s.UnreadSummary(ctx, tenantID, memberID)
	}
	if _, err = s.messages.MarkInboxRead(ctx, tenantID, memberID, inboxID); err != nil {
		return nil, err
	}
	return s.UnreadSummary(ctx, tenantID, memberID)
}

// MarkAllRead 批量已读：分类必填、事件可选归属校验；through 为前端回传的
// serverTime（东八区秒级），只标记 occurred_at <= through 的旧消息
func (s *inboxService) MarkAllRead(
	ctx context.Context, tenantID, memberID uint, req model.ReadAllRequest,
) (*model.UnreadSummaryView, error) {
	if _, ok := LookupCategory(req.CategoryID); !ok {
		return nil, apperrors.ErrCategoryUnknown
	}
	if req.EventCode != "" {
		def, ok := LookupEvent(req.EventCode)
		if !ok || def.Category != req.CategoryID {
			return nil, apperrors.ErrEventUnknown
		}
	}
	var (
		through    time.Time
		hasThrough bool
	)
	if req.Through != "" {
		parsed, err := time.ParseInLocation("2006-01-02 15:04:05", req.Through, kernel.CSTLocation())
		if err != nil {
			return nil, apperrors.ErrCursorInvalid
		}
		through, hasThrough = parsed, true
	}

	// 单语句 UPDATE 自身原子，事务用于未来批量已读后置处理的边界收口
	err := s.tx.WithinTransaction(ctx, func(tctx context.Context) error {
		_, err := s.messages.MarkCategoryRead(tctx, tenantID, memberID, req.CategoryID, req.EventCode, through, hasThrough)
		return err
	})
	if err != nil {
		return nil, err
	}
	return s.UnreadSummary(ctx, tenantID, memberID)
}

// encodeCursor 不透明游标：base64(JSON 载荷)，固化 (occurred_at 纳秒, 收件箱 id)
func encodeCursor(payload model.CursorPayload) string {
	data, _ := json.Marshal(payload)
	return base64.RawURLEncoding.EncodeToString(data)
}

// decodeCursor 解析并校验游标载荷（字段零值视为非法，防垃圾输入穿透分页）
func decodeCursor(raw string) (model.CursorPayload, error) {
	data, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return model.CursorPayload{}, err
	}
	var payload model.CursorPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return model.CursorPayload{}, err
	}
	if payload.InboxID == 0 || payload.OccurredAtNano <= 0 {
		return model.CursorPayload{}, apperrors.ErrCursorInvalid
	}
	return payload, nil
}

// normalizedAction 动作出网归一：空/NULL 统一输出 {}，前端零分支处理
func normalizedAction(action model.JSONContent) json.RawMessage {
	if len(action) == 0 || string(action) == "null" {
		return json.RawMessage("{}")
	}
	return json.RawMessage(action)
}
