package diff

import (
	"reflect"
	"testing"
)

func TestParseContext(t *testing.T) {

	cases := []struct {
		In   string
		Want []HunkContext
	}{
		// regular
		{"@@ -0,0 +1,8 @@", []HunkContext{{OpDel, 0, 0}, {OpAdd, 1, 8}}},
		// section heading
		{"@@ -3,2 +2,0 @@ package main", []HunkContext{{OpDel, 3, 2}, {OpAdd, 2, 0}}},
	}

	for _, c := range cases {
		got := parseContext([]byte(c.In))
		if !reflect.DeepEqual(got, c.Want) {
			t.Errorf("wanted %+v, got %+v, for input %v", c.Want, got, c.In)
		}
	}

}
