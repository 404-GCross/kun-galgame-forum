package repository

import (
	"fmt"

	"kun-galgame-api/internal/admin/dto"

	"gorm.io/gorm"
)

type PurgeRepository struct {
	db *gorm.DB
}

func NewPurgeRepository(db *gorm.DB) *PurgeRepository {
	return &PurgeRepository{db: db}
}

func (r *PurgeRepository) CountUserContent(userID int) dto.UserContentStats {
	return r.counts(r.db, userID)
}

func (r *PurgeRepository) counts(q *gorm.DB, userID int) dto.UserContentStats {
	var s dto.UserContentStats
	countBy := func(table, col string) int64 {
		var n int64
		q.Table(table).Where(col+" = ?", userID).Count(&n)
		return n
	}

	s.Topics = countBy("topic", "user_id")
	s.Replies = countBy("topic_reply", "user_id")
	s.TopicComments = countBy("topic_comment", "user_id")
	s.Ratings = countBy("galgame_rating", "user_id")
	s.Resources = countBy("galgame_resource", "user_id")
	s.Websites = countBy("galgame_website", "user_id")
	s.Toolsets = countBy("galgame_toolset", "user_id")
	s.ToolsetResources = countBy("galgame_toolset_resource", "user_id")
	s.ChatMessages = countBy("chat_message", "sender_id")
	s.Messages = countBy("message", "sender_id") + countBy("message", "receiver_id")

	for _, t := range interactionTables {
		s.Interactions += countBy(t, "user_id")
	}

	s.Total = s.Topics + s.Replies + s.TopicComments +
		s.Ratings + s.Resources + s.Websites +
		s.Toolsets + s.ToolsetResources +
		s.ChatMessages + s.Messages + s.Interactions
	return s
}

var interactionTables = []string{
	"topic_like", "topic_dislike", "topic_favorite", "topic_upvote",
	"topic_reply_like", "topic_reply_dislike", "topic_comment_like",
	"topic_poll_vote",
	"galgame_like", "galgame_favorite",
	"galgame_post_like",
	"galgame_rating_like", "galgame_resource_like",
	"galgame_website_like", "galgame_website_favorite",
	"galgame_toolset_practicality", "galgame_toolset_contributor",
}

func (r *PurgeRepository) PurgeUserContent(userID int) (dto.UserContentStats, error) {
	stats := r.counts(r.db, userID)

	err := r.db.Transaction(func(tx *gorm.DB) error {
		var affTopics, affReplies, affGalgames, affChatRooms []int
		captures := []struct {
			dst *[]int
			sql string
		}{
			{&affTopics, `SELECT DISTINCT topic_id FROM topic_reply WHERE user_id = ?
				UNION SELECT DISTINCT topic_id FROM topic_comment WHERE user_id = ?`},
			{&affReplies, `SELECT DISTINCT topic_reply_id FROM topic_comment WHERE user_id = ?`},
			{&affGalgames, `SELECT DISTINCT galgame_id FROM galgame_rating WHERE user_id = ?
				UNION SELECT DISTINCT galgame_id FROM galgame_resource WHERE user_id = ?`},
			{&affChatRooms, `SELECT DISTINCT chat_room_id FROM chat_message WHERE sender_id = ?
				UNION SELECT chat_room_id FROM chat_room_participant WHERE user_id = ?`},
		}
		for _, c := range captures {
			args := make([]any, countPlaceholders(c.sql))
			for i := range args {
				args[i] = userID
			}
			if err := tx.Raw(c.sql, args...).Scan(c.dst).Error; err != nil {
				return err
			}
		}

		for _, q := range []string{
			"DELETE FROM topic WHERE user_id = ?",
			"DELETE FROM galgame_website WHERE user_id = ?",
			"DELETE FROM galgame_toolset WHERE user_id = ?",
			"DELETE FROM topic_poll WHERE user_id = ?",
		} {
			if err := del(tx, q, userID); err != nil {
				return err
			}
		}

		for _, q := range []string{
			"DELETE FROM topic_reply WHERE user_id = ?",
			"DELETE FROM topic_comment WHERE user_id = ?",
			"DELETE FROM galgame_rating WHERE user_id = ?",
			"DELETE FROM galgame_resource WHERE user_id = ?",
			"DELETE FROM galgame_toolset_resource WHERE user_id = ?",
		} {
			if err := del(tx, q, userID); err != nil {
				return err
			}
		}

		for _, it := range interactionDeletes {
			var ids []int
			if it.parentTable != "" {
				if err := tx.Raw(
					"SELECT DISTINCT "+it.parentCol+" FROM "+it.table+" WHERE user_id = ?", userID,
				).Scan(&ids).Error; err != nil {
					return err
				}
			}
			if err := del(tx, "DELETE FROM "+it.table+" WHERE user_id = ?", userID); err != nil {
				return err
			}
			if it.parentTable != "" {
				if err := recount(tx, it.parentTable, it.countCol, it.table, it.parentCol, ids); err != nil {
					return err
				}
			}
		}

		for _, q := range []string{
			"DELETE FROM chat_message WHERE sender_id = ?",
			"DELETE FROM chat_message_reaction WHERE user_id = ?",
			"DELETE FROM chat_message_read_by WHERE user_id = ?",
			"DELETE FROM chat_room_admin WHERE user_id = ?",
			"DELETE FROM chat_room_participant WHERE user_id = ?",
		} {
			if err := del(tx, q, userID); err != nil {
				return err
			}
		}
		if len(affChatRooms) > 0 {
			if err := del(tx, `UPDATE chat_room cr SET
					last_message_content = lm.content,
					last_message_time = lm.created,
					last_message_sender_id = lm.sender_id,
					last_message_sender_name = ''
				FROM (
					SELECT DISTINCT ON (chat_room_id) chat_room_id, content, created, sender_id
					FROM chat_message WHERE chat_room_id IN ?
					ORDER BY chat_room_id, created DESC
				) lm
				WHERE cr.id = lm.chat_room_id`, affChatRooms); err != nil {
				return err
			}
			if err := del(tx, `DELETE FROM chat_room WHERE id IN ?
				AND NOT EXISTS (SELECT 1 FROM chat_message m WHERE m.chat_room_id = chat_room.id)`,
				affChatRooms); err != nil {
				return err
			}
		}

		if err := del(tx, "DELETE FROM message WHERE sender_id = ? OR receiver_id = ?", userID, userID); err != nil {
			return err
		}
		for _, q := range []string{
			"DELETE FROM system_message WHERE user_id = ?",
			"DELETE FROM system_message_read_state WHERE user_id = ?",
			"DELETE FROM wiki_message_read_state WHERE user_id = ?",
		} {
			if err := del(tx, q, userID); err != nil {
				return err
			}
		}

		for _, q := range []string{
			"DELETE FROM user_follow WHERE follower_id = ? OR followed_id = ?",
			"DELETE FROM user_friend WHERE user_id = ? OR friend_id = ?",
		} {
			if err := del(tx, q, userID, userID); err != nil {
				return err
			}
		}

		if err := del(tx, "DELETE FROM kungal_user_state WHERE user_id = ?", userID); err != nil {
			return err
		}

		recounts := []struct {
			parentTable, countCol, childTable, parentCol string
			ids                                          []int
		}{
			{"topic", "reply_count", "topic_reply", "topic_id", affTopics},
			{"topic", "comment_count", "topic_comment", "topic_id", affTopics},
			{"topic_reply", "comment_count", "topic_comment", "topic_reply_id", affReplies},
			{"galgame", "rating_count", "galgame_rating", "galgame_id", affGalgames},
			{"galgame", "resource_count", "galgame_resource", "galgame_id", affGalgames},
		}
		for _, rc := range recounts {
			if err := recount(tx, rc.parentTable, rc.countCol, rc.childTable, rc.parentCol, rc.ids); err != nil {
				return err
			}
		}

		return nil
	})

	return stats, err
}

type interactionDelete struct {
	table       string
	parentCol   string
	parentTable string
	countCol    string
}

var interactionDeletes = []interactionDelete{
	{"topic_like", "topic_id", "topic", "like_count"},
	{"topic_dislike", "topic_id", "topic", "dislike_count"},
	{"topic_favorite", "topic_id", "topic", "favorite_count"},
	{"topic_upvote", "topic_id", "topic", "upvote_count"},
	{"topic_reply_like", "topic_reply_id", "topic_reply", "like_count"},
	{"topic_reply_dislike", "topic_reply_id", "topic_reply", "dislike_count"},
	{"topic_comment_like", "topic_comment_id", "", ""},
	{"topic_poll_vote", "option_id", "topic_poll_option", "vote_count"},
	{"galgame_like", "galgame_id", "galgame", "like_count"},
	{"galgame_favorite", "galgame_id", "galgame", "favorite_count"},
	{"galgame_post_like", "post_id", "", ""},
	{"galgame_rating_like", "galgame_rating_id", "galgame_rating", "like_count"},
	{"galgame_resource_like", "galgame_resource_id", "galgame_resource", "like_count"},
	{"galgame_website_like", "website_id", "galgame_website", "like_count"},
	{"galgame_website_favorite", "website_id", "galgame_website", "favorite_count"},
	{"galgame_toolset_practicality", "toolset_id", "", ""},
	{"galgame_toolset_contributor", "toolset_id", "", ""},
}

func del(tx *gorm.DB, query string, args ...any) error {
	return tx.Exec(query, args...).Error
}

func recount(tx *gorm.DB, parentTable, countCol, childTable, parentCol string, ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	sql := fmt.Sprintf(
		"UPDATE %s SET %s = (SELECT COUNT(*) FROM %s WHERE %s.%s = %s.id) WHERE %s.id IN ?",
		parentTable, countCol, childTable, childTable, parentCol, parentTable, parentTable,
	)
	return tx.Exec(sql, ids).Error
}

func countPlaceholders(sql string) int {
	n := 0
	for _, c := range sql {
		if c == '?' {
			n++
		}
	}
	return n
}
