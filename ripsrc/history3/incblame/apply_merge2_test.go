package incblame

import "testing"

// data source
// testdata/merge_basic
func TestMerge2Basic(t *testing.T) {
	diff1 := `diff --git a/main.go b/main.go
new file mode 100644
index 0000000..1661cbb
--- /dev/null
+++ b/main.go
@@ -0,0 +1,4 @@
+package main
+
+func main(){
+}`

	diff2 := `diff --git a/main.go b/main.go
index 1661cbb..4cd4b38 100644
--- a/main.go
+++ b/main.go
@@ -1,4 +1,5 @@
	package main
 
	func main(){
+// A
	}`

	diff3 := `diff --git a/main.go b/main.go
index 1661cbb..1dbddb0 100644
--- a/main.go
+++ b/main.go
@@ -1,4 +1,5 @@
	package main
 
	func main(){
+// M
	}`

	mergeDiff1 := `diff --git a/main.go b/main.go
index 1dbddb0..904d55b 100644
--- a/main.go
+++ b/main.go
@@ -2,4 +2,5 @@ package main
 
 func main(){
 // M
+// A
 }`

	mergeDiff2 := `diff --git a/main.go b/main.go
index 4cd4b38..904d55b 100644
--- a/main.go
+++ b/main.go
@@ -1,5 +1,6 @@
 package main
 
 func main(){
+// M
 // A
 }`

	c1base := "c1base"
	c2branch := "c2branch"
	c3master := "c3master"
	c4merge := "c4merge"

	f1base := Apply(Blame{}, tparse(diff1), c1base, "")
	f2branch := Apply(f1base, tparse(diff2), c2branch, "")
	f3master := Apply(f1base, tparse(diff3), c3master, "")
	f4merge := ApplyMerge(
		[]Blame{f3master, f2branch},
		tparseDiffs(mergeDiff1, mergeDiff2),
		c4merge, "")

	want := file(c4merge,
		line(`package main`, c1base),
		line(``, c1base),
		line(`func main(){`, c1base),
		line(`// M`, c3master),
		line(`// A`, c2branch),
		line(`}`, c1base),
	)

	assertEqualFiles(t, f4merge, want)
}

// data source
// testdata/merge_multidiff
func TestMerge2Multidiff(t *testing.T) {
	diff1 := `diff --git a/a.go b/a.go
new file mode 100644
index 0000000..3440bbd
--- /dev/null
+++ b/a.go
@@ -0,0 +1,21 @@
+q
+w
+e
+r
+t
+y
+u
+i
+o
+p
+a
+s
+d
+f
+g
+h
+j
+k
+l
+z
+x`

	diff2 := `diff --git a/a.go b/a.go
index 3440bbd..7a73855 100644
--- a/a.go
+++ b/a.go
@@ -1,3 +1,8 @@
+1
+2
+3
+4
+5
	q
	w
	e
@@ -10,12 +15,14 @@ o
	p
	a
	s
+1
+2
+3
+4
+5
	d
	f
	g
	h
-j
-k
-l
	z
	x`

	diff3 := `diff --git a/a.go b/a.go
index 3440bbd..0bb73a0 100644
--- a/a.go
+++ b/a.go
@@ -7,6 +7,14 @@ y
	u
	i
	o
+9
+9
+9
+9
+9
+9
+9
+9
	p
	a
	s
@@ -17,5 +25,3 @@ h
	j
	k
	l
-z
-x`

	mergeDiff1 := `diff --git a/a.go b/a.go
index 0bb73a0..e5f72e3 100644
--- a/a.go
+++ b/a.go
@@ -1,3 +1,8 @@
+1
+2
+3
+4
+5
 q
 w
 e
@@ -18,10 +23,14 @@ o
 p
 a
 s
+1
+2
+3
+4
+5
 d
 f
 g
 h
-j
-k
-l
+z
+x`

	mergeDiff2 := `diff --git a/a.go b/a.go
index 7a73855..e5f72e3 100644
--- a/a.go
+++ b/a.go
@@ -12,6 +12,14 @@ y
 u
 i
 o
+9
+9
+9
+9
+9
+9
+9
+9
 p
 a
 s`

	c1base := "c1base"
	c2branch := "c2branch"
	c3master := "c3master"
	c4merge := "c4merge"

	f1base := Apply(Blame{}, tparse(diff1), c1base, "")
	f2branch := Apply(f1base, tparse(diff2), c2branch, "")
	f3master := Apply(f1base, tparse(diff3), c3master, "")
	f4merge := ApplyMerge(
		[]Blame{f3master, f2branch},
		tparseDiffs(mergeDiff1, mergeDiff2),
		c4merge, "")

	want := file(c4merge,

		line(`1`, c2branch),
		line(`2`, c2branch),
		line(`3`, c2branch),
		line(`4`, c2branch),
		line(`5`, c2branch),
		line(`q`, c1base),
		line(`w`, c1base),
		line(`e`, c1base),
		line(`r`, c1base),
		line(`t`, c1base),
		line(`y`, c1base),
		line(`u`, c1base),
		line(`i`, c1base),
		line(`o`, c1base),
		line(`9`, c3master),
		line(`9`, c3master),
		line(`9`, c3master),
		line(`9`, c3master),
		line(`9`, c3master),
		line(`9`, c3master),
		line(`9`, c3master),
		line(`9`, c3master),
		line(`p`, c1base),
		line(`a`, c1base),
		line(`s`, c1base),
		line(`1`, c2branch),
		line(`2`, c2branch),
		line(`3`, c2branch),
		line(`4`, c2branch),
		line(`5`, c2branch),
		line(`d`, c1base),
		line(`f`, c1base),
		line(`g`, c1base),
		line(`h`, c1base),
		line(`z`, c1base),
		line(`x`, c1base),
	)

	assertEqualFiles(t, f4merge, want)
}

// data source
// testdata/merge_multiparents
func TestMerge2Multiparents(t *testing.T) {
	diff1base := `diff --git a/a.go b/a.go
new file mode 100644
index 0000000..f2c18b2
--- /dev/null
+++ b/a.go
@@ -0,0 +1,8 @@
+q
+w
+e
+r
+t
+y
+u
+i`

	diff2a := `diff --git a/a.go b/a.go
index f2c18b2..5886731 100644
--- a/a.go
+++ b/a.go
@@ -1,3 +1,4 @@
+1
	q
	w
	e`

	diff3b := `diff --git a/a.go b/a.go
index f2c18b2..9102991 100644
--- a/a.go
+++ b/a.go
@@ -3,6 +3,7 @@ w
	e
	r
	t
+2
	y
	u
	i`

	diff4m := `diff --git a/a.go b/a.go
index f2c18b2..b702c0b 100644
--- a/a.go
+++ b/a.go
@@ -4,5 +4,5 @@ e
	r
	t
	y
-u
	i
+3`

	mergeDiff1 := `diff --git a/a.go b/a.go
index b702c0b..7570414 100644
--- a/a.go
+++ b/a.go
@@ -1,8 +1,10 @@
+1
 q
 w
 e
 r
 t
+2
 y
 i
 3`

	mergeDiff2 := `diff --git a/a.go b/a.go
index 5886731..7570414 100644
--- a/a.go
+++ b/a.go
@@ -4,6 +4,7 @@ w
 e
 r
 t
+2
 y
-u
 i
+3`

	mergeDiff3 := `diff --git a/a.go b/a.go
index 9102991..7570414 100644
--- a/a.go
+++ b/a.go
@@ -1,3 +1,4 @@
+1
 q
 w
 e
@@ -5,5 +6,5 @@ r
 t
 2
 y
-u
 i
+3`

	c1base := "c1base"
	c2a := "c2a"
	c3b := "c3b"
	c4m := "c4m"
	c5merge := "c5merge"

	f1base := Apply(Blame{}, tparse(diff1base), c1base, "")
	f2a := Apply(f1base, tparse(diff2a), c2a, "")
	f3b := Apply(f1base, tparse(diff3b), c3b, "")
	f4m := Apply(f1base, tparse(diff4m), c4m, "")
	f5merge := ApplyMerge(
		[]Blame{f4m, f2a, f3b},
		tparseDiffs(mergeDiff1, mergeDiff2, mergeDiff3),
		c5merge, "")

	want := file(c5merge,
		line(`1`, c2a),
		line(`q`, c1base),
		line(`w`, c1base),
		line(`e`, c1base),
		line(`r`, c1base),
		line(`t`, c1base),
		line(`2`, c3b),
		line(`y`, c1base),
		line(`i`, c1base),
		line(`3`, c4m),
	)

	assertEqualFiles(t, f5merge, want)
}
