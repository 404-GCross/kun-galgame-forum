package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	apply := flag.Bool("apply", false, "write the changes (default: dry-run, counts only)")
	flag.Parse()

	_ = godotenv.Load()
	db, err := gorm.Open(postgres.Open(os.Getenv("KUN_DATABASE_URL")),
		&gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		fmt.Println("连接数据库失败:", err)
		os.Exit(1)
	}

	const replyMatch = `(SELECT r.floor FROM topic_reply r
		WHERE r.user_id = m.sender_id
		  AND r.topic_id = substring(m.link from '^/topic/([0-9]+)$')::int
		  AND r.created BETWEEN m.created - interval '60 seconds' AND m.created + interval '60 seconds'
		ORDER BY abs(extract(epoch FROM (r.created - m.created))) LIMIT 1)`
	const commentMatch = `(SELECT c.id FROM topic_comment c
		WHERE c.user_id = m.sender_id
		  AND c.topic_id = substring(m.link from '^/topic/([0-9]+)$')::int
		  AND c.created BETWEEN m.created - interval '60 seconds' AND m.created + interval '60 seconds'
		ORDER BY abs(extract(epoch FROM (c.created - m.created))) LIMIT 1)`

	const likedReplyMatch = `(SELECT r.floor FROM topic_reply r
		JOIN topic_reply_reaction rr ON rr.topic_reply_id = r.id
		  AND rr.user_id = m.sender_id AND rr.reaction = 'like'
		WHERE r.topic_id = substring(m.link from '^/topic/([0-9]+)$')::int
		  AND r.user_id = m.receiver_id
		ORDER BY r.id DESC LIMIT 1)`
	const likedCommentMatch = `(SELECT cc.id FROM topic_comment cc
		JOIN topic_comment_like cl ON cl.topic_comment_id = cc.id AND cl.user_id = m.sender_id
		WHERE cc.topic_id = substring(m.link from '^/topic/([0-9]+)$')::int
		  AND cc.user_id = m.receiver_id
		ORDER BY cc.id DESC LIMIT 1)`

	const bare = `m.link ~ '^/topic/[0-9]+$'`

	steps := []struct {
		label    string
		countSQL string
		applySQL string
	}{
		{
			"replied→reply",
			`SELECT count(*) FROM message m WHERE m.type='replied' AND ` + bare + ` AND ` + replyMatch + ` IS NOT NULL`,
			`UPDATE message m SET link = m.link || '?reply=' || ` + replyMatch + ` WHERE m.type='replied' AND ` + bare + ` AND ` + replyMatch + ` IS NOT NULL`,
		},
		{
			"commented→comment",
			`SELECT count(*) FROM message m WHERE m.type='commented' AND ` + bare + ` AND ` + commentMatch + ` IS NOT NULL`,
			`UPDATE message m SET link = m.link || '?comment=' || ` + commentMatch + ` WHERE m.type='commented' AND ` + bare + ` AND ` + commentMatch + ` IS NOT NULL`,
		},
		{
			"mentioned→reply",
			`SELECT count(*) FROM message m WHERE m.type='mentioned' AND ` + bare + ` AND ` + replyMatch + ` IS NOT NULL`,
			`UPDATE message m SET link = m.link || '?reply=' || ` + replyMatch + ` WHERE m.type='mentioned' AND ` + bare + ` AND ` + replyMatch + ` IS NOT NULL`,
		},
		{
			"mentioned→comment",
			`SELECT count(*) FROM message m WHERE m.type='mentioned' AND ` + bare + ` AND ` + replyMatch + ` IS NULL AND ` + commentMatch + ` IS NOT NULL`,
			`UPDATE message m SET link = m.link || '?comment=' || ` + commentMatch + ` WHERE m.type='mentioned' AND ` + bare + ` AND ` + replyMatch + ` IS NULL AND ` + commentMatch + ` IS NOT NULL`,
		},
		{
			"solution→best-answer",
			`SELECT count(*) FROM message m JOIN topic t ON t.id = substring(m.link from '^/topic/([0-9]+)$')::int WHERE m.type='solution' AND ` + bare + ` AND t.best_answer_id IS NOT NULL`,
			`UPDATE message m SET link = m.link || '?reply=' || r.floor FROM topic t JOIN topic_reply r ON r.id = t.best_answer_id WHERE m.type='solution' AND ` + bare + ` AND t.id = substring(m.link from '^/topic/([0-9]+)$')::int AND t.best_answer_id IS NOT NULL`,
		},
		{
			"pin-reply→pinned",
			`SELECT count(*) FROM message m JOIN topic t ON t.id = substring(m.link from '^/topic/([0-9]+)$')::int WHERE m.type='pin-reply' AND ` + bare + ` AND t.pinned_reply_id IS NOT NULL`,
			`UPDATE message m SET link = m.link || '?reply=' || r.floor FROM topic t JOIN topic_reply r ON r.id = t.pinned_reply_id WHERE m.type='pin-reply' AND ` + bare + ` AND t.id = substring(m.link from '^/topic/([0-9]+)$')::int AND t.pinned_reply_id IS NOT NULL`,
		},
		{
			"liked→reply",
			`SELECT count(*) FROM message m WHERE m.type='liked' AND ` + bare + ` AND ` + likedReplyMatch + ` IS NOT NULL`,
			`UPDATE message m SET link = m.link || '?reply=' || ` + likedReplyMatch + ` WHERE m.type='liked' AND ` + bare + ` AND ` + likedReplyMatch + ` IS NOT NULL`,
		},
		{
			"liked→comment",
			`SELECT count(*) FROM message m WHERE m.type='liked' AND ` + bare + ` AND ` + likedReplyMatch + ` IS NULL AND ` + likedCommentMatch + ` IS NOT NULL`,
			`UPDATE message m SET link = m.link || '?comment=' || ` + likedCommentMatch + ` WHERE m.type='liked' AND ` + bare + ` AND ` + likedReplyMatch + ` IS NULL AND ` + likedCommentMatch + ` IS NOT NULL`,
		},
	}

	if *apply {
		fmt.Println("回填消息深链接（--apply 实际写入）:")
	} else {
		fmt.Println("回填消息深链接（dry-run，仅统计；加 --apply 实际写入）:")
	}

	var total int64
	for _, s := range steps {
		if *apply {
			res := db.Exec(s.applySQL)
			if res.Error != nil {
				fmt.Printf("  %-22s 失败: %v\n", s.label, res.Error)
				os.Exit(1)
			}
			fmt.Printf("  %-22s 回填 %d 条\n", s.label, res.RowsAffected)
			total += res.RowsAffected
		} else {
			var n int64
			if err := db.Raw(s.countSQL).Scan(&n).Error; err != nil {
				fmt.Printf("  %-22s 失败: %v\n", s.label, err)
				os.Exit(1)
			}
			fmt.Printf("  %-22s 可回填 %d 条\n", s.label, n)
			total += n
		}
	}

	var leftover int64
	db.Raw(`SELECT count(*) FROM message m WHERE ` + bare +
		` AND m.type IN ('replied','commented','mentioned','solution','pin-reply','liked')`).Scan(&leftover)

	if *apply {
		fmt.Printf("合计回填 %d 条；仍为话题根链接 %d 条（目标已删除/已变更，无法恢复）。\n", total, leftover)
	} else {
		fmt.Printf("合计可回填 %d 条；执行后预计仍为话题根链接 %d 条。\n", total, leftover-total)
	}
}
