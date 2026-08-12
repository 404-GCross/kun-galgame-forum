package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"

	"kun-galgame-api/pkg/errors"
)

const catalogCharacterWorksCap = 50

const catalogCharacterSpoilerCeiling = 2

type CatalogCharacter struct {
	ID          int64                       `json:"id"`
	DisplayName string                      `json:"display_name"`
	Lang        string                      `json:"lang"`
	Localized   map[string]catLocalizedName `json:"localized"`
	Latin       string                      `json:"latin"`
	Image       string                      `json:"image"`
	Figure      string                      `json:"figure"`
	ImageMeta   ArtMeta                     `json:"-"`
	FigureMeta  ArtMeta                     `json:"-"`
	Intros      []struct {
		Lang    string `json:"lang"`
		Intro   string `json:"intro"`
		Source  string `json:"source"`
		Machine bool   `json:"machine"`
	} `json:"intros"`
	Traits []CatalogCharacterTrait `json:"traits"`
	Refs   []catRef                `json:"refs"`
	Works  []struct {
		Work struct {
			ID            int64         `json:"id"`
			DisplayName   string        `json:"display_name"`
			Medium        string        `json:"medium"`
			ContentRating string        `json:"content_rating"`
			ClaimedBy     *catClaimedBy `json:"claimed_by"`
		} `json:"work"`
		Voices []struct {
			ID    int64  `json:"id"`
			Name  string `json:"name"`
			Lang  string `json:"lang"`
			Latin string `json:"latin"`
		} `json:"voices"`
	} `json:"works"`
	NextOffset *int `json:"next_offset"`
}

type CatalogCharacterTrait struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	NameZh  string `json:"name_zh"`
	Group   string `json:"group"`
	GroupZh string `json:"group_zh"`
	Spoiler int    `json:"spoiler"`
	Sexual  bool   `json:"sexual"`
	Lie     bool   `json:"lie"`
}

func (c *GalgameClient) CatalogCharacterDetail(
	ctx context.Context, id int64, limit, offset int, withWorks bool,
) (*CatalogCharacter, bool, int64, *errors.AppError) {
	if limit <= 0 || limit > catalogCharacterWorksCap {
		limit = catalogCharacterWorksCap
	}
	q := url.Values{
		"spoilers": {strconv.Itoa(catalogCharacterSpoilerCeiling)},
		"limit":    {strconv.Itoa(limit)},
		"offset":   {strconv.Itoa(max(offset, 0))},
	}
	if withWorks {
		q.Set("include", "works")
	}
	openPopulation(q)

	status, env, appErr := c.getV1Envelope(ctx, "/catalog/characters/"+strconv.FormatInt(id, 10), q)
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
	var ch CatalogCharacter
	if err := json.Unmarshal(env.Data, &ch); err != nil {
		return nil, false, 0, errors.ErrInternal("解析 Catalog 角色详情响应失败")
	}
	if meta := c.resolveArtMeta([]string{ch.Image, ch.Figure}); meta != nil {
		ch.ImageMeta, ch.FigureMeta = meta[ch.Image], meta[ch.Figure]
	}
	return &ch, true, 0, nil
}
