package client

// Intrinsic dimensions for the catalog's ENTITY artwork — character busts and
// 立绘 today, and anything else the catalog hands over as a bare CDN URL.
//
// The work-level media blocks (covers, screenshots) already carry width /
// height / thumbhash, because the catalog aggregates those through its media
// read lane. Entity art does not: the character face resolves image_hash and
// figure_hash straight to URLs and says nothing about their shape.
//
// That gap is visible on the page rather than theoretical. Without a ratio the
// only honest frame is a guess, and a guess is wrong twice: it reserves the
// wrong box (the layout jumps when the picture lands) and it forces every
// figure into one shape, when in fact a Getchu 立绘 is square for some titles
// and distinctly tall for others.
//
// The hash is recoverable from the URL — the CDN path IS {aa}/{bb}/{hash}.webp
// — so kungal can ask image_service directly, in one batch per response, off a
// cache that never expires (metadata is immutable per content hash). The
// thumbhash rides along for free, which is the blur-up.

import (
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/pkg/imageclient"
)

// ImageMetaResolver is the shape of imageclient.MetaResolver.Resolve. Injected
// rather than constructed so the client keeps one image dependency (the CDN
// base it already has) and the resolver's permanent cache stays process-wide.
type ImageMetaResolver func(hashes []string) map[string]imageclient.ImageMeta

// SetImageMetaResolver wires the resolver used to size entity artwork. Unset
// (the default, and every test) simply means no dimensions are published: the
// frontend then falls back to its own frame, which is the pre-existing
// behaviour rather than a broken one.
func (c *GalgameClient) SetImageMetaResolver(resolve ImageMetaResolver) {
	c.imageMeta = resolve
}

// ArtMeta is one artwork's intrinsic shape. Zero width/height means "not
// resolved" — an unconfigured image_service, a hash it does not know, or a
// URL that is not content-addressed at all. Consumers must treat it as unknown
// and keep their own default frame; it is never a claim that the image is 0×0.
type ArtMeta struct {
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Thumbhash string `json:"thumbhash"`
}

// ArtMetaDTO publishes a resolved shape, or nothing at all. A zero width or
// height is dropped rather than emitted: "unknown" and "0 pixels wide" must not
// look the same on the wire, or a consumer will divide by it.
func ArtMetaDTO(m ArtMeta) *dto.GalgameArtMeta {
	if m.Width <= 0 || m.Height <= 0 {
		return nil
	}
	return &dto.GalgameArtMeta{Width: m.Width, Height: m.Height, Thumbhash: m.Thumbhash}
}

// resolveArtMeta answers metadata for a batch of CDN URLs, keyed by the URL it
// was asked about. One image_service call for the whole response; a URL with no
// answer is simply absent from the map.
func (c *GalgameClient) resolveArtMeta(urls []string) map[string]ArtMeta {
	if c.imageMeta == nil || len(urls) == 0 {
		return nil
	}
	hashByURL := make(map[string]string, len(urls))
	hashes := make([]string, 0, len(urls))
	for _, u := range urls {
		hash := hashFromURL(u)
		if hash == "" {
			continue
		}
		if _, seen := hashByURL[u]; seen {
			continue
		}
		hashByURL[u] = hash
		hashes = append(hashes, hash)
	}
	if len(hashes) == 0 {
		return nil
	}
	metaByHash := c.imageMeta(hashes)
	out := make(map[string]ArtMeta, len(hashByURL))
	for u, hash := range hashByURL {
		if m, ok := metaByHash[hash]; ok {
			out[u] = ArtMeta{Width: m.Width, Height: m.Height, Thumbhash: m.Thumbhash}
		}
	}
	return out
}
