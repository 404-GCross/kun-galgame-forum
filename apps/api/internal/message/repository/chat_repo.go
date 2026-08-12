package repository

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

type ChatRepository struct {
	db *gorm.DB
}

func NewChatRepository(db *gorm.DB) *ChatRepository {
	return &ChatRepository{db: db}
}

func (r *ChatRepository) DB() *gorm.DB {
	return r.db
}

type RoomListRow struct {
	ID                 int     `gorm:"column:id"`
	Name               string  `gorm:"column:name"`
	Avatar             string  `gorm:"column:avatar"`
	Type               string  `gorm:"column:type"`
	LastMessageContent string  `gorm:"column:last_message_content"`
	LastMessageTime    *string `gorm:"column:last_message_time"`
}

type ParticipantRow struct {
	ChatRoomID int `gorm:"column:chat_room_id"`
	UserID     int `gorm:"column:user_id"`
}

type CountRow struct {
	ChatRoomID int `gorm:"column:chat_room_id"`
	Count      int `gorm:"column:count"`
}

type RoomRef struct {
	ID   int    `gorm:"column:id"`
	Name string `gorm:"column:name"`
}

type ChatMessageRow struct {
	ID           int     `gorm:"column:id"`
	ChatroomName string  `gorm:"column:chatroom_name"`
	SenderID     int     `gorm:"column:sender_id"`
	ReceiverID   int     `gorm:"column:receiver_id"`
	Content      string  `gorm:"column:content"`
	IsRecall     bool    `gorm:"column:is_recall"`
	Created      string  `gorm:"column:created"`
	RecallTime   *string `gorm:"column:recall_time"`
	EditTime     *string `gorm:"column:edit_time"`
}

func (r *ChatRepository) FindRoomsForUser(userID int) ([]RoomListRow, error) {
	var rooms []RoomListRow
	err := r.db.Table("chat_room cr").
		Select(`cr.id, cr.name, cr.avatar, cr.type,
			cr.last_message_content, cr.last_message_time`).
		Joins("JOIN chat_room_participant crp ON crp.chat_room_id = cr.id").
		Where("crp.user_id = ? AND cr.last_message_sender_id != 0 AND cr.last_message_time IS NOT NULL", userID).
		Order("cr.last_message_time DESC").
		Scan(&rooms).Error
	return rooms, err
}

func (r *ChatRepository) FindParticipantsByRoomIDs(roomIDs []int) []ParticipantRow {
	var rows []ParticipantRow
	r.db.Table("chat_room_participant p").
		Select("p.chat_room_id, p.user_id").
		Where("p.chat_room_id IN ?", roomIDs).
		Scan(&rows)
	return rows
}

func (r *ChatRepository) CountUnreadByRoomIDs(roomIDs []int, userID int) []CountRow {
	var rows []CountRow
	r.db.Table("chat_message cm").
		Select("cm.chat_room_id, COUNT(*) AS count").
		Where("cm.chat_room_id IN ? AND cm.sender_id != ?", roomIDs, userID).
		Where("cm.id NOT IN (SELECT chat_message_id FROM chat_message_read_by WHERE user_id = ?)", userID).
		Group("cm.chat_room_id").
		Scan(&rows)
	return rows
}

func (r *ChatRepository) CountTotalByRoomIDs(roomIDs []int) []CountRow {
	var rows []CountRow
	r.db.Table("chat_message").
		Select("chat_room_id, COUNT(*) AS count").
		Where("chat_room_id IN ?", roomIDs).
		Group("chat_room_id").
		Scan(&rows)
	return rows
}

func (r *ChatRepository) FindPrivateRoomBetween(uid1, uid2 int) RoomRef {
	var room RoomRef
	r.db.Raw(`
		SELECT cr.id, cr.name FROM chat_room cr
		WHERE cr.type = 'private'
		AND cr.id IN (
			SELECT chat_room_id FROM chat_room_participant WHERE user_id = ?
		)
		AND cr.id IN (
			SELECT chat_room_id FROM chat_room_participant WHERE user_id = ?
		)
		LIMIT 1`, uid1, uid2).Scan(&room)
	return room
}

func (r *ChatRepository) CreatePrivateRoom(roomName string, uid1, uid2 int) (RoomRef, error) {
	var room RoomRef
	now := time.Now()
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(
			`INSERT INTO chat_room (name, type, created, updated) VALUES (?, 'private', ?, ?)`,
			roomName, now, now,
		).Error; err != nil {
			return err
		}
		if err := tx.Raw(`SELECT id, name FROM chat_room WHERE name = ?`, roomName).Scan(&room).Error; err != nil {
			return err
		}
		if room.ID > 0 {
			if err := tx.Exec(
				`INSERT INTO chat_room_participant (chat_room_id, user_id, created, updated) VALUES (?, ?, ?, ?), (?, ?, ?, ?)`,
				room.ID, uid1, now, now, room.ID, uid2, now, now,
			).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return room, err
}

func (r *ChatRepository) FindMessagesByRoom(roomID int, roomName string, page, limit int) []ChatMessageRow {
	var rows []ChatMessageRow
	offset := (page - 1) * limit
	r.db.Table("chat_message cm").
		Select(`cm.id, cm.chatroom_name, cm.sender_id,
			cm.receiver_id, cm.content, cm.is_recall,
			cm.created, cm.recall_time, cm.edit_time`).
		Where("cm.chat_room_id = ? OR cm.chatroom_name = ?", roomID, roomName).
		Order("cm.id DESC").
		Offset(offset).Limit(limit).
		Scan(&rows)
	return rows
}

type MessageHeader struct {
	ID           int
	ChatRoomID   int    `gorm:"column:chat_room_id"`
	ChatroomName string `gorm:"column:chatroom_name"`
	SenderID     int    `gorm:"column:sender_id"`
	IsRecall     bool   `gorm:"column:is_recall"`
}

func (r *ChatRepository) FindMessageHeader(id int) (MessageHeader, bool) {
	var h MessageHeader
	err := r.db.Table("chat_message m").
		Select(`m.id, m.chat_room_id, m.chatroom_name, m.sender_id, m.is_recall`).
		Where("m.id = ?", id).
		Scan(&h).Error
	if err != nil || h.ID == 0 {
		return MessageHeader{}, false
	}
	return h, true
}

func (r *ChatRepository) MarkMessageRecalled(id int, now time.Time) error {
	return r.db.Exec(
		`UPDATE chat_message SET is_recall = TRUE, recall_time = ?, updated = ? WHERE id = ?`,
		now, now, id,
	).Error
}

func (r *ChatRepository) IsLatestMessageInRoom(roomID, msgID int) bool {
	var latest int
	r.db.Table("chat_message").
		Select("MAX(id)").
		Where("chat_room_id = ?", roomID).
		Scan(&latest)
	return latest == msgID
}

func (r *ChatRepository) MarkMessagesRead(msgIDs []int, userID int) error {
	if len(msgIDs) == 0 {
		return nil
	}
	now := time.Now()
	placeholders := make([]string, 0, len(msgIDs))
	args := make([]any, 0, len(msgIDs)*4)
	for _, mid := range msgIDs {
		placeholders = append(placeholders, "(?, ?, ?, ?)")
		args = append(args, mid, userID, now, now)
	}
	sql := `INSERT INTO chat_message_read_by (chat_message_id, user_id, created, updated) VALUES ` +
		strings.Join(placeholders, ", ") + ` ON CONFLICT DO NOTHING`
	return r.db.Exec(sql, args...).Error
}

func (r *ChatRepository) InsertChatMessage(db *gorm.DB, roomID int, roomName string, senderID, receiverID int, content string, now time.Time) error {
	return db.Exec(
		`INSERT INTO chat_message (chat_room_id, chatroom_name, sender_id, receiver_id, content, created, updated)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		roomID, roomName, senderID, receiverID, content, now, now,
	).Error
}

func (r *ChatRepository) UpdateRoomLastMessage(db *gorm.DB, roomID int, content string, senderID int, senderName string, now time.Time) error {
	return db.Exec(
		`UPDATE chat_room SET last_message_content = ?, last_message_time = ?,
		last_message_sender_id = ?, last_message_sender_name = ?, updated = ? WHERE id = ?`,
		content, now, senderID, senderName, now, roomID,
	).Error
}
