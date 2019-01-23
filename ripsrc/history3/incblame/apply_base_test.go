package incblame

import (
	"reflect"
	"testing"
)

// TODO: rename to assertEqualLines
func assertEqualFiles(t *testing.T, got, want Blame) {
	t.Helper()
	if len(got.Lines) != len(want.Lines) {
		t.Errorf("len mismatch, got\n%+v\nwanted\n%+v", got, want)
		return
	}
	ok := true
	for i := range got.Lines {
		g := got.Lines[i]
		w := want.Lines[i]
		if !reflect.DeepEqual(g, w) {
			t.Errorf("line mismatch, got \n%+v\nwanted\n%+v", g, w)
			ok = false
		}
	}
	if !ok {
		t.Errorf("got\n%+v\nwanted\n%+v", got, want)
	}
}

func file(hash string, lines ...*Line) Blame {
	return Blame{Commit: hash, Lines: lines}
}

func line(str string, commit string) *Line {
	return &Line{Line: []byte(str), Commit: commit}
}
