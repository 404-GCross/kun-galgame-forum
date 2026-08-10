package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"kun-galgame-api/pkg/errors"
)

type CatalogLabelRelationNode struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	LogoHash  string `json:"logo_hash"`
	WorkCount int    `json:"work_count"`
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
