package handler

// Taxonomy picker + staff READ-BACK proxy (A2-3 / doc 106 R11, A2-1g).
//
// The admin taxonomy console edits the WIKI's taxonomy rows, and its update
// payload is a wholesale replacement — so every field it cannot read back is a
// field it silently ERASES on save. It used to prefill from the public browse
// lanes, which is now impossible for two independent reasons:
//
//  1. those lanes moved to the CATALOG id space (P2/R1), while the write ops
//     still take wiki ids — sending one to the other would edit a different row;
//  2. the catalog projection drops fields the wiki row still has (a maker's
//     `original`, a tag's wiki category), so saving would blank them.
//
// The staff face grew a matching read pair for exactly this, and these routes
// forward to it with the caller's Bearer. The whole lane — search, prefill,
// save — is wiki ids end to end, and stays that way until the editing engine
// retires this face.

import (
	"strconv"
	"strings"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"

	"github.com/gofiber/fiber/v3"
)

// StaffTaxonomyHandler proxies the staff read-back pair.
type StaffTaxonomyHandler struct {
	galgameClient *client.GalgameClient
}

func NewStaffTaxonomyHandler(galgameClient *client.GalgameClient) *StaffTaxonomyHandler {
	return &StaffTaxonomyHandler{galgameClient: galgameClient}
}

// staffFamilies is the closed set of taxonomy families the staff face serves.
// A closed vocabulary here keeps the path parameter from becoming an open
// proxy into arbitrary upstream routes.
var staffFamilies = map[string]bool{
	"tag": true, "official": true, "engine": true, "series": true,
}

func (h *StaffTaxonomyHandler) family(c fiber.Ctx) (string, bool) {
	f := strings.ToLower(strings.TrimSpace(c.Params("family")))
	return f, staffFamilies[f]
}

// Search — GET /galgame-taxonomy/:family/search?q=
//
// Serves BOTH consumers of a taxonomy picker: the admin console's edit tabs and
// the galgame submission form. They want the same thing — a wiki id for a name
// — so they share one route, gated on being signed in (A2-1g). Editing
// authority is enforced where editing happens: on the write ops, and again
// upstream. The full-record read-back below stays admin-only because it returns
// an editable record, not an identity.
func (h *StaffTaxonomyHandler) Search(c fiber.Ctx) error {
	family, ok := h.family(c)
	if !ok {
		return response.Error(c, errors.ErrBadRequest("不支持的词表类型"))
	}
	items, appErr := h.galgameClient.TaxonomyPickerSearch(
		c.Context(), family, c.Query("q"), middleware.GetAccessToken(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, items)
}

// Detail — GET /galgame-taxonomy/:family/:id (id is a WIKI id)
func (h *StaffTaxonomyHandler) Detail(c fiber.Ctx) error {
	family, ok := h.family(c)
	if !ok {
		return response.Error(c, errors.ErrBadRequest("不支持的词表类型"))
	}
	id := c.Params("id")
	if n, err := strconv.Atoi(id); err != nil || n <= 0 {
		return response.Error(c, errors.ErrBadRequest("无效的词条 ID"))
	}
	rec, appErr := h.galgameClient.StaffTaxonomyDetail(
		c.Context(), family, id, middleware.GetAccessToken(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, rec)
}
