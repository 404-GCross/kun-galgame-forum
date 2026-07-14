package middleware

import (
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"

	"github.com/gofiber/fiber/v3"
)

// GalgameCommentReadonly gates the OLD galgame comment WRITE routes during the
// migration cutover freeze window (step 05, KUN_GALGAME_COMMENT_READONLY). When
// enabled it answers 503 with a clear notice BEFORE the old handler runs; when
// disabled it is a pure passthrough (c.Next()), so the flag-off path is
// byte-identical to today. Reads are never mounted behind it. This is a thin
// pre-check — it does not touch the old handler body.
func GalgameCommentReadonly(enabled bool) fiber.Handler {
	return func(c fiber.Ctx) error {
		if !enabled {
			return c.Next()
		}
		return response.Error(c, errors.New(
			errors.CodeBiz,
			"评论区正在迁移维护，暂时只读，请稍后再试",
			fiber.StatusServiceUnavailable,
		))
	}
}
