package utils

import (
	"encoding/json"
	"net/url"

	"github.com/gofiber/fiber/v3"
)

func IsSFW(c fiber.Ctx) bool {
	raw := c.Cookies("KUNGalgameSettings", "")
	if raw == "" {
		return true
	}

	decoded, err := url.QueryUnescape(raw)
	if err != nil {
		decoded = raw
	}

	var settings struct {
		ShowKUNGalgameContentLimit string `json:"showKUNGalgameContentLimit"`
	}
	if err := json.Unmarshal([]byte(decoded), &settings); err != nil {
		return true
	}

	return settings.ShowKUNGalgameContentLimit != "nsfw" &&
		settings.ShowKUNGalgameContentLimit != "all"
}
