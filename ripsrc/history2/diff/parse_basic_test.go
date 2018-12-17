package diff

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
		Hunks: []Hunk{
			{
				Contexts: []HunkContext{
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
		Hunks: []Hunk{
			{
				Contexts: []HunkContext{
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
		Hunks: []Hunk{
			{
				Contexts: []HunkContext{
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
				Contexts: []HunkContext{
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
