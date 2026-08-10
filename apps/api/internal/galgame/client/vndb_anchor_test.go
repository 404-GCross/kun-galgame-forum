package client

import "testing"

func TestRefsMap_VndbWorkAnchorWinsRegardlessOfOrder(t *testing.T) {
	for name, refs := range map[string][]catRef{
		"release first": {{Source: "vndb", ExternalID: "r69531"}, {Source: "vndb", ExternalID: "v27920"}},
		"work first":    {{Source: "vndb", ExternalID: "v27920"}, {Source: "vndb", ExternalID: "r69531"}},
	} {
		t.Run(name, func(t *testing.T) {
			if got := refsMap(refs)["vndb"]; got != "v27920" {
				t.Errorf("vndb = %q, want v27920 (the WORK anchor)", got)
			}
		})
	}
}

func TestRefsMap_KeepsFirstWinsForEveryOtherSource(t *testing.T) {
	got := refsMap([]catRef{
		{Source: "dlsite", ExternalID: "RJ249792"},
		{Source: "dlsite", ExternalID: "RJ249793"},
		{Source: "bangumi", ExternalID: "379478"},
	})
	if got["dlsite"] != "RJ249792" {
		t.Errorf("dlsite = %q, want the first anchor RJ249792", got["dlsite"])
	}
	if got["bangumi"] != "379478" {
		t.Errorf("bangumi = %q, want 379478", got["bangumi"])
	}
}

func TestRefsMap_ReleaseOnlyWorkHasNoWorkAnchorToPromote(t *testing.T) {
	if got := refsMap([]catRef{{Source: "vndb", ExternalID: "r69531"}})["vndb"]; got != "r69531" {
		t.Errorf("vndb = %q, want the release anchor carried as-is", got)
	}
}

func TestIsVndbWorkID(t *testing.T) {
	for in, want := range map[string]bool{
		"v27920": true, "v1": true,
		"r69531": false, "v": false, "v12a": false, "vn3": false, "": false,
	} {
		if got := isVndbWorkID(in); got != want {
			t.Errorf("isVndbWorkID(%q) = %v, want %v", in, got, want)
		}
	}
}
