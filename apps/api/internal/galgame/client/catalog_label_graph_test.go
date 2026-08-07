package client

// The 会社 relation-graph read lane. Hermetic: an httptest server plays the
// catalog, pinned to the contract wave 188 fixed — enveloped
// {nodes:[{id,name,logo_hash,work_count}], edges:[{from,to,relation}]}.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// graphCatalog plays a catalog holding the VisualArt's family (a parent, two
// imprints) plus a lone maker with no relations and an unknown id. It records
// the paths and queries it was asked for.
func graphCatalog(t *testing.T) (*GalgameClient, func() (string, string)) {
	t.Helper()
	var (
		mu    sync.Mutex
		path  string
		query string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		path, query = r.URL.Path, r.URL.RawQuery
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/catalog/labels/24/relation-graph":
			_, _ = w.Write([]byte(`{"code":0,"message":"成功","data":{"nodes":[` +
				`{"id":24,"name":"Key","logo_hash":"aabbccdd","work_count":33},` +
				`{"id":993,"name":"VisualArt's","logo_hash":"","work_count":120},` +
				`{"id":994,"name":"Na-Ga","logo_hash":"11223344","work_count":0}],` +
				`"edges":[{"from":24,"to":993,"relation":"parent"},` +
				`{"from":994,"to":993,"relation":"parent"}]}}`))
		case "/v1/catalog/labels/309/relation-graph":
			// A maker with no recorded relations still gets itself back.
			_, _ = w.Write([]byte(`{"code":0,"message":"成功","data":{"nodes":[` +
				`{"id":309,"name":"无关系社","logo_hash":"","work_count":2}],"edges":[]}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"code":4,"message":"资源不存在"}`))
		}
	}))
	t.Cleanup(srv.Close)
	return New(srv.URL, "test-key", "https://cdn.test/image"), func() (string, string) {
		mu.Lock()
		defer mu.Unlock()
		return path, query
	}
}

func TestCatalogLabelRelationGraphReadsTheContractShape(t *testing.T) {
	c, asked := graphCatalog(t)

	graph, found, appErr := c.CatalogLabelRelationGraph(context.Background(), "24")
	if appErr != nil {
		t.Fatalf("unexpected error: %v", appErr)
	}
	if !found {
		t.Fatal("found = false for a live label")
	}
	if len(graph.Nodes) != 3 || len(graph.Edges) != 2 {
		t.Fatalf("graph = %d nodes / %d edges, want 3/2", len(graph.Nodes), len(graph.Edges))
	}
	if graph.Nodes[0].Name != "Key" || graph.Nodes[0].WorkCount != 33 {
		t.Errorf("seed node = %+v, want Key/33", graph.Nodes[0])
	}
	// "" is a real answer (no logo), never an error.
	if graph.Nodes[1].LogoHash != "" {
		t.Errorf("logoless node kept %q", graph.Nodes[1].LogoHash)
	}
	// The edge reads "to is the relation of from" — losing the direction would
	// invert the whole family tree.
	if e := graph.Edges[0]; e.From != 24 || e.To != 993 || e.Relation != "parent" {
		t.Errorf("edge[0] = %+v, want 24→993 parent", e)
	}

	path, query := asked()
	if path != "/v1/catalog/labels/24/relation-graph" {
		t.Errorf("path = %q", path)
	}
	// Identity lane: a maker's corporate family must not be age-gated away.
	if query != "nsfw=1" {
		t.Errorf("query = %q, want nsfw=1", query)
	}
}

func TestCatalogLabelRelationGraphKeepsTheLoneMaker(t *testing.T) {
	c, _ := graphCatalog(t)

	graph, found, appErr := c.CatalogLabelRelationGraph(context.Background(), "309")
	if appErr != nil || !found {
		t.Fatalf("found=%v err=%v, want a one-node graph", found, appErr)
	}
	if len(graph.Nodes) != 1 || len(graph.Edges) != 0 {
		t.Fatalf("graph = %d nodes / %d edges, want 1/0", len(graph.Nodes), len(graph.Edges))
	}
}

func TestCatalogLabelRelationGraphMissIsNotAnError(t *testing.T) {
	c, _ := graphCatalog(t)

	graph, found, appErr := c.CatalogLabelRelationGraph(context.Background(), "999999")
	if appErr != nil {
		t.Fatalf("a 404 became an error: %v", appErr)
	}
	if found || graph != nil {
		t.Fatalf("found=%v graph=%v, want the miss reported as found=false", found, graph)
	}
}
