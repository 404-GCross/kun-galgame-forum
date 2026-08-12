package service

import (
	"fmt"

	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/infrastructure/markdown"
	msgModel "kun-galgame-api/internal/message/model"
	msgService "kun-galgame-api/internal/message/service"
	"kun-galgame-api/internal/moemoepoint"

	"gorm.io/gorm"
)

type InteractionHelpers struct{}

func (InteractionHelpers) AdjustMoemoepoint(_ *gorm.DB, userID, delta int, reason, ref string) {
	moemoepoint.Award(userID, delta, reason, ref, moemoepoint.KeyNonce(reason, ref))
}

func (InteractionHelpers) CreateTopicMessage(tx *gorm.DB, senderID, receiverID int, msgType string, topicID int) {
	if senderID == receiverID || receiverID <= 0 {
		return
	}
	link := fmt.Sprintf("/topic/%d", topicID)
	createDedupMessage(tx, senderID, receiverID, msgType, "", link)
}

func (InteractionHelpers) CreateTopicMessageWithContent(
	tx *gorm.DB,
	senderID, receiverID int,
	msgType, content string,
	topicID, replyFloor, commentID int,
) {
	if senderID == receiverID || receiverID <= 0 {
		return
	}
	link := msgService.BuildTopicLink(topicID, replyFloor, commentID)
	createDedupMessage(tx, senderID, receiverID, msgType, content, link)
}

func (InteractionHelpers) CreateReplyMessage(
	tx *gorm.DB,
	senderID, receiverID int,
	msgType, content string,
	topicID, replyFloor, commentID int,
) {
	if senderID == receiverID || receiverID <= 0 {
		return
	}
	link := msgService.BuildTopicLink(topicID, replyFloor, commentID)
	tx.Create(&msgModel.Message{
		SenderID:   senderID,
		ReceiverID: receiverID,
		Type:       msgType,
		Content:    content,
		Link:       link,
		Status:     "unread",
	})
}

func (h InteractionHelpers) NotifyMentions(tx *gorm.DB, senderID, topicID, replyFloor, commentID int, content string) {
	preview := truncate(markdown.StripReferenceTokens(content), constants.TextPreviewLength)
	for _, uid := range markdown.ExtractMentionIDs(content) {
		h.CreateTopicMessageWithContent(tx, senderID, uid, "mentioned", preview, topicID, replyFloor, commentID)
	}
}

func createDedupMessage(tx *gorm.DB, senderID, receiverID int, msgType, content, link string) {
	if senderID == receiverID || receiverID <= 0 {
		return
	}
	var count int64
	tx.Model(&msgModel.Message{}).
		Where("sender_id = ? AND receiver_id = ? AND type = ? AND link = ?",
			senderID, receiverID, msgType, link).
		Count(&count)
	if count > 0 {
		return
	}
	tx.Create(&msgModel.Message{
		SenderID:   senderID,
		ReceiverID: receiverID,
		Type:       msgType,
		Content:    content,
		Link:       link,
		Status:     "unread",
	})
}

func recomputeTopicCounts(tx *gorm.DB, topicID int) error {
	return tx.Exec(`
		UPDATE topic SET
			reply_count   = (SELECT COUNT(*) FROM topic_reply  WHERE topic_id = ? AND status = 0),
			comment_count = (SELECT COUNT(*) FROM topic_comment WHERE topic_id = ? AND status = 0)
		WHERE id = ?`, topicID, topicID, topicID).Error
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen])
}
