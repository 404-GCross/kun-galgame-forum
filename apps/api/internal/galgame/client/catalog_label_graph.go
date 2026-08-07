package client

// The 会社 relation graph — the whole corporate family around one label, not
// the one-hop `relations[]` the detail record carries.
//
// It is a SEPARATE upstream call on purpose: the 会社 detail lane is refetched
// on every pagination / filter change of the games grid, and the family tree
// does not change when the reader turns a page.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"kun-galgame-api/pkg/errors"
)

// CatalogLabelRelationNode is one company in the family.
//
// WorkCount is the catalog-wide count (the same number the browse lane shows),
// not the forum-local one: the graph exists to describe the corporate family,
// and walking every sibling's local catalogue to render a badge would cost one
// members query per node.
type CatalogLabelRelationNode struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	// LogoHash — see CatalogTaxonomyItem.LogoHash. "" = no logo.
	LogoHash  string `json:"logo_hash"`
	WorkCount int    `json:"work_count"`
}

// CatalogLabelRelationEdge is one relation, read as "To is the Relation of
// From" — the same reading as the detail record's relations[].
//
// Only the canonical orientations arrive (parent / imprint / spawned /
// succeeded_by); the mirror rows (subsidiary / imprint_of / origin / formerly)
// are implied by the same edge read backwards, so a consumer must never expect
// both halves of a pair.
type CatalogLabelRelationEdge struct {
	From     int64  `json:"from"`
	To       int64  `json:"to"`
	Relation string `json:"relation"`
}

// CatalogLabelRelationGraph is the connected component around the requested
// label: capped upstream (depth ≤ 4, ≤ 60 nodes), cycle-safe, and always
// containing the seed itself — a label with no relations at all comes back as
// a one-node, zero-edge graph.
type CatalogLabelRelationGraph struct {
	Nodes []CatalogLabelRelationNode `json:"nodes"`
	Edges []CatalogLabelRelationEdge `json:"edges"`
}

// CatalogLabelRelationGraph fetches one label's relation graph.
//
// found=false on an unknown id — a 404 here is not an error for a page that
// simply has no family section to draw. Unlike the detail lane there is no
// merged-id (301) leg: this face is never linked to directly, it is only ever
// asked about an id the detail lane already resolved.
func (c *GalgameClient) CatalogLabelRelationGraph(ctx context.Context, id string) (*CatalogLabelRelationGraph, bool, *errors.AppError) {
	// Open population, like every identity lane: an r18 maker's corporate
	// family is identity, not content.
	q := openPopulation(url.Values{})
	status, env, appErr := c.getV1Envelope(ctx, "/catalog/labels/"+id+"/relation-graph", q)
	if appErr != nil {
		return nil, false, appErr
	}
	if status == http.StatusNotFound {
		return nil, false, nil
	}
	if env.Code != 0 {
		return nil, false, errors.New(env.Code, env.Message, status)
	}
	var graph CatalogLabelRelationGraph
	if err := json.Unmarshal(env.Data, &graph); err != nil {
		return nil, false, errors.ErrInternal("解析 Catalog 会社关系图响应失败")
	}
	return &graph, true, nil
}
