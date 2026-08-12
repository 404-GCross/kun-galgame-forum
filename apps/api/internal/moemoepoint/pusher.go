// Package moemoepoint is kungal's single emitter for moemoepoint changes.
//
// The balance is single-sourced in OAuth (contract C3); the local
// kungal_user_state.moemoepoint is a read-cache. NEVER do a local `+=` — only
// ever write the value OAuth returns. A local increment double-counts after the
// cross-site merge migration.
//
// Awards are best-effort and non-blocking: a failed push only logs, and the
// stable idempotency key makes a retry or a cmd/sync-moemoepoint re-seed safe.
package moemoepoint

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"kun-galgame-api/pkg/userclient"

	"gorm.io/gorm"
)

const (
	ReasonDailyCheckin    = "daily_checkin"
	ReasonLiked           = "liked"
	ReasonContentApproved = "content_approved"
	ReasonContentRemoved  = "content_removed"
)

type adjuster interface {
	AdjustMoemoepoint(ctx context.Context, userID, delta int, reason, ref, idempotencyKey string) (userclient.MoemoepointResult, error)
}

type Awarder struct {
	client  adjuster
	db      *gorm.DB
	timeout time.Duration
}

func NewAwarder(client adjuster, db *gorm.DB) *Awarder {
	return &Awarder{client: client, db: db, timeout: 5 * time.Second}
}

func (a *Awarder) Award(userID, delta int, reason, ref, idempotencyKey string) {
	if a == nil || a.client == nil || userID <= 0 || delta == 0 {
		return
	}
	if delta > maxDelta {
		delta = maxDelta
	} else if delta < -maxDelta {
		delta = -maxDelta
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), a.timeout)
		defer cancel()
		res, err := a.client.AdjustMoemoepoint(ctx, userID, delta, reason, ref, idempotencyKey)
		if err != nil {
			slog.Warn("moemoepoint award failed (best-effort, skipped)",
				"user_id", userID, "delta", delta, "reason", reason, "ref", ref, "err", err)
			return
		}
		if a.db != nil {
			if err := a.db.WithContext(ctx).
				Exec(`UPDATE kungal_user_state SET moemoepoint = ? WHERE user_id = ?`, res.Balance, userID).Error; err != nil {
				slog.Warn("moemoepoint cache mirror failed",
					"user_id", userID, "balance", res.Balance, "err", err)
			}
		}
	}()
}

func (a *Awarder) AwardSync(userID, delta int, reason, ref, idempotencyKey string) error {
	if a == nil || a.client == nil || userID <= 0 || delta == 0 {
		return nil
	}
	if delta > maxDelta {
		delta = maxDelta
	} else if delta < -maxDelta {
		delta = -maxDelta
	}
	ctx, cancel := context.WithTimeout(context.Background(), a.timeout)
	defer cancel()
	res, err := a.client.AdjustMoemoepoint(ctx, userID, delta, reason, ref, idempotencyKey)
	if err != nil {
		return err
	}
	if a.db != nil {
		if err := a.db.WithContext(ctx).
			Exec(`UPDATE kungal_user_state SET moemoepoint = ? WHERE user_id = ?`, res.Balance, userID).Error; err != nil {
			slog.Warn("moemoepoint cache mirror failed",
				"user_id", userID, "balance", res.Balance, "err", err)
		}
	}
	return nil
}

func IsPermanentAwardError(err error) bool {
	if err == nil {
		return false
	}
	var oerr *userclient.OAuthError
	return errors.As(err, &oerr)
}

const maxDelta = 1_000_000

var defaultAwarder *Awarder

func SetDefault(a *Awarder) { defaultAwarder = a }

func Award(userID, delta int, reason, ref, idempotencyKey string) {
	defaultAwarder.Award(userID, delta, reason, ref, idempotencyKey)
}

func AwardSync(userID, delta int, reason, ref, idempotencyKey string) error {
	return defaultAwarder.AwardSync(userID, delta, reason, ref, idempotencyKey)
}

func Key(parts ...string) string {
	return "kungal:" + strings.Join(parts, ":")
}

func KeyNonce(parts ...string) string {
	return Key(append(parts, strconv.FormatInt(time.Now().UnixNano(), 36))...)
}

func Ref(kind string, id int) string {
	return kind + ":" + strconv.Itoa(id)
}
