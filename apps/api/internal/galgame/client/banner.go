package client

import (
	"bytes"
	"encoding/json"
	"strings"
)

func bannerURLFromHash(cdnBase, hash string) string {
	if len(hash) < 4 {
		return ""
	}
	return strings.TrimRight(cdnBase, "/") + "/" +
		hash[:2] + "/" + hash[2:4] + "/" + hash + ".webp"
}

func (c *GalgameClient) ImageURLFromHash(hash string) string {
	if c.imageCDNBase == "" {
		return ""
	}
	return bannerURLFromHash(c.imageCDNBase, hash)
}

func rewriteBanners(raw json.RawMessage, cdnBase string) json.RawMessage {
	if cdnBase == "" || len(raw) == 0 {
		return raw
	}

	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var tree any
	if err := dec.Decode(&tree); err != nil {
		return raw
	}

	if !walkResolveBanner(tree, cdnBase) {
		return raw
	}

	out, err := json.Marshal(tree)
	if err != nil {
		return raw
	}
	return out
}

func walkResolveBanner(node any, cdnBase string) bool {
	changed := false
	switch v := node.(type) {
	case map[string]any:
		if hashRaw, ok := v["effective_banner_hash"]; ok {
			hash, _ := hashRaw.(string)
			existing, _ := v["effective_banner_url"].(string)
			if strings.TrimSpace(hash) != "" && strings.TrimSpace(existing) == "" {
				if url := bannerURLFromHash(cdnBase, hash); url != "" {
					v["effective_banner_url"] = url
					changed = true
				}
			}
		}
		if hashRaw, ok := v["image_hash"]; ok {
			if _, isRow := v["sort_order"]; isRow {
				hash, _ := hashRaw.(string)
				existing, _ := v["cdn_url"].(string)
				if strings.TrimSpace(hash) != "" && strings.TrimSpace(existing) == "" {
					if url := bannerURLFromHash(cdnBase, hash); url != "" {
						v["cdn_url"] = url
						changed = true
					}
				}
			}
		}
		for _, child := range v {
			if walkResolveBanner(child, cdnBase) {
				changed = true
			}
		}
	case []any:
		for _, child := range v {
			if walkResolveBanner(child, cdnBase) {
				changed = true
			}
		}
	}
	return changed
}
