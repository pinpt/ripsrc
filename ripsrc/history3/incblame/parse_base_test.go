package incblame

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
	res = append(res, `paths:'`+diff.PathPrev+`:`+diff.Path+`'`)
	for _, h := range diff.Hunks {
		res = append(res,
			fmt.Sprintf("%+v", h.Locations),
			fmt.Sprintf("len(data) = %v", len(h.Data)),
			string(h.Data))
	}
	return strings.Join(res, "\n")
}

func tparse(diff string) Diff {
	return Parse([]byte(diff))
}

func tparseDiffs(diffs ...string) (res []Diff) {
	for _, d := range diffs {
		res = append(res, Parse([]byte(d)))
	}
	return
}
