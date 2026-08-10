package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
)

type ImageTokens []string

func (t ImageTokens) Value() (driver.Value, error) {
	if len(t) == 0 {
		return "", nil
	}
	b, err := json.Marshal([]string(t))
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (t *ImageTokens) Scan(src any) error {
	var s string
	switch v := src.(type) {
	case nil:
		*t = nil
		return nil
	case string:
		s = v
	case []byte:
		s = string(v)
	default:
		return fmt.Errorf("ImageTokens: unsupported Scan type %T", src)
	}
	if strings.TrimSpace(s) == "" {
		*t = nil
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return err
	}
	*t = out
	return nil
}
