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

// data generated with the following command:
// git log -p --reverse --no-abbrev --pretty='short' -m
// using short to show "from" parent for merges, no code for this to show in custom format

func TestBasic(t *testing.T) {

	data := `commit e99cb00954f08c1d33c5935742809868335483bf
Author: User1 <user1@example.com>

    c1

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

commit d497eccaf64c229771f471386cf49e4f653a00cb
Author: User1 <user1@example.com>

    c2

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
			Hash: "e99cb00954f08c1d33c5935742809868335483bf",
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
			Hash: "d497eccaf64c229771f471386cf49e4f653a00cb",
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

func TestNoChangesInMerge(t *testing.T) {

	data := `commit 80775358d2088bb31deedf7d18173ad916b025d3
Merge: 2531fb0b42d18f1dd97e6b1a303f05bf05aef83e 6aac6cbfcdae43f0ebd2351b59794e37e6bd6364
Author: User1 <user1@example.com>

    Merge branch 'b'

commit 80775358d2088bb31deedf7d18173ad916b025d3
Merge: 2531fb0b42d18f1dd97e6b1a303f05bf05aef83e 6aac6cbfcdae43f0ebd2351b59794e37e6bd6364
Author: User1 <user1@example.com>

    Merge branch 'b'

`

	p := New(strings.NewReader(data))
	got, err := p.RunGetAll()
	if err != nil {
		t.Fatal(err)
	}

	want := []Commit{
		{
			Hash: "80775358d2088bb31deedf7d18173ad916b025d3",
		},
		{
			Hash: "80775358d2088bb31deedf7d18173ad916b025d3",
		},
	}

	assertEqualCommits(t, got, want)
}

func TestMergeFromLine(t *testing.T) {

	data := `commit f82b3491fbf1e4fd5666748efe0b198b82d587be (from b7f8fa5c1794de8c7c36b61ba5e7e41e647ae97a)
Merge: 2fbc9d8afd98d677074ab2dc77658dbc2988e853 b7f8fa5c1794de8c7c36b61ba5e7e41e647ae97a
Author: F L <name@name.com>

    Merge branch 'master' into branch

diff --git a/a.txt b/a.txt
new file mode 100644
index 0000000..7898192
--- /dev/null
+++ b/a.txt
@@ -0,0 +1 @@
+a
`

	p := New(strings.NewReader(data))
	got, err := p.RunGetAll()
	if err != nil {
		t.Fatal(err)
	}

	want := []Commit{
		{
			Hash:          "f82b3491fbf1e4fd5666748efe0b198b82d587be",
			MergeDiffFrom: "b7f8fa5c1794de8c7c36b61ba5e7e41e647ae97a",
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
			},
		},
	}

	assertEqualCommits(t, got, want)
}

func TestExtraEmptyLineAfterCommitMessageOnWindows(t *testing.T) {

	data := `commit e99cb00954f08c1d33c5935742809868335483bf
Author: User1 <user1@example.com>

    c1


diff --git a/a.txt b/a.txt
new file mode 100644
index 0000000..7898192
--- /dev/null
+++ b/a.txt
@@ -0,0 +1 @@
+a
+b

commit d497eccaf64c229771f471386cf49e4f653a00cb
Author: User1 <user1@example.com>

    c2

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
			Hash: "e99cb00954f08c1d33c5935742809868335483bf",
			Changes: []Change{
				{
					Diff: tb(`diff --git a/a.txt b/a.txt
new file mode 100644
index 0000000..7898192
--- /dev/null
+++ b/a.txt
@@ -0,0 +1 @@
+a
+b
`),
				},
			},
		},
		{
			Hash: "d497eccaf64c229771f471386cf49e4f653a00cb",
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
