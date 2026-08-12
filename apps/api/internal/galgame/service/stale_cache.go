package service

import (
	"context"
	"sync"
	"time"

	"kun-galgame-api/pkg/errors"
)

type staleCache[T any] struct {
	mu        sync.Mutex
	rows      []T
	ok        bool
	built     time.Time
	buildingC chan struct{}
}

// A failed rebuild keeps the previous rows AND the previous timestamp, so the
// next caller retries instead of caching an outage for a whole ttl.
func (c *staleCache[T]) get(
	ctx context.Context,
	ttl time.Duration,
	build func(context.Context) ([]T, *errors.AppError),
) ([]T, *errors.AppError) {
	c.mu.Lock()
	rows, ok, fresh, building := c.rows, c.ok, time.Since(c.built) < ttl, c.buildingC
	if ok && fresh {
		c.mu.Unlock()
		return rows, nil
	}
	if building != nil {
		c.mu.Unlock()
		if ok {
			return rows, nil
		}
		<-building
		c.mu.Lock()
		rows, ok = c.rows, c.ok
		c.mu.Unlock()
		if !ok {
			return nil, errors.ErrInternal("获取列表失败")
		}
		return rows, nil
	}
	done := make(chan struct{})
	c.buildingC = done
	c.mu.Unlock()

	if ok {
		go func() {
			built, appErr := build(context.WithoutCancel(ctx))
			c.finish(built, appErr, done)
		}()
		return rows, nil
	}

	built, appErr := build(ctx)
	c.finish(built, appErr, done)
	if appErr != nil {
		return nil, appErr
	}
	return built, nil
}

func (c *staleCache[T]) finish(rows []T, appErr *errors.AppError, done chan struct{}) {
	c.mu.Lock()
	if appErr == nil {
		c.rows, c.ok, c.built = rows, true, time.Now()
	}
	c.buildingC = nil
	c.mu.Unlock()
	close(done)
}
