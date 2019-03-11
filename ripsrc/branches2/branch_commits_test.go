package branches2

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/parentsgraph"
)

func assertCommits(t *testing.T, got, want []string) {
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wanted %v got %v", want, got)
	}
}

func TestProcessBranchBasic1(t *testing.T) {
	gr := parentsgraph.NewFromMap(map[string][]string{
		"m1": nil,
		"b1": []string{"m1"},
	})
	cache := newReachableFromHead(gr, "m1")
	got, branchedFrom := branchCommits(gr, "m1", cache, "b1")
	assertCommits(t, got, []string{"b1"})
	assertCommits(t, branchedFrom, []string{"m1"})
}

func TestProcessBranchMerged1(t *testing.T) {
	gr := parentsgraph.NewFromMap(map[string][]string{
		"m1": nil,
		"b1": []string{"m1"},
		"b2": []string{"b1"},
		"m2": []string{"m1", "b2"},
	})
	cache := newReachableFromHead(gr, "m2")
	got, branchedFrom := branchCommits(gr, "m2", cache, "b2")
	assertCommits(t, got, []string{"b1", "b2"})
	assertCommits(t, branchedFrom, []string{"m1"})
}

// Checking that it is fast enough for merged branches
// This is fast already, no need to optimize
// 2ms per branch
func BenchmarkProcessBranchMerged1(b *testing.B) {
	m := map[string][]string{}
	m["0"] = nil
	for i := 0; i < 10000; i++ {
		m[strconv.Itoa(i+1)] = []string{strconv.Itoa(i)}
	}
	m["b"] = []string{"9900"}
	m["m"] = []string{"b", "9999"}
	gr := parentsgraph.NewFromMap(m)
	cache := newReachableFromHead(gr, "m")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		branchCommits(gr, "m", cache, "b")
	}
}

func TestProcessBranchMultipleFromMaster(t *testing.T) {
	gr := parentsgraph.NewFromMap(map[string][]string{
		"m1": nil,
		"m2": []string{"m1"},
		"b1": []string{"m1"},
		"b2": []string{"b1", "m2"},
	})
	cache := newReachableFromHead(gr, "m2")
	got, branchedFrom := branchCommits(gr, "m2", cache, "b2")
	assertCommits(t, got, []string{"b1", "b2"})
	// the result should not contain m2, because we also depend on m1 which is before m2
	assertCommits(t, branchedFrom, []string{"m1"})
}

func TestProcessBranchSeparateRoot(t *testing.T) {
	gr := parentsgraph.NewFromMap(map[string][]string{
		"m1": nil,
		"b1": nil,
		"b2": []string{"b1"},
	})
	cache := newReachableFromHead(gr, "m1")
	got, branchedFrom := branchCommits(gr, "m1", cache, "b2")
	assertCommits(t, got, []string{"b1", "b2"})
	assertCommits(t, branchedFrom, nil)
}

func TestDedupLinearFromHead1(t *testing.T) {
	gr := parentsgraph.NewFromMap(map[string][]string{
		"m1": nil,
		"m2": []string{"m1"},
		"m3": []string{"m2"},
	})
	got := dedupLinearFromHead(gr, []string{"m1", "m3"}, "m3")
	assertCommits(t, got, []string{"m1"})
}

func TestDedupLinearFromHead2(t *testing.T) {
	gr := parentsgraph.NewFromMap(map[string][]string{
		"m1": nil,
		"m2": []string{"m1"},
		"m3": []string{"m1"},
		"m4": []string{"m2", "m3"},
		"b1": []string{"m2", "m3"},
	})
	got := dedupLinearFromHead(gr, []string{"m2", "m3"}, "m4")
	assertCommits(t, got, []string{"m2", "m3"})
}

func TestProcessBranchMultiBranch(t *testing.T) {
	gr := parentsgraph.NewFromMap(map[string][]string{
		"m1": nil,
		"b1": []string{"m1"},
		"c1": []string{"m1"},
		"d1": []string{"b1", "c1"},
	})
	cache := newReachableFromHead(gr, "m1")
	got, branchedFrom := branchCommits(gr, "m1", cache, "d1")
	assertCommits(t, got, []string{"b1", "c1", "d1"})
	assertCommits(t, branchedFrom, []string{"m1"})
}

func TestProcessBranchMultiSource(t *testing.T) {
	gr := parentsgraph.NewFromMap(map[string][]string{
		"m1": nil,
		"m2": []string{"m1"},
		"m3": []string{"m1"},
		"m4": []string{"m2", "m3"},
		"b1": []string{"m2", "m3"},
	})
	cache := newReachableFromHead(gr, "m4")
	got, branchedFrom := branchCommits(gr, "m4", cache, "b1")
	assertCommits(t, got, []string{"b1"})
	assertCommits(t, branchedFrom, []string{"m2", "m3"})
}

func TestProcessBranchDups(t *testing.T) {
	gr := parentsgraph.NewFromMap(map[string][]string{
		"m1": nil,
		"b1": []string{"m1"},
		"b2": []string{"b1"},
		"b3": []string{"b1"},
		"b4": []string{"b2", "b3"},
	})
	cache := newReachableFromHead(gr, "m1")
	got, branchedFrom := branchCommits(gr, "m1", cache, "b4")
	assertCommits(t, got, []string{"b2", "b1", "b3", "b4"})
	assertCommits(t, branchedFrom, []string{"m1"})
}
