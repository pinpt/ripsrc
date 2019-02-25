package branches2

import (
	"reflect"
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/parentsgraph"
)

func assertEqual(t *testing.T, got, want interface{}) {
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wanted %v got %v", want, got)
	}
}

func TestBehindBranch1(t *testing.T) {
	gr := parentsgraph.NewFromMap(map[string][]string{
		"m1": nil,
		"m2": []string{"m1"},
		"m3": []string{"m2"},
		"b":  []string{"m1"},
	})
	def := "m3"
	rfh := newReachableFromHead(gr, def)
	res := behindBranch(gr, rfh, "b", def)
	assertEqual(t, res, 2)
}

func TestBehindBranch2(t *testing.T) {
	gr := parentsgraph.NewFromMap(map[string][]string{
		"a1": nil,
		"a2": []string{"a1"},
		"b1": nil,
		"m":  []string{"a2", "b1"},
		"f":  []string{"a1"},
	})
	def := "m"
	rfh := newReachableFromHead(gr, def)
	res := behindBranch(gr, rfh, "f", def)
	assertEqual(t, res, 3)
}
