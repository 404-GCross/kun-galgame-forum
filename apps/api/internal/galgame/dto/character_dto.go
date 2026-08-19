package dto

type GalgameCharacterIntro struct {
	Lang    string `json:"lang"`
	Intro   string `json:"intro"`
	Source  string `json:"source"`
	Machine bool   `json:"machine"`
}

type GalgameCharacterTrait struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Group   string `json:"group"`
	Spoiler int    `json:"spoiler"`
	Lie     bool   `json:"lie"`
}

type GalgameCharacterLink struct {
	Source string `json:"source"`
	Name   string `json:"name"`
	URL    string `json:"url,omitempty"`
}

type GalgameCharacterWork struct {
	GalgameCard
	CatalogID int                           `json:"catalog_id"`
	Voices    []GalgameDetailCharacterVoice `json:"voices"`
}

type GalgameCharacterDetail struct {
	ID           int                     `json:"id"`
	Name         string                  `json:"name"`
	NameOriginal string                  `json:"name_original,omitempty"`
	Latin        string                  `json:"latin,omitempty"`
	Image        string                  `json:"image"`
	Figure       string                  `json:"figure"`
	ImageMeta    *GalgameArtMeta         `json:"image_meta,omitempty"`
	FigureMeta   *GalgameArtMeta         `json:"figure_meta,omitempty"`
	Intro        string                  `json:"intro"`
	Intros       []GalgameCharacterIntro `json:"intros"`
	Traits       []GalgameCharacterTrait `json:"traits"`
	Links        []GalgameCharacterLink  `json:"links"`
	Works        []GalgameCharacterWork  `json:"works"`
	NextOffset   *int                    `json:"next_offset"`
	MovedTo      int                     `json:"moved_to,omitempty"`
}
