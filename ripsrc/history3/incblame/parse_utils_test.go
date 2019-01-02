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

func TestParseDiffDecl(t *testing.T) {

	cases := []struct {
		Label     string
		In        string
		FromPath  string
		ToPath    string
		ErrMerge  bool
		ErrSpaces bool
		ErrOther  bool
	}{
		{
			Label:    "basic",
			In:       "diff --git a/a.txt b/b.txt",
			FromPath: "a.txt",
			ToPath:   "b.txt",
		},
		{
			Label:    "merge prefix",
			In:       "diff --combined main.go",
			ErrMerge: true,
		},
		{
			Label:    "space in name",
			In:       "diff --git a/a a.txt b/a a.txt",
			FromPath: "a a.txt",
			ToPath:   "a a.txt",
		},
		{
			Label:    "in subdir",
			In:       "diff --git a/a/a.txt b/a/a.txt",
			FromPath: "a/a.txt",
			ToPath:   "a/a.txt",
		},
		// tricky
		// dir name = "a.txt b"
		// file name = "b.txt"
		// actual output from git log
		{
			Label:    "tricky",
			In:       "diff --git a/a.txt b/b.txt b/a.txt b/b.txt",
			FromPath: "a.txt b/b.txt",
			ToPath:   "a.txt b/b.txt",
		},
		{
			Label:     "rename with spaces",
			In:        "diff --git a/a a.txt b/b b.txt",
			ErrSpaces: true,
		},
		// check that invalid format does not panic
		{Label: "invalid format", In: "", ErrOther: true},
		{Label: "invalid format", In: "diff --git ", ErrOther: true},
		{Label: "invalid format", In: "diff --git  ", ErrOther: true},
		{Label: "invalid format", In: "diff --git x/a.txt x/b.txt", ErrOther: true},
	}

	for _, c := range cases {
		fromPath, toPath, err := parseDiffDecl([]byte(c.In))
		t.Logf("test case %v, in %v, wanted %v:%v got %v:%v", c.Label, c.In, c.FromPath, c.ToPath, fromPath, toPath)
		if c.ErrMerge {
			if err != errParseDiffDeclMerge {
				t.Fatalf("wanted errParseDiffDeclMerge, got %v", err)
			}
		} else if c.ErrSpaces {
			if err != errParseDiffDeclRenameWithSpaces {
				t.Fatalf("wanted to get errParseDiffDeclRenameWithSpaces, got %v", err)
			}

		} else if c.ErrOther {
			if err == nil {
				t.Fatal("wanted error returned for this case")
			}
		} else {
			if err != nil {
				t.Fatal("got err", err)
			}
			if fromPath != c.FromPath || toPath != c.ToPath {
				t.Fatal("paths do not match")
			}
		}
	}

}
