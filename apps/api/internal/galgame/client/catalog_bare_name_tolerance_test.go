package client

import (
	"context"
	"encoding/json"
	"testing"
)

// Every projection here still ships a bare "name" today and is scheduled to be
// reshaped to the wave 209 primitive. The relation graph was reshaped without a
// forum release and rendered blank labels in production for a day, so each of
// these decodes both shapes and this test is what keeps the old branch alive.
func TestCatalogProjectionsReadBothNameShapes(t *testing.T) {
	const reshaped = `{"display_name":"ねこねこソフト","localized":{"zh-Hans":{"value":"猫猫社","kind":"translation"}}}`
	const bare = `{"name":"ねこねこソフト"}`

	ctx := context.Background()

	labels := map[string]func(string) string{
		"work engine": func(raw string) string {
			var v catWorkEngine
			mustDecode(t, raw, &v)
			return v.Label(ctx)
		},
		"work series": func(raw string) string {
			var v catWorkSeries
			mustDecode(t, raw, &v)
			return v.Label(ctx)
		},
		"work tag": func(raw string) string {
			var v catWorkTag
			mustDecode(t, raw, &v)
			return v.Label()
		},
		"tag detail": func(raw string) string {
			var v CatalogTagDetail
			mustDecode(t, raw, &v)
			return v.Label()
		},
		"engine detail": func(raw string) string {
			var v CatalogEngineDetail
			mustDecode(t, raw, &v)
			return v.Label(ctx)
		},
		"series detail": func(raw string) string {
			var v CatalogSeriesDetail
			mustDecode(t, raw, &v)
			return v.Label(ctx)
		},
		"relation node": func(raw string) string {
			var v CatalogLabelRelationNode
			mustDecode(t, raw, &v)
			return v.LocalName(ctx)
		},
	}

	for name, label := range labels {
		if got := label(reshaped); got != "猫猫社" {
			t.Errorf("%s on the reshaped wire = %q, want 猫猫社", name, got)
		}
		if got := label(bare); got != "ねこねこソフト" {
			t.Errorf("%s on the bare wire = %q, want ねこねこソフト", name, got)
		}
	}
}

func mustDecode(t *testing.T, raw string, into any) {
	t.Helper()
	if err := json.Unmarshal([]byte(raw), into); err != nil {
		t.Fatalf("decode %s: %v", raw, err)
	}
}
