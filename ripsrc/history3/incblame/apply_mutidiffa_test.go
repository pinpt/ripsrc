package incblame

import (
	"testing"
)

const multiDiffA1 = `diff --git a/a.go b/a.go
new file mode 100644
index 0000000..0eb2edb
--- /dev/null
+++ b/a.go
@@ -0,0 +1,29 @@
+a
+b
+c
+d
+e
+f
+g
+h
+j
+k
+l
+;
+p
+o
+i
+y
+t
+r
+e
+e
+w
+q
+z
+x
+c
+v
+b
+n
+m`

const multiDiffA2 = `diff --git a/a.go b/a.go
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

func TestMultiDiffA2(t *testing.T) {
	c1 := "c1"
	c2 := "c2"
	f := Apply(nil, Parse([]byte(multiDiffA1)), c1)
	f = applyOneParent(f, Parse([]byte(multiDiffA2)), c2)

	want := Blame{
		Lines: []Line{
			tl("a", c1),
			tl("b", c1),
			tl("c", c1),
			tl("1", c2),
			tl("2", c2),
			tl("3", c2),
			tl("d", c1),
			tl("e", c1),
			tl("f", c1),
			tl("g", c1),
			tl("h", c1),
			tl("j", c1),
			tl("k", c1),
			tl("l", c1),
			tl(";", c1),
			tl("p", c1),
			tl("o", c1),
			tl("i", c1),
			tl("y", c1),
			tl("t", c1),
			tl("r", c1),
			tl("e", c1),
			tl("e", c1),
			tl("w", c1),
			tl("q", c1),
			tl("4", c2), // line 25
			tl("5", c2),
			tl("6", c2),
			tl("z", c1),
			tl("x", c1),
			tl("c", c1),
			tl("v", c1),
			tl("b", c1),
			tl("n", c1),
			tl("m", c1),
		},
	}

	assertEqualFiles(t, f, want)
}
