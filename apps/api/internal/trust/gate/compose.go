package gate

import (
	"strings"

	"kun-galgame-api/pkg/errors"
)

func ErrContentBlocked() *errors.AppError {
	return errors.New(errors.CodeBiz, "内容包含违禁词，无法发布", 422)
}

func ComposeText(parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, "\n")
}
