package dto

// The 角色 detail page — the character herself, rather than the roster line the
// game page shows.
//
// What this face can say is bounded by what the catalog's PUBLIC lane
// publishes. The physical-attribute block (性别 / 生日 / 三围 / 血型 …) exists in
// the registry but only on the staff-side face, so its absence here is a
// deliberate boundary rather than a gap to fill in later from somewhere else.

// GalgameCharacterIntro is one language's description of the character.
// Several may arrive; the page renders one and offers the rest.
type GalgameCharacterIntro struct {
	Lang  string `json:"lang"`
	Intro string `json:"intro"`
	// Machine marks an LLM translation, surfaced only for a language no source
	// wrote. It is labelled in the UI: a machine-translated bio is worth
	// reading and worth knowing about.
	Machine bool `json:"machine"`
}

// GalgameCharacterTrait is one VNDB trait.
type GalgameCharacterTrait struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	// Group is the trait's root group ("Hair", "Personality" …), "" when the
	// trait IS a root. The page groups by it.
	Group string `json:"group"`
	// Spoiler: 0 = none, 1 = minor, 2 = major. Everything above 0 arrives but
	// stays hidden until the reader explicitly asks — the full set travels in
	// one response so the reveal costs no round trip.
	Spoiler int `json:"spoiler"`
	// Lie is VNDB's "appears true, is actually false" marker, kept as-is: it
	// is a fact about the character's presentation, not a data defect.
	Lie bool `json:"lie"`
}

// GalgameCharacterLink is one external page for the character.
type GalgameCharacterLink struct {
	Source string `json:"source"`
	Name   string `json:"name"`
	// URL is empty for a source kungal has no verified character-page template
	// for; the row then renders as text rather than as a guess that 404s.
	URL string `json:"url,omitempty"`
}

// GalgameCharacterWork is one game the character appears in: the SHARED galgame
// card plus who voiced her THERE.
//
// Per-work voices rather than one CV for the character, because a recast is a
// real event — a heroine can be one seiyuu in the original and another in the
// remake, and folding them into a single line would erase which was which.
type GalgameCharacterWork struct {
	GalgameCard
	// CatalogID is the registry id, and the only stable key on this list: every
	// work the forum has not ingested shares gid 0.
	CatalogID int                           `json:"catalog_id"`
	Voices    []GalgameDetailCharacterVoice `json:"voices"`
}

// GalgameCharacterDetail is the 角色 page.
type GalgameCharacterDetail struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	NameJa string `json:"name_ja,omitempty"`
	NameZh string `json:"name_zh,omitempty"`
	Latin  string `json:"latin,omitempty"`
	// Image is the BUST and Figure the FULL-BODY 立绘, both ready-made absolute
	// CDN URLs at original size (the FE derives the variant it needs), "" for
	// none. Two fields because they are two assets: cropping a figure into a
	// portrait box leaves a picture of someone's midriff.
	Image  string `json:"image"`
	Figure string `json:"figure"`
	// Each artwork's own shape, resolved from image_service (the catalog
	// publishes none for entity art). Absent = unknown; see GalgameArtMeta.
	ImageMeta  *GalgameArtMeta `json:"image_meta,omitempty"`
	FigureMeta *GalgameArtMeta `json:"figure_meta,omitempty"`
	// Intro is the one description the page leads with (Chinese first — a bio
	// is read, not identified by); Intros is every language, for the reader who
	// wants the original.
	Intro  string                  `json:"intro"`
	Intros []GalgameCharacterIntro `json:"intros"`
	// Traits is the full set INCLUDING spoilers; each row carries its own
	// level and the frontend withholds the flagged ones behind a click.
	// Sexual-family traits are already dropped here for a SFW reader — that
	// decision is the server's, not the browser's.
	Traits []GalgameCharacterTrait `json:"traits"`
	Links  []GalgameCharacterLink  `json:"links"`
	Works  []GalgameCharacterWork  `json:"works"`
	// NextOffset is null on the last page — the only end-of-list signal the
	// catalog gives, and the reason this page counts nothing.
	NextOffset *int `json:"next_offset"`
	// MovedTo is the ONLY field set when this character id was merged away:
	// the page 301s to the survivor instead of rendering anything under the
	// dead id.
	MovedTo int `json:"moved_to,omitempty"`
}
