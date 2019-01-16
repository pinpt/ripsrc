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

func TestMultiDiff(t *testing.T) {
	c1 := "c1"
	c2 := "c2"
	f := Apply(Blame{}, Parse([]byte(multiDiffA1)), c1, "")
	f = Apply(f, Parse([]byte(multiDiffA2)), c2, "")

	want := file(c2,
		line("a", c1),
		line("b", c1),
		line("c", c1),
		line("1", c2),
		line("2", c2),
		line("3", c2),
		line("d", c1),
		line("e", c1),
		line("f", c1),
		line("g", c1),
		line("h", c1),
		line("j", c1),
		line("k", c1),
		line("l", c1),
		line(";", c1),
		line("p", c1),
		line("o", c1),
		line("i", c1),
		line("y", c1),
		line("t", c1),
		line("r", c1),
		line("e", c1),
		line("e", c1),
		line("w", c1),
		line("q", c1),
		line("4", c2), // line 25
		line("5", c2),
		line("6", c2),
		line("z", c1),
		line("x", c1),
		line("c", c1),
		line("v", c1),
		line("b", c1),
		line("n", c1),
		line("m", c1),
	)

	assertEqualFiles(t, f, want)
}
