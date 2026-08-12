package repository

import (
	"fmt"

	"gorm.io/gorm"
)

type OverviewRepository struct {
	db *gorm.DB
}

func NewOverviewRepository(db *gorm.DB) *OverviewRepository {
	return &OverviewRepository{db: db}
}

type DailyStat struct {
	Date  string `gorm:"column:date"`
	Count int64  `gorm:"column:count"`
}

func (r *OverviewRepository) CountTable(table string) (int64, error) {
	var count int64
	err := r.db.Table(table).Count(&count).Error
	return count, err
}

func (r *OverviewRepository) DailyCountsSince(table string, since any) ([]DailyStat, error) {
	var stats []DailyStat
	err := r.db.Raw(fmt.Sprintf(`
		SELECT date_trunc('day', created)::date::text AS date, COUNT(*) AS count
		FROM %s WHERE created >= ? GROUP BY 1 ORDER BY 1
	`, table), since).Scan(&stats).Error
	return stats, err
}

func (r *OverviewRepository) CountFeedType(feedType string) (int64, error) {
	var count int64
	err := r.db.Table("feed_activity").Where("type = ?", feedType).Count(&count).Error
	return count, err
}

func (r *OverviewRepository) DailyFeedCountsSince(feedType string, since any) ([]DailyStat, error) {
	var stats []DailyStat
	err := r.db.Raw(`
		SELECT date_trunc('day', created)::date::text AS date, COUNT(*) AS count
		FROM feed_activity WHERE type = ? AND created >= ? GROUP BY 1 ORDER BY 1
	`, feedType, since).Scan(&stats).Error
	return stats, err
}
