package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"kun-galgame-api/pkg/errors"
)

const catalogNameCreditsCap = 50

type catLocalizedName struct {
	Value string `json:"value"`
	Kind  string `json:"kind"`
}

var (
	catalogZhLocales = []string{"zh-Hans", "zh", "zh-Hant"}
	catalogJaLocales = []string{"ja"}
)

type CatalogName struct {
	ID          int64                       `json:"id"`
	DisplayName string                      `json:"display_name"`
	Lang        string                      `json:"lang"`
	Localized   map[string]catLocalizedName `json:"localized"`
	Latin       string                      `json:"latin"`
	PersonID    int64                       `json:"person_id"`

	PhotoHash string `json:"photo_hash"`
	Gender    *int   `json:"gender"`
	BirthY    *int   `json:"birth_y"`
	BirthM    *int   `json:"birth_m"`
	BirthD    *int   `json:"birth_d"`
	Siblings  []struct {
		ID          int64  `json:"id"`
		DisplayName string `json:"display_name"`
		Lang        string `json:"lang"`
		Latin       string `json:"latin"`
	} `json:"siblings"`
	Intros []struct {
		Lang   string `json:"lang"`
		Intro  string `json:"intro"`
		Source string `json:"source"`
	} `json:"intros"`
	Refs    []catRef         `json:"refs"`
	Links   []catRelatedLink `json:"links"`
	Credits []struct {
		Work struct {
			ID            int64         `json:"id"`
			DisplayName   string        `json:"display_name"`
			Medium        string        `json:"medium"`
			ContentRating string        `json:"content_rating"`
			ClaimedBy     *catClaimedBy `json:"claimed_by"`
		} `json:"work"`
		Roles []struct {
			RoleKey     string `json:"role_key"`
			RoleName    string `json:"role_name"`
			CharacterID int64  `json:"character_id"`
			Character   string `json:"character"`
		} `json:"roles"`
	} `json:"credits"`
	NextOffset *int `json:"next_offset"`
}

func (c *GalgameClient) CatalogNameDetail(
	ctx context.Context, id int64, limit, offset int,
) (*CatalogName, bool, int64, *errors.AppError) {
	if limit <= 0 || limit > catalogNameCreditsCap {
		limit = catalogNameCreditsCap
	}
	q := url.Values{
		"include": {"credits"},
		"limit":   {strconv.Itoa(limit)},
		"offset":  {strconv.Itoa(max(offset, 0))},
	}
	openPopulation(q)

	status, env, appErr := c.getV1Envelope(ctx, "/catalog/names/"+strconv.FormatInt(id, 10), q)
	if appErr != nil {
		return nil, false, 0, appErr
	}
	switch {
	case status == http.StatusNotFound:
		return nil, false, 0, nil
	case status == http.StatusMovedPermanently && env.Code == catalogMovedCode:
		var moved struct {
			CurrentID int64 `json:"current_id"`
		}
		if err := json.Unmarshal(env.Data, &moved); err != nil || moved.CurrentID == 0 {
			return nil, false, 0, nil
		}
		return nil, false, moved.CurrentID, nil
	case env.Code != 0:
		return nil, false, 0, errors.New(env.Code, env.Message, status)
	}
	var n CatalogName
	if err := json.Unmarshal(env.Data, &n); err != nil {
		return nil, false, 0, errors.ErrInternal("解析 Catalog 名义详情响应失败")
	}
	return &n, true, 0, nil
}

func (c *GalgameClient) CatalogRowsByCatalogIDs(
	ctx context.Context, ids []int64, isSFW bool,
) (map[int64]CatalogWorkListItem, *errors.AppError) {
	rows, appErr := c.worksByCatalogIDs(ctx, ids, catalogBriefInclude, contentLimitFor(isSFW))
	if appErr != nil {
		return nil, appErr
	}
	out := make(map[int64]CatalogWorkListItem, len(rows))
	for _, r := range rows {
		out[r.ID] = r
	}
	return out, nil
}

func PickCatalogName(displayName, latin string) string {
	if displayName != "" {
		return displayName
	}
	return latin
}

func CatalogNameByScript(localized map[string]catLocalizedName, displayName, lang string) (ja, zh string) {
	switch {
	case strings.HasPrefix(lang, "ja"):
		ja = displayName
	case strings.HasPrefix(lang, "zh"):
		zh = displayName
	}
	if localizedForm := pickLocalized(localized, catalogJaLocales); localizedForm != "" {
		ja = localizedForm
	}
	if localizedForm := pickLocalized(localized, catalogZhLocales); localizedForm != "" {
		zh = localizedForm
	}
	return ja, zh
}

func pickLocalized(localized map[string]catLocalizedName, locales []string) string {
	for _, locale := range locales {
		if entry, ok := localized[locale]; ok && entry.Value != "" {
			return entry.Value
		}
	}
	return ""
}
