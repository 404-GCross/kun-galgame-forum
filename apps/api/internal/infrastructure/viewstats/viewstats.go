package viewstats

import (
	"fmt"

	"gorm.io/gorm"
)

const (
	GalgameDaily = "galgame_view_daily"
	TopicDaily   = "topic_view_daily"
	QuizDaily    = "galgame_quiz_view_daily"
)

const keepDays = 35

var pairs = []struct{ daily, entity string }{
	{GalgameDaily, "galgame"},
	{TopicDaily, "topic"},
	{QuizDaily, "galgame_quiz"},
}

func BumpDaily(db *gorm.DB, table string, entityID int) error {
	return db.Exec(fmt.Sprintf(
		`INSERT INTO %s (entity_id, day, count) VALUES (?, CURRENT_DATE, 1)
		 ON CONFLICT (entity_id, day) DO UPDATE SET count = %s.count + 1`,
		table, table), entityID).Error
}

func RunRollup(db *gorm.DB) error {
	for _, p := range pairs {
		if err := rollupOne(db, p.daily, p.entity); err != nil {
			return fmt.Errorf("rollup %s: %w", p.entity, err)
		}
	}
	return nil
}

func rollupOne(db *gorm.DB, daily, entity string) error {
	if err := db.Exec(fmt.Sprintf(`
		WITH agg AS (
			SELECT entity_id,
				COALESCE(SUM(count) FILTER (WHERE day >= CURRENT_DATE - INTERVAL '6 days'), 0)  AS v7,
				COALESCE(SUM(count) FILTER (WHERE day >= CURRENT_DATE - INTERVAL '29 days'), 0) AS v30
			FROM %s
			WHERE day >= CURRENT_DATE - INTERVAL '29 days'
			GROUP BY entity_id
		)
		UPDATE %s e SET view_7d = agg.v7, view_30d = agg.v30
		FROM agg WHERE e.id = agg.entity_id`, daily, entity)).Error; err != nil {
		return err
	}
	if err := db.Exec(fmt.Sprintf(`
		UPDATE %s e SET view_7d = 0, view_30d = 0
		WHERE (e.view_7d <> 0 OR e.view_30d <> 0)
		  AND NOT EXISTS (
			SELECT 1 FROM %s d
			WHERE d.entity_id = e.id AND d.day >= CURRENT_DATE - INTERVAL '29 days')`,
		entity, daily)).Error; err != nil {
		return err
	}
	return db.Exec(fmt.Sprintf(
		`DELETE FROM %s WHERE day < CURRENT_DATE - INTERVAL '%d days'`, daily, keepDays)).Error
}
