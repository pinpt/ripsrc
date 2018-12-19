package diff

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func assertEqualDiffs(t *testing.T, got, want Diff) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got\n%+v\nwanted\n%+v", diffString(got), diffString(want))
	}
}

func diffString(diff Diff) string {
	res := []string{}
	for _, h := range diff.Hunks {
		res = append(res,
			fmt.Sprintf("%+v", h.Contexts),
			fmt.Sprintf("len(data) = %v", len(h.Data)),
			string(h.Data))
	}
	return strings.Join(res, "\n")
}

func tparse(diff string) Diff {
	return Parse([]byte(diff))
}
