package middleware

import (
	"kun-galgame-api/pkg/namepref"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v3"
)

// NamePreference puts the reader's 中文名 / 原名 choice on the request context,
// which every handler already hands to its service as c.Context().
func NamePreference(c fiber.Ctx) error {
	if utils.PrefersOriginalName(c) {
		c.SetContext(namepref.With(c.Context(), true))
	}
	return c.Next()
}
