package diff

import (
	"reflect"
	"testing"
)

func assertEqualFiles(t *testing.T, got, want File) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got\n%+v\nwanted\n%+v", got, want)
	}
}

func tl(str string, commit string) Line {
	return Line{Line: []byte(str), Commit: commit}
}
