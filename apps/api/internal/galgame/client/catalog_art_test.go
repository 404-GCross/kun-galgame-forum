package client

import (
	"testing"

	"kun-galgame-api/pkg/imageclient"
)

// The whole point of the shape lookup is telling "unknown" apart from a number.
// A zero-sized meta published as 0×0 would reach the browser as a real ratio
// and collapse the frame; it has to be absent so the renderer falls back.
func TestArtMetaDTO_UnknownIsAbsentNotZero(t *testing.T) {
	if got := ArtMetaDTO(ArtMeta{}); got != nil {
		t.Errorf("ArtMetaDTO(zero) = %+v, want nil", got)
	}
	if got := ArtMetaDTO(ArtMeta{Width: 0, Height: 800}); got != nil {
		t.Errorf("ArtMetaDTO(half-known) = %+v, want nil", got)
	}
	got := ArtMetaDTO(ArtMeta{Width: 600, Height: 800, Thumbhash: "abc"})
	if got == nil || got.Width != 600 || got.Height != 800 || got.Thumbhash != "abc" {
		t.Errorf("ArtMetaDTO(known) = %+v", got)
	}
}

// The roster is sized in ONE batch for the whole cast, keyed back by URL, and a
// character whose hash image_service has never seen keeps an empty shape rather
// than borrowing someone else's.
func TestHydrateRosterArt_OneBatchKeyedByURL(t *testing.T) {
	const bust = "https://cdn.test/aa/bb/aabbhash1.webp"
	const figure = "https://cdn.test/cc/dd/ccddhash2.webp"
	const unknown = "https://cdn.test/ee/ff/eeffhash3.webp"

	calls := 0
	c := New("https://catalog.test", "nm_test_key", "")
	c.SetImageMetaResolver(func(hashes []string) map[string]imageclient.ImageMeta {
		calls++
		return map[string]imageclient.ImageMeta{
			"aabbhash1": {Width: 256, Height: 360},
			"ccddhash2": {Width: 700, Height: 900, Thumbhash: "hash"},
		}
	})

	chars := []catWorkCharacter{
		{ID: 1, Name: "A", Image: bust, Figure: figure},
		{ID: 2, Name: "B", Figure: unknown},
		{ID: 3, Name: "C"},
	}
	c.hydrateRosterArt(chars)

	if calls != 1 {
		t.Errorf("resolver called %d times, want exactly 1 for the whole cast", calls)
	}
	if chars[0].ImageMeta.Width != 256 || chars[0].FigureMeta.Height != 900 {
		t.Errorf("character 1 = %+v / %+v", chars[0].ImageMeta, chars[0].FigureMeta)
	}
	if chars[0].FigureMeta.Thumbhash != "hash" {
		t.Errorf("thumbhash = %q, want it carried for the blur-up", chars[0].FigureMeta.Thumbhash)
	}
	// An unanswered hash and no picture at all both stay zero — and both publish
	// as nothing, so neither can be mistaken for a measured shape.
	if chars[1].FigureMeta != (ArtMeta{}) || chars[2].ImageMeta != (ArtMeta{}) {
		t.Errorf("unresolved art = %+v / %+v, want zero", chars[1].FigureMeta, chars[2].ImageMeta)
	}
}

// No resolver wired (image_service unconfigured, and every other test) must be
// a no-op rather than a panic: dimensions are an enhancement, not a dependency.
func TestHydrateRosterArt_NoResolverIsANoOp(t *testing.T) {
	c := New("https://catalog.test", "nm_test_key", "")
	chars := []catWorkCharacter{{ID: 1, Image: "https://cdn.test/aa/bb/hash.webp"}}
	c.hydrateRosterArt(chars)
	if chars[0].ImageMeta != (ArtMeta{}) {
		t.Errorf("ImageMeta = %+v, want zero", chars[0].ImageMeta)
	}
}
