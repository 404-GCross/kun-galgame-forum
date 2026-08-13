package service

import (
	"fmt"

	msgModel "kun-galgame-api/internal/message/model"
	"kun-galgame-api/internal/moemoepoint"

	"gorm.io/gorm"
)

type InteractionHelpers struct{}

func (InteractionHelpers) AdjustMoemoepoint(_ *gorm.DB, userID, delta int, reason, ref string) {
	moemoepoint.Award(userID, delta, reason, ref, moemoepoint.KeyNonce(reason, ref))
}

func (InteractionHelpers) CreateGalgameMessageWithContent(
	tx *gorm.DB,
	senderID, receiverID int,
	msgType, content string,
	galgameID int,
) {
	if senderID == receiverID || receiverID <= 0 {
		return
	}
	link := fmt.Sprintf("/galgame/%d", galgameID)

	var count int64
	tx.Model(&msgModel.Message{}).
		Where("sender_id = ? AND receiver_id = ? AND type = ? AND link = ?",
			senderID, receiverID, msgType, link).
		Count(&count)
	if count > 0 {
		return
	}

	tx.Create(&msgModel.Message{
		SenderID: senderID, ReceiverID: receiverID,
		Type: msgType, Content: content, Link: link, Status: "unread",
	})
}

func (InteractionHelpers) CreateQuizAnswerMessage(
	tx *gorm.DB,
	senderID, receiverID int,
	content string,
	quizID int,
) {
	if senderID == receiverID || receiverID <= 0 {
		return
	}
	link := fmt.Sprintf("/galgame-quiz/%d", quizID)

	var count int64
	tx.Model(&msgModel.Message{}).
		Where("sender_id = ? AND receiver_id = ? AND type = ? AND link = ?",
			senderID, receiverID, "quiz-answered", link).
		Count(&count)
	if count > 0 {
		return
	}

	tx.Create(&msgModel.Message{
		SenderID: senderID, ReceiverID: receiverID,
		Type: "quiz-answered", Content: content, Link: link, Status: "unread",
	})
}

func (InteractionHelpers) CreateGalgameCommentMention(
	tx *gorm.DB,
	senderID, receiverID int,
	content string,
	galgameID, commentID, rootID int,
) {
	if senderID == receiverID || receiverID <= 0 {
		return
	}
	link := fmt.Sprintf("/galgame/%d?comment=%d&thread=%d", galgameID, commentID, rootID)

	var count int64
	tx.Model(&msgModel.Message{}).
		Where("sender_id = ? AND receiver_id = ? AND type = ? AND link = ?",
			senderID, receiverID, "mentioned", link).
		Count(&count)
	if count > 0 {
		return
	}

	tx.Create(&msgModel.Message{
		SenderID: senderID, ReceiverID: receiverID,
		Type: "mentioned", Content: content, Link: link, Status: "unread",
	})
}
