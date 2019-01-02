package incblame

import (
	"testing"
)

func TestParseBasic1(t *testing.T) {
	data := `` +
		`diff --git a/main.go b/main.go
new file mode 100644
index 0000000..43f9419
--- /dev/null
+++ b/main.go
@@ -0,0 +1,8 @@
+package main
+
+import "github.com/pinpt/ripsrc/cmd"
+
+func main() {
+	cmd.Execute()
+}
+`

	want := Diff{
		PathPrev: "",
		Path:     "main.go",
		Hunks: []Hunk{
			{
				Locations: []HunkLocation{
					{OpDel, 0, 0},
					{OpAdd, 1, 8},
				},
				Data: []byte(`+package main
+
+import "github.com/pinpt/ripsrc/cmd"
+
+func main() {
+	cmd.Execute()
+}
+
`),
			},
		},
	}

	got := Parse([]byte(data))
	assertEqualDiffs(t, got, want)
}

func TestParseBasic2(t *testing.T) {
	data := `` +
		`diff --git a/main.go b/main.go
index 43f9419..1671209 100644
--- a/main.go
+++ b/main.go
@@ -1,8 +1,6 @@
	package main
	
-import "github.com/pinpt/ripsrc/cmd"
-
	func main() {
-       cmd.Execute()
+  // do nothing
}`

	want := Diff{
		PathPrev: "main.go",
		Path:     "main.go",
		Hunks: []Hunk{
			{
				Locations: []HunkLocation{
					{OpDel, 1, 8},
					{OpAdd, 1, 6},
				},
				Data: []byte(`	package main
	
-import "github.com/pinpt/ripsrc/cmd"
-
	func main() {
-       cmd.Execute()
+  // do nothing
}
`),
			},
		},
	}

	got := Parse([]byte(data))
	assertEqualDiffs(t, got, want)
}

func TestParseBasic3(t *testing.T) {
	data := `` +
		`diff --git a/a.go b/a.go
index 0eb2edb..4422a7d 100644
--- a/a.go
+++ b/a.go
@@ -1,6 +1,9 @@
	a
	b
	c
+1
+2
+3
	d
	e
	f
@@ -20,6 +23,9 @@ e
	e
	w
	q
+4
+5
+6
	z
	x
	c`

	want := Diff{
		PathPrev: "a.go",
		Path:     "a.go",
		Hunks: []Hunk{
			{
				Locations: []HunkLocation{
					{OpDel, 1, 6},
					{OpAdd, 1, 9},
				},
				Data: []byte(`	a
	b
	c
+1
+2
+3
	d
	e
	f
`),
			},
			{
				Locations: []HunkLocation{
					{OpDel, 20, 6},
					{OpAdd, 23, 9},
				},
				Data: []byte(`	e
	w
	q
+4
+5
+6
	z
	x
	c
`),
			},
		},
	}

	got := Parse([]byte(data))
	assertEqualDiffs(t, got, want)
}

func TestParseRename(t *testing.T) {
	data := `` +
		`diff --git a/a.txt b/b.txt
similarity index 100%
rename from a.txt
rename to b.txt`

	want := Diff{
		PathPrev: "a.txt",
		Path:     "b.txt",
	}

	got := Parse([]byte(data))
	assertEqualDiffs(t, got, want)
}

func TestParseNewline1(t *testing.T) {
	data := `` +
		`diff --git a/a.txt b/a.txt
index 7898192..2e65efe 100644
--- a/a.txt
+++ b/a.txt
@@ -1 +1 @@
-a
+a
\ No newline at end of file`

	want := Diff{
		PathPrev: "a.txt",
		Path:     "a.txt",
		Hunks: []Hunk{
			{
				Locations: []HunkLocation{
					{OpDel, 0, 1},
					{OpAdd, 0, 1},
				},
				Data: []byte(`-a
+a
\ No newline at end of file
`),
			},
		},
	}

	got := Parse([]byte(data))
	assertEqualDiffs(t, got, want)
}

func TestParseNewFile(t *testing.T) {
	data := `` +
		`diff --git a/a.txt b/a.txt
new file mode 100644
index 0000000..e69de29`

	want := Diff{
		PathPrev: "",
		Path:     "a.txt",
	}

	got := Parse([]byte(data))
	assertEqualDiffs(t, got, want)
}

func TestParseSpaceInName(t *testing.T) {
	data := `` +
		`diff --git a/a a.txt b/a a.txt
new file mode 100644
index 0000000..7898192
--- /dev/null
+++ b/a a.txt   
@@ -0,0 +1 @@
+a
`

	want := Diff{
		PathPrev: "",
		Path:     "a a.txt",
		Hunks: []Hunk{
			{
				Locations: []HunkLocation{
					{OpDel, 0, 0},
					{OpAdd, 1, 1},
				},
				Data: []byte(`-a
+a
`),
			},
		},
	}

	got := Parse([]byte(data))
	assertEqualDiffs(t, got, want)
}
