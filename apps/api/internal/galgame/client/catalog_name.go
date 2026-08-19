package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"

	"kun-galgame-api/pkg/errors"
)

const catalogNameCreditsCap = 50

// catLocalizedName is one locale slot of the public face's name primitive.
// Machine marks a machine-translated fill-in: since wave 209 such a row may
// occupy a locale that has no source-provenance name, so a rendered name can
// itself be machine text.
type catLocalizedName struct {
	Value   string `json:"value"`
	Kind    string `json:"kind"`
	Machine bool   `json:"machine"`
}

// CatalogAlias is one row of an entity's alias list. Wave 209 turned these from
// bare strings into rows that carry their own language and provenance.
type CatalogAlias struct {
	Value   string `json:"value"`
	Lang    string `json:"lang"`
	Kind    string `json:"kind"`
	Machine bool   `json:"machine"`
}

// CatalogIntro is the single intro shape the work, character, name, label and
// tag faces have all shared since wave 209. Tag intros never set Machine.
type CatalogIntro struct {
	Lang    string `json:"lang"`
	Intro   string `json:"intro"`
	Source  string `json:"source"`
	Machine bool   `json:"machine"`
}

// CatalogPerson is the name projection the public face uses wherever a person
// is named inside another record: roster voices, credit rows, siblings.
type CatalogPerson struct {
	ID          int64                       `json:"id"`
	DisplayName string                      `json:"display_name"`
	Lang        string                      `json:"lang"`
	Latin       string                      `json:"latin"`
	Localized   map[string]catLocalizedName `json:"localized"`
}

func (p *CatalogPerson) Name() string {
	return CatalogEntityName(p.Localized, p.DisplayName, p.Latin)
}

var catalogZhLocales = []string{"zh-Hans", "zh", "zh-Hant"}

type CatalogName struct {
	ID          int64                       `json:"id"`
	DisplayName string                      `json:"display_name"`
	Lang        string                      `json:"lang"`
	Localized   map[string]catLocalizedName `json:"localized"`
	Latin       string                      `json:"latin"`
	PersonID    int64                       `json:"person_id"`

	PhotoHash string           `json:"photo_hash"`
	Gender    *int             `json:"gender"`
	BirthY    *int             `json:"birth_y"`
	BirthM    *int             `json:"birth_m"`
	BirthD    *int             `json:"birth_d"`
	Siblings  []CatalogPerson  `json:"siblings"`
	Intros    []CatalogIntro   `json:"intros"`
	Refs      []catRef         `json:"refs"`
	Links     []catRelatedLink `json:"links"`
	Credits   []struct {
		Work  catWorkBrief `json:"work"`
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

// CatalogEntityName renders an entity name for this site's Chinese readers:
// localized["zh-Hans"] ?? display_name ?? latin. Wave 209 made that chain
// terminal across every projection of the public face — detail records, roster
// rows, voices, credits, via_label and search hits all carry the same three
// fields — so there is exactly one render path and no projection needs its own.
func CatalogEntityName(localized map[string]catLocalizedName, displayName, latin string) string {
	if zh := pickLocalized(localized, catalogZhLocales); zh != "" {
		return zh
	}
	return PickCatalogName(displayName, latin)
}

// CatalogEntityNames adds the record's own name for a secondary line, empty
// when that is already what CatalogEntityName rendered.
func CatalogEntityNames(localized map[string]catLocalizedName, displayName, latin string) (name, original string) {
	name = CatalogEntityName(localized, displayName, latin)
	if displayName != "" && displayName != name {
		original = displayName
	}
	return name, original
}

func pickLocalized(localized map[string]catLocalizedName, locales []string) string {
	for _, locale := range locales {
		if entry, ok := localized[locale]; ok && entry.Value != "" {
			return entry.Value
		}
	}
	return ""
}
