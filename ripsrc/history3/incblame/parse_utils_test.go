package incblame

import (
	"reflect"
	"testing"
)

func TestParseContext(t *testing.T) {

	cases := []struct {
		In   string
		Want []HunkLocation
	}{
		// regular
		{"@@ -0,0 +1,8 @@", []HunkLocation{{OpDel, 0, 0}, {OpAdd, 1, 8}}},
		// section heading
		{"@@ -3,2 +2,0 @@ package main", []HunkLocation{{OpDel, 3, 2}, {OpAdd, 2, 0}}},
		// when adding to empty file
		{"@@ -0,0 +1 @@ package main", []HunkLocation{{OpDel, 0, 0}, {OpAdd, 0, 1}}},
		// bug encountered extra @ after heading
		{`@@ -13,7 +13,7 @@ const onRE = /^@|^v-on:/ invalid op`, []HunkLocation{{OpDel, 13, 7}, {OpAdd, 13, 7}}},
	}

	for _, c := range cases {
		got := parseContext([]byte(c.In))
		if !reflect.DeepEqual(got, c.Want) {
			t.Errorf("wanted %+v, got %+v, for input %v", c.Want, got, c.In)
		}
	}

}
