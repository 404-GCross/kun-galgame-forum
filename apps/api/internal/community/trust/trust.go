package trust

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/role"
)

const veteranAge = 30 * 24 * time.Hour

const boostOnceTTL = 90 * 24 * time.Hour

type Reporter struct {
	cli *communityclient.Client
	rdb *redis.Client
	db  *gorm.DB
}

func New(cli *communityclient.Client, rdb *redis.Client, db *gorm.DB) *Reporter {
	return &Reporter{cli: cli, rdb: rdb, db: db}
}

func (r *Reporter) active() bool {
	return r != nil && r.cli != nil && r.cli.Configured()
}

func (r *Reporter) Boost(userID int, roles []string) {
	if !r.active() || userID <= 0 {
		return
	}
	go r.declare(userID, roles)
}

func (r *Reporter) declare(userID int, roles []string) {
	ctx := context.Background()
	boost := boostForRoles(roles)
	if boost == communityclient.BoostNone && r.isVeteran(ctx, userID) {
		boost = communityclient.BoostVeteran
	}
	if boost == communityclient.BoostNone {
		return
	}

	if r.rdb != nil {
		set, err := r.rdb.SetNX(ctx, "kungal:community:boosted:"+strconv.Itoa(userID), boost, boostOnceTTL).Result()
		if err == nil && !set {
			return
		}
	}
	if _, err := r.cli.Boost(ctx, communityclient.SetBoostRequest{UserID: int64(userID), Boost: boost}); err != nil {
		slog.Warn("community trust boost declare failed", "user_id", userID, "boost", boost, "error", err)
	}
}

func boostForRoles(roles []string) int32 {
	if role.CanModerate(roles) {
		return communityclient.BoostStaff
	}
	return communityclient.BoostNone
}

func isVeteranAge(created, now time.Time) bool {
	return !created.IsZero() && now.Sub(created) >= veteranAge
}

func (r *Reporter) isVeteran(ctx context.Context, userID int) bool {
	if r.db == nil {
		return false
	}
	var created time.Time
	err := r.db.WithContext(ctx).
		Raw("SELECT created FROM kungal_user_state WHERE user_id = ?", userID).
		Scan(&created).Error
	if err != nil {
		return false
	}
	return isVeteranAge(created, time.Now())
}
