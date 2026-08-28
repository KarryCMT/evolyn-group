package service

import (
	"context"
	"testing"
	"time"

	apperrors "evolyn/internal/platform/notification"
	"evolyn/internal/platform/notification/model"

	"github.com/stretchr/testify/assert"
)

// fakeMessageRepo 内存版消息/收件箱仓储
type fakeMessageRepo struct {
	inboxes map[uint]*model.MemberInbox // inboxID → 行
	nextID  uint
}

func newFakeMessageRepo() *fakeMessageRepo {
	return &fakeMessageRepo{inboxes: map[uint]*model.MemberInbox{}, nextID: 1000}
}

func (f *fakeMessageRepo) Migrate() error { return nil }

func (f *fakeMessageRepo) InsertIgnoreConflict(ctx context.Context, msg *model.Message) (uint, error) {
	f.nextID++
	msg.ID = f.nextID
	return msg.ID, nil
}

func (f *fakeMessageRepo) InsertInboxesIgnoreConflict(
	ctx context.Context, tenantID, messageID uint, categoryCode string, occurredAt time.Time, memberIDs []uint,
) error {
	for _, memberID := range memberIDs {
		f.nextID++
		f.inboxes[f.nextID] = &model.MemberInbox{
			ID: f.nextID, TenantID: tenantID, MessageID: messageID,
			MemberID: memberID, CategoryCode: categoryCode, OccurredAt: occurredAt,
		}
	}
	return nil
}

func (f *fakeMessageRepo) ListInbox(
	ctx context.Context, tenantID, memberID uint, q model.InboxQuery,
	cursor model.CursorPayload, hasCursor bool,
) ([]model.InboxRow, bool, error) {
	rows := make([]model.InboxRow, 0)
	for _, inbox := range f.inboxes {
		if inbox.TenantID != tenantID || inbox.MemberID != memberID || inbox.CategoryCode != q.CategoryID {
			continue
		}
		if q.UnreadOnly && inbox.ReadAt != nil {
			continue
		}
		if hasCursor && inbox.ID >= cursor.InboxID {
			continue
		}
		readAt := time.Time{}
		if inbox.ReadAt != nil {
			readAt = *inbox.ReadAt
		}
		rows = append(rows, model.InboxRow{
			InboxID: inbox.ID, ReadAt: inbox.ReadAt, CategoryCode: inbox.CategoryCode,
			EventCode: "application.asset.changed", Severity: "info", Content: "c",
			OccurredAt: inbox.OccurredAt, CreatedAt: inbox.OccurredAt,
		})
		_ = readAt
	}
	// 简化排序：id 倒序近似（fake 场景 occurred_at 单调）
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
	if len(rows) > q.Limit {
		return rows[:q.Limit], true, nil
	}
	return rows, false, nil
}

func (f *fakeMessageRepo) CountUnreadByCategory(
	ctx context.Context, tenantID, memberID uint,
) ([]model.CategoryUnreadView, error) {
	counts := map[string]int64{}
	for _, inbox := range f.inboxes {
		if inbox.TenantID == tenantID && inbox.MemberID == memberID && inbox.ReadAt == nil {
			counts[inbox.CategoryCode]++
		}
	}
	rows := make([]model.CategoryUnreadView, 0, len(counts))
	for category, count := range counts {
		rows = append(rows, model.CategoryUnreadView{CategoryID: category, UnreadCount: count})
	}
	return rows, nil
}

func (f *fakeMessageRepo) GetInboxReadState(
	ctx context.Context, tenantID, memberID, inboxID uint,
) (*time.Time, bool, error) {
	inbox, ok := f.inboxes[inboxID]
	if !ok || inbox.TenantID != tenantID || inbox.MemberID != memberID {
		return nil, false, nil
	}
	return inbox.ReadAt, true, nil
}

func (f *fakeMessageRepo) MarkInboxRead(ctx context.Context, tenantID, memberID, inboxID uint) (int64, error) {
	inbox, ok := f.inboxes[inboxID]
	if !ok || inbox.TenantID != tenantID || inbox.MemberID != memberID || inbox.ReadAt != nil {
		return 0, nil
	}
	now := time.Now()
	inbox.ReadAt = &now
	return 1, nil
}

func (f *fakeMessageRepo) MarkCategoryRead(
	ctx context.Context, tenantID, memberID uint, categoryCode, eventCode string, through time.Time, hasThrough bool,
) (int64, error) {
	var affected int64
	for _, inbox := range f.inboxes {
		if inbox.TenantID != tenantID || inbox.MemberID != memberID ||
			inbox.CategoryCode != categoryCode || inbox.ReadAt != nil {
			continue
		}
		if hasThrough && inbox.OccurredAt.After(through) {
			continue
		}
		now := time.Now()
		inbox.ReadAt = &now
		affected++
	}
	return affected, nil
}

func (f *fakeMessageRepo) DeleteExpiredInboxes(ctx context.Context, batch int) (int64, error) {
	return 0, nil
}

func (f *fakeMessageRepo) DeleteOrphanExpiredMessages(ctx context.Context, batch int) (int64, error) {
	return 0, nil
}

func newTestInboxService(repo *fakeMessageRepo) *inboxService {
	return newInboxService(passThroughTx{}, repo, 6)
}

func seedInbox(t *testing.T, repo *fakeMessageRepo, tenantID, memberID uint, category string, occurredAt time.Time) uint {
	t.Helper()
	repo.nextID++
	repo.inboxes[repo.nextID] = &model.MemberInbox{
		ID: repo.nextID, TenantID: tenantID, MemberID: memberID,
		CategoryCode: category, OccurredAt: occurredAt,
	}
	return repo.nextID
}

// ---- 收件箱服务 ----

func TestInboxListValidationsAndCursor(t *testing.T) {
	repo := newFakeMessageRepo()
	svc := newTestInboxService(repo)
	ctx := context.Background()

	// 未知分类 / 事件不属于分类 / 非法游标稳定报错
	_, err := svc.ListInbox(ctx, 1, 1, model.InboxQuery{CategoryID: "nope"})
	assert.ErrorIs(t, err, apperrors.ErrCategoryUnknown)
	_, err = svc.ListInbox(ctx, 1, 1, model.InboxQuery{
		CategoryID: CategoryAppLog, EventCode: "edition.subscription.downgrade",
	})
	assert.ErrorIs(t, err, apperrors.ErrEventUnknown)
	_, err = svc.ListInbox(ctx, 1, 1, model.InboxQuery{CategoryID: CategoryAppLog, Cursor: "###"})
	assert.ErrorIs(t, err, apperrors.ErrCursorInvalid)

	// 游标编解码往返
	payload := model.CursorPayload{OccurredAtNano: time.Now().UnixNano(), InboxID: 42}
	assert.Equal(t, payload, mustDecode(t, encodeCursor(payload)))
}

func TestInboxReadFlows(t *testing.T) {
	repo := newFakeMessageRepo()
	svc := newTestInboxService(repo)
	ctx := context.Background()
	const tenantID, memberID = uint(1), uint(11)

	old := time.Now().Add(-time.Hour)
	newer := time.Now()
	appLogID := seedInbox(t, repo, tenantID, memberID, CategoryAppLog, old)
	seedInbox(t, repo, tenantID, memberID, CategoryAppLog, newer)
	seedInbox(t, repo, tenantID, memberID, CategorySystemManagement, old)

	// 他成员/他租户同 ID 不可见（猜 ID 不能标记已读，也不泄露存在性）
	_, err := svc.MarkRead(ctx, tenantID+1, memberID, appLogID)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
	_, err = svc.MarkRead(ctx, tenantID, memberID+1, appLogID)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)

	// 首次已读 + 幂等（摘要返回不重复计数）
	summary, err := svc.MarkRead(ctx, tenantID, memberID, appLogID)
	assert.NoError(t, err)
	assert.EqualValues(t, 2, summary.UnreadTotal)
	firstReadAt := repo.inboxes[appLogID].ReadAt
	assert.NotNil(t, firstReadAt)
	_, err = svc.MarkRead(ctx, tenantID, memberID, appLogID)
	assert.NoError(t, err)
	assert.Equal(t, firstReadAt, repo.inboxes[appLogID].ReadAt, "重复已读不得改写首次时间")

	// 批量已读：仅当前分类 + through 之前的旧消息；不影响其他分类与新到达
	through := old.Add(time.Minute).Format("2006-01-02 15:04:05")
	summary, err = svc.MarkAllRead(ctx, tenantID, memberID, model.ReadAllRequest{
		CategoryID: CategoryAppLog, Through: through,
	})
	assert.NoError(t, err)
	// 剩余未读：app-log 的新消息 + system-management 的旧消息
	assert.EqualValues(t, 2, summary.UnreadTotal)
	var appLogUnread, systemUnread int64
	for _, category := range summary.Categories {
		if category.CategoryID == CategoryAppLog {
			appLogUnread = category.UnreadCount
		}
		if category.CategoryID == CategorySystemManagement {
			systemUnread = category.UnreadCount
		}
	}
	assert.EqualValues(t, 1, appLogUnread)
	assert.EqualValues(t, 1, systemUnread)

	// through 非法格式 → 游标口径错误码拒绝
	_, err = svc.MarkAllRead(ctx, tenantID, memberID, model.ReadAllRequest{
		CategoryID: CategoryAppLog, Through: "not-a-time",
	})
	assert.ErrorIs(t, err, apperrors.ErrCursorInvalid)
}

func mustDecode(t *testing.T, raw string) model.CursorPayload {
	t.Helper()
	payload, err := decodeCursor(raw)
	assert.NoError(t, err)
	return payload
}
