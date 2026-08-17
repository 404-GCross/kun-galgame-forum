package service

import (
	"reflect"
	"testing"
)

func TestSanitizeMutedKeysDropsRetiredWikiKeys(t *testing.T) {
	got := SanitizeMutedKeys([]string{
		"liked",
		"wiki:approved",
		"wiki:declined",
		"chat",
		"liked",
		"not-a-key",
	})
	want := []string{"liked", "chat"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SanitizeMutedKeys = %#v, want %#v", got, want)
	}
}

func TestSplitMutedIgnoresRetiredWikiKeys(t *testing.T) {
	local, chatMuted := SplitMuted([]string{"liked", "wiki:banned", "chat"})
	if !chatMuted {
		t.Fatal("expected chat muted")
	}
	if !reflect.DeepEqual(local, []string{"liked"}) {
		t.Fatalf("local = %#v, want [liked]", local)
	}
}
