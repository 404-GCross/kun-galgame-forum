package client

import (
	"cmp"
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"kun-galgame-api/pkg/errors"
)

// CatalogLabelRelationNode carries two generations of the node's name. The
// relation graph was the last label projection still sending a bare "name",
// and the day catalog reshaped it to the wave 209 primitive every node on the
// live 会社关系图 rendered with an empty label — the decode found no "name"
// key and reported nothing, so the graph drew unnamed boxes instead of failing.
type CatalogLabelRelationNode struct {
	ID          int64                       `json:"id"`
	Name        string                      `json:"name"`
	DisplayName string                      `json:"display_name"`
	Localized   map[string]catLocalizedName `json:"localized"`
	LogoHash    string                      `json:"logo_hash"`
	WorkCount   int                         `json:"work_count"`
}

func (n *CatalogLabelRelationNode) LocalName(ctx context.Context) string {
	return CatalogEntityName(ctx, n.Localized, cmp.Or(n.DisplayName, n.Name), "")
}

type CatalogLabelRelationEdge struct {
	From     int64  `json:"from"`
	To       int64  `json:"to"`
	Relation string `json:"relation"`
}

type CatalogLabelRelationGraph struct {
	Nodes []CatalogLabelRelationNode `json:"nodes"`
	Edges []CatalogLabelRelationEdge `json:"edges"`
}

func (c *GalgameClient) CatalogLabelRelationGraph(ctx context.Context, id string) (*CatalogLabelRelationGraph, bool, *errors.AppError) {
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
