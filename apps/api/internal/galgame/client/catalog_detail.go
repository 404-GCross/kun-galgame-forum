package client

import (
	"context"

	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/pkg/errors"
)

func CatalogDetailToFull(d *catWorkDetail, gid int) dto.NextMoeGalgameDetailFull {
	f := dto.NextMoeGalgameDetailFull{
		ID:                 gid,
		ContentLimit:       contentLimitOf(d.ClaimedBy, d.ContentRating),
		AgeLimit:           ageLimitFromRating(d.ContentRating),
		OriginalLanguage:   productLocale(d.OLang),
		ReleaseDate:        d.ReleaseDate,
		ReleaseDateTBA:     false,
		Updated:            d.Updated,
		Created:            d.Created,
		ResourceUpdateTime: zeroTS,
		Refs:               refsMap(d.Refs),
		Staff:              catalogStaffFromCredits(d.Credits),
		Characters:         catalogRosterToNextMoe(d.Characters),
		Covers:             catalogCoversToNextMoe(d),
		Screenshots:        catalogScreenshotsToNextMoe(d),
		Contributor:        []dto.NextMoeContributor{},
	}
	f.VndbID = f.Refs["vndb"]

	names, aliases := catalogTitles(d)
	f.NameJaJp, f.NameZhCn, f.NameZhTw, f.NameEnUs = names[0], names[1], names[2], names[3]
	if f.NameJaJp == "" && f.NameZhCn == "" && f.NameZhTw == "" && f.NameEnUs == "" {
		f.NameJaJp = d.DisplayName
	}
	f.Alias = aliases

	f.IntroJaJp, f.IntroZhCn, f.IntroZhTw, f.IntroEnUs = catalogIntros(d)

	if d.ClaimedBy != nil {
		f.Status = statusFromClaimState(d.ClaimedBy.State)
	}

	if hash, url, w, h, thumb := detailHero(d.CoverSlots, f.Covers); url != "" {
		f.EffectiveBannerHash = hash
		f.EffectiveBannerURL = url
		f.EffectiveBannerWidth = w
		f.EffectiveBannerHeight = h
		f.EffectiveBannerThumbhash = thumb
	}

	if hash, url, w, h, thumb := detailPortrait(d.CoverSlots, f.Covers); url != "" {
		f.EffectivePortraitHash = hash
		f.EffectivePortraitURL = url
		f.EffectivePortraitWidth = w
		f.EffectivePortraitHeight = h
		f.EffectivePortraitThumbhash = thumb
	}

	f.ExternalRatings = catalogExternalRatings(d.Ratings)

	labelAt := make(map[int64]int, len(d.Labels))
	for _, l := range d.Labels {
		if i, seen := labelAt[l.ID]; seen {
			f.Official[i].Official.Roles = appendUniqueStr(f.Official[i].Official.Roles, l.Kind)
			continue
		}
		labelAt[l.ID] = len(f.Official)
		f.Official = append(f.Official, dto.NextMoeOfficialRel{Official: dto.NextMoeOfficial{
			ID: int(l.ID), Name: l.DisplayName,
			Category:     l.LabelKind,
			Roles:        appendUniqueStr(nil, l.Kind),
			Lang:         l.Lang,
			Alias:        []dto.NextMoeAlias{},
			GalgameCount: l.WorkCount,
		}})
	}
	for _, sr := range d.Series {
		f.Series = append(f.Series, dto.NextMoeSeriesRef{ID: int(sr.ID), Name: sr.Name})
	}
	for _, e := range d.Engines {
		var ew dto.NextMoeEngineWithAlias
		ew.Engine.ID = int(e.ID)
		ew.Engine.Name = e.Name
		ew.Engine.Alias = []string{}
		ew.Engine.GalgameCount = e.WorkCount
		f.Engine = append(f.Engine, ew)
	}
	for _, t := range d.Tags {
		if t.CanonicalID == 0 {
			continue
		}
		if t.Tier == TagTierHidden {
			continue
		}
		f.Tag = append(f.Tag, dto.NextMoeTagWithSpoiler{
			SpoilerLevel: t.Spoiler,
			Tag: dto.NextMoeTag{
				ID: int(t.CanonicalID), Name: t.Name,
				Category:     catalogTagCategory(t.Kind, t.Sexual),
				GalgameCount: t.WorkCount,
			},
		})
	}
	return f
}

func detailHero(slots *catCoverSlots, covers []dto.NextMoeGalgameCover) (hash, url string, w, h int, thumb string) {
	if slots != nil {
		if s := slots.Banner; s != nil {
			return hashFromURL(s.URL), s.URL, s.Width, s.Height, s.Thumbhash
		}
		if s := slots.Portrait; s != nil {
			return hashFromURL(s.URL), s.URL, s.Width, s.Height, s.Thumbhash
		}
	} else if c := legacyLandscapeCover(covers); c != nil {
		return c.ImageHash, c.CDNURL, c.Width, c.Height, c.Thumbhash
	}
	if len(covers) == 0 {
		return "", "", 0, 0, ""
	}
	c := covers[0]
	return c.ImageHash, c.CDNURL, c.Width, c.Height, c.Thumbhash
}

// cover_slots.portrait is filled from any cover when no portrait-shaped one exists, so it is not
// guaranteed to be taller than wide; consumers must crop rather than trust the aspect ratio.
func detailPortrait(slots *catCoverSlots, covers []dto.NextMoeGalgameCover) (hash, url string, w, h int, thumb string) {
	if slots != nil {
		if s := slots.Portrait; s != nil {
			return hashFromURL(s.URL), s.URL, s.Width, s.Height, s.Thumbhash
		}
		return "", "", 0, 0, ""
	}
	if c := legacyPortraitCover(covers); c != nil {
		return c.ImageHash, c.CDNURL, c.Width, c.Height, c.Thumbhash
	}
	return "", "", 0, 0, ""
}

func legacyPortraitCover(covers []dto.NextMoeGalgameCover) *dto.NextMoeGalgameCover {
	for i := range covers {
		c := &covers[i]
		if c.Width > 0 && c.Height > 0 && c.Height*20 > c.Width*21 {
			return c
		}
	}
	return nil
}

func catalogExternalRatings(rows []catRating) []dto.GalgameExternalRating {
	out := make([]dto.GalgameExternalRating, 0, len(rows))
	for _, r := range rows {
		out = append(out, dto.GalgameExternalRating{
			Source: r.Source, Score: r.Score, VoteCount: r.VoteCount, Rank: r.Rank,
		})
	}
	return out
}

func legacyLandscapeCover(covers []dto.NextMoeGalgameCover) *dto.NextMoeGalgameCover {
	for i := range covers {
		c := &covers[i]
		if c.Width > 0 && c.Height > 0 && c.Height*20 <= c.Width*21 {
			return c
		}
	}
	return nil
}

func catalogTagCategory(kind string, sexual bool) string {
	if sexual {
		return "sexual"
	}
	return kind
}

func catalogTitles(d *catWorkDetail) (names [4]string, aliases []dto.NextMoeAlias) {
	aliases = []dto.NextMoeAlias{}
	idx := map[string]int{"ja-jp": 0, "zh-cn": 1, "zh-tw": 2, "en-us": 3}
	for _, t := range d.Titles {
		key := productLocale(t.Lang)
		i, isProductKey := idx[key]
		if isProductKey && t.Kind == "official" && names[i] == "" {
			names[i] = t.Title
			continue
		}
		aliases = append(aliases, dto.NextMoeAlias{Name: t.Title})
	}
	for _, t := range d.Titles {
		if i, ok := idx[productLocale(t.Lang)]; ok && names[i] == "" {
			names[i] = t.Title
		}
	}
	return names, aliases
}

func catalogIntros(d *catWorkDetail) (jaJP, zhCN, zhTW, enUS string) {
	slots := map[string]*string{"ja-jp": &jaJP, "zh-cn": &zhCN, "zh-tw": &zhTW, "en-us": &enUS}
	for _, in := range d.Intro {
		if slot, ok := slots[productLocale(in.Lang)]; ok && *slot == "" {
			*slot = in.Intro
		}
	}
	return jaJP, zhCN, zhTW, enUS
}

func catalogRosterToNextMoe(chars []catWorkCharacter) []dto.NextMoeGalgameCharacter {
	out := make([]dto.NextMoeGalgameCharacter, 0, len(chars))
	for _, c := range chars {
		if c.Name == "" && c.Latin == "" {
			continue
		}
		voices := make([]dto.NextMoeCharacterVoice, 0, len(c.Voices))
		for _, v := range c.Voices {
			voices = append(voices, dto.NextMoeCharacterVoice{ID: int(v.ID), Name: v.Name})
		}
		out = append(out, dto.NextMoeGalgameCharacter{
			ID: int(c.ID), Name: c.Name, Latin: c.Latin,
			Kind: c.Kind, Spoiler: c.Spoiler,
			Image: c.Image, Figure: c.Figure,
			ImageMeta: ArtMetaDTO(c.ImageMeta), FigureMeta: ArtMetaDTO(c.FigureMeta),
			Voices: voices,
		})
	}
	return out
}

func catalogCoversToNextMoe(d *catWorkDetail) []dto.NextMoeGalgameCover {
	out := make([]dto.NextMoeGalgameCover, 0, len(d.Covers))
	for i, c := range d.Covers {
		out = append(out, dto.NextMoeGalgameCover{
			ImageHash: hashFromURL(c.URL), SortOrder: i,
			Sexual: c.Sexual, Violence: c.Violence, Kind: c.Kind,
			CDNURL: c.URL, Width: c.Width, Height: c.Height, Thumbhash: c.Thumbhash,
		})
	}
	return out
}

func catalogScreenshotsToNextMoe(d *catWorkDetail) []dto.NextMoeGalgameScreenshot {
	out := make([]dto.NextMoeGalgameScreenshot, 0, len(d.Screenshots))
	for i, s := range d.Screenshots {
		out = append(out, dto.NextMoeGalgameScreenshot{
			ImageHash: hashFromURL(s.URL), SortOrder: i, Caption: s.Caption,
			Sexual: s.Sexual, Violence: s.Violence,
			CDNURL: s.URL, Width: s.Width, Height: s.Height, Thumbhash: s.Thumbhash,
		})
	}
	return out
}

type GalgameLink struct {
	Name   string `json:"name"`
	Link   string `json:"link"`
	Source string `json:"source"`
}

func (c *GalgameClient) CatalogWorkLinks(ctx context.Context, gid int) ([]GalgameLink, *errors.AppError) {
	d, found, appErr := c.CatalogWorkDetail(ctx, gid)
	if appErr != nil {
		return nil, appErr
	}
	out := []GalgameLink{}
	if !found {
		return out, nil
	}
	for _, l := range d.Links {
		out = append(out, GalgameLink{Name: linkDisplayName(l), Link: l.URL, Source: l.Source})
	}
	return out, nil
}

func linkDisplayName(l catWorkLink) string {
	return LinkDisplayName(l.Source, l.URL)
}

func appendUniqueStr(slice []string, val string) []string {
	if val == "" {
		return slice
	}
	for _, s := range slice {
		if s == val {
			return slice
		}
	}
	return append(slice, val)
}
