package parser

import (
	"reflect"
	"strings"
	"testing"
)

func assertEqualCommits(t *testing.T, got, want []Commit) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("wanted %v commits, got %v", len(want), len(got))
	}
	for i := range got {
		g := got[i]
		w := want[i]
		if g.Hash != w.Hash {
			t.Error("hash")
		}
		if !reflect.DeepEqual(g.Parents, w.Parents) {
			t.Error("parents")
		}
		if !reflect.DeepEqual(g.Changes, w.Changes) {
			t.Error("changes")
		}
		if !reflect.DeepEqual(g, w) {
			t.Fatalf("wanted %v\ngot\n%v", w, g)
		}
	}
}

func TestBasic(t *testing.T) {

	// output of
	// git log -p -c --reverse --no-abbrev --pretty='format:!Hash: %H%n!Parents: %P'

	data := `!Hash: e99cb00954f08c1d33c5935742809868335483bf
!Parents: 
diff --git a/a.txt b/a.txt
new file mode 100644
index 0000000..7898192
--- /dev/null
+++ b/a.txt
@@ -0,0 +1 @@
+a
diff --git a/b.txt b/b.txt
new file mode 100644
index 0000000..6178079
--- /dev/null
+++ b/b.txt
@@ -0,0 +1 @@
+b

!Hash: d497eccaf64c229771f471386cf49e4f653a00cb
!Parents: e99cb00954f08c1d33c5935742809868335483bf
diff --git a/a.txt b/a.txt
index 7898192..e61ef7b 100644
--- a/a.txt
+++ b/a.txt
@@ -1 +1 @@
-a
+aa
`

	p := New(strings.NewReader(data))
	got, err := p.RunGetAll()
	if err != nil {
		t.Fatal(err)
	}

	want := []Commit{
		{
			Hash:    "e99cb00954f08c1d33c5935742809868335483bf",
			Parents: nil,
			Changes: []Change{
				{
					Diff: tb(`diff --git a/a.txt b/a.txt
new file mode 100644
index 0000000..7898192
--- /dev/null
+++ b/a.txt
@@ -0,0 +1 @@
+a
`),
				},
				{
					Diff: tb(`diff --git a/b.txt b/b.txt
new file mode 100644
index 0000000..6178079
--- /dev/null
+++ b/b.txt
@@ -0,0 +1 @@
+b
`),
				},
			},
		},
		{
			Hash:    "d497eccaf64c229771f471386cf49e4f653a00cb",
			Parents: []string{"e99cb00954f08c1d33c5935742809868335483bf"},
			Changes: []Change{
				{
					Diff: tb(`diff --git a/a.txt b/a.txt
index 7898192..e61ef7b 100644
--- a/a.txt
+++ b/a.txt
@@ -1 +1 @@
-a
+aa
`),
				},
			},
		},
	}

	assertEqualCommits(t, got, want)
}

func tb(s string) []byte {
	return []byte(s)
}
