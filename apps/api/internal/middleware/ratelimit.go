package middleware

import (
	"fmt"
	"time"

	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
)

var rateLimitScript = redis.NewScript(`
local count = redis.call('INCR', KEYS[1])
if redis.call('PTTL', KEYS[1]) < 0 then
	redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
return count
`)

func RateLimit(rdb *redis.Client, prefix string, maxRequests int, window time.Duration) fiber.Handler {
	return func(c fiber.Ctx) error {
		user := GetUser(c)
		if user == nil {
			return response.Error(c, errors.ErrAuthExpired())
		}

		key := fmt.Sprintf("ratelimit:%s:%d", prefix, user.ID)
		count, err := rateLimitScript.Run(
			c.Context(), rdb, []string{key}, window.Milliseconds(),
		).Int64()
		if err != nil {
			return c.Next()
		}

		if count > int64(maxRequests) {
			return response.Error(c, errors.ErrBadRequest("操作过于频繁，请稍后再试"))
		}

		return c.Next()
	}
}
