package repository

import (
	"errors"

	"kun-galgame-api/internal/message/model"

	"gorm.io/gorm"
)

type MessageRepository struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) DB() *gorm.DB {
	return r.db
}

// ──────────────────────────────────────────
// Message CRUD
// ──────────────────────────────────────────

type MessageRow struct {
	ID         int
	SenderID   int
	ReceiverID int
	Link       string
	Content    string
	Status     string
	Type       string
	CreatedAt  string
}

func (r *MessageRepository) FindMessages(
	receiverID int,
	msgType, sortOrder string,
	page, limit int,
) ([]MessageRow, int64, error) {
	var rows []MessageRow
	var total int64

	query := r.db.Table("message m").
		Select(`m.id, m.sender_id,
			m.receiver_id, m.link, m.content, m.status, m.type, m.created AS created_at`).
		Where("m.receiver_id = ?", receiverID)

	if msgType != "" {
		query = query.Where("m.type = ?", msgType)
	}

	query.Count(&total)

	err := query.
		Order("m.created " + sortOrder).
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&rows).Error

	return rows, total, err
}

func (r *MessageRepository) DeleteByIDAndReceiver(id, receiverID int) error {
	return r.db.Where("id = ? AND receiver_id = ?", id, receiverID).
		Delete(&model.Message{}).Error
}

func (r *MessageRepository) MarkAllRead(receiverID int) error {
	return r.db.Model(&model.Message{}).
		Where("receiver_id = ? AND status = 'unread'", receiverID).
		Update("status", "read").Error
}

// ──────────────────────────────────────────
// System messages
// ──────────────────────────────────────────

type SystemMessageRow struct {
	ID          int
	ContentEnUS string
	ContentJaJP string
	ContentZhCN string
	ContentZhTW string
	UserID      int
	CreatedAt   string
}

func (r *MessageRepository) FindSystemMessages() ([]SystemMessageRow, error) {
	var rows []SystemMessageRow
	err := r.db.Table("system_message sm").
		Select(`sm.id, sm.content_en_us, sm.content_ja_jp,
			sm.content_zh_cn, sm.content_zh_tw,
			sm.user_id,
			sm.created AS created_at`).
		Order("sm.created DESC").
		Find(&rows).Error
	return rows, err
}

// GetMaxSystemMessageID returns the highest system_message.id currently
// in the table — used by MarkAllSystemRead to advance the user's cursor.
// Returns 0 when there are no broadcasts.
func (r *MessageRepository) GetMaxSystemMessageID() (int64, error) {
	var maxID *int64
	err := r.db.Table("system_message").Select("MAX(id)").Scan(&maxID).Error
	if err != nil || maxID == nil {
		return 0, err
	}
	return *maxID, nil
}

// GetSystemReadCursor returns the user's "read up to" marker, or 0 when
// the user has never marked any broadcast as read.
func (r *MessageRepository) GetSystemReadCursor(userID int) (int64, error) {
	var row model.SystemMessageReadState
	err := r.db.First(&row, "user_id = ?", userID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	return row.LastReadMessageID, err
}

// UpsertSystemReadCursorForward advances the user's read marker.
//
// GREATEST() ensures the cursor only moves forward — stale "mark all
// read" requests from a tab still showing yesterday's max id can't
// rewind a fresher cursor written by another tab. Mirrors the wiki
// repository pattern (see WikiMessageRepository.UpsertForward).
func (r *MessageRepository) UpsertSystemReadCursorForward(userID int, lastReadID int64) error {
	return r.db.Exec(`
		INSERT INTO system_message_read_state (user_id, last_read_message_id, updated_at)
		VALUES (?, ?, NOW())
		ON CONFLICT (user_id) DO UPDATE
		SET last_read_message_id = GREATEST(
		        system_message_read_state.last_read_message_id,
		        EXCLUDED.last_read_message_id
		    ),
		    updated_at = NOW()
	`, userID, lastReadID).Error
}

// ──────────────────────────────────────────
// Nav summary
// ──────────────────────────────────────────

// GetNavSummary builds the [notice, system] nav badges. localMuted/systemMuted
// come from the caller's notification preferences (migration 053): muted
// categories are excluded from the *unread* counts so the badges match the
// top-bar red dot, while the total counts (and the message rows themselves) are
// left intact — muting suppresses the badge, not the record.
func (r *MessageRepository) GetNavSummary(userID int, localMuted []string, systemMuted bool) ([]map[string]any, error) {
	// Notice messages
	var noticeTotal, noticeUnread int64
	r.db.Model(&model.Message{}).Where("receiver_id = ?", userID).Count(&noticeTotal)
	unreadQuery := r.db.Model(&model.Message{}).Where("receiver_id = ? AND status = 'unread'", userID)
	if len(localMuted) > 0 {
		unreadQuery = unreadQuery.Not("type IN ?", localMuted)
	}
	unreadQuery.Count(&noticeUnread)

	var latestNotice model.Message
	noticeResult := r.db.Where("receiver_id = ?", userID).Order("created DESC").First(&latestNotice)

	// System messages — unread is computed against the user's HWM cursor
	// (see migration 012). A missing cursor row means the user has never
	// read anything → treat as cursor=0 so every existing broadcast is
	// unread (matches "new user sees backlog" intent).
	var sysTotal, sysUnread int64
	r.db.Model(&model.SystemMessage{}).Count(&sysTotal)
	if !systemMuted {
		cursor, _ := r.GetSystemReadCursor(userID)
		r.db.Model(&model.SystemMessage{}).Where("id > ?", cursor).Count(&sysUnread)
	}

	var latestSys model.SystemMessage
	sysResult := r.db.Order("created DESC").First(&latestSys)

	noticeContent := ""
	if latestNotice.Content != "" {
		if len(latestNotice.Content) > 100 {
			noticeContent = latestNotice.Content[:100]
		} else {
			noticeContent = latestNotice.Content
		}
	}

	var noticeTime any = ""
	if noticeResult.RowsAffected > 0 {
		noticeTime = latestNotice.CreatedAt
	}
	var sysTime any = ""
	if sysResult.RowsAffected > 0 {
		sysTime = latestSys.CreatedAt
	}

	result := []map[string]any{
		{
			"chatroom_name":     "",
			"content":           noticeContent,
			"last_message_time": noticeTime,
			"count":             noticeTotal,
			"unread_count":      noticeUnread,
			"route":             "notice",
			"title":             "zako~",
			"avatar":            "",
		},
		{
			"chatroom_name":     "",
			"content":           "",
			"last_message_time": sysTime,
			"count":             sysTotal,
			"unread_count":      sysUnread,
			"route":             "system",
			"title":             "zako~",
			"avatar":            "",
		},
	}
	return result, nil
}
