package branches2

import (
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/parentsgraph"
)

func TestGetMergeCommit1(t *testing.T) {
	gr := parentsgraph.NewFromMap(map[string][]string{
		"m1": nil,
		"b1": []string{"m1"},
		"m2": []string{"m1", "b1"},
	})
	cache := newReachableFromHead(gr, "m2")
	got := getMergeCommit(gr, cache, "b1")
	want := "m2"
	if got != want {
		t.Errorf("wanted %v got %v", want, got)
	}
}

func TestGetMergeCommit2(t *testing.T) {
	gr := parentsgraph.NewFromMap(map[string][]string{
		"m1": nil,
		"b1": []string{"m1"},
		"m2": []string{"m1", "b1"},
	})
	cache := newReachableFromHead(gr, "m1")
	got := getMergeCommit(gr, cache, "b1")
	want := ""
	if got != want {
		t.Errorf("wanted %v got %v", want, got)
	}
}

func TestGetMergeCommit3(t *testing.T) {
	gr := parentsgraph.NewFromMap(map[string][]string{
		"m1": nil,
		"b1": []string{"m1"},
		"m2": []string{"m1", "b1"},
		"m3": []string{"m1", "b1"},
		"m4": []string{"m2", "m3"},
	})
	cache := newReachableFromHead(gr, "m4")
	got := getMergeCommit(gr, cache, "b1")
	want := "m2"
	if got != want {
		t.Errorf("wanted %v got %v", want, got)
	}
}
