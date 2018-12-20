package incblame

import "testing"

func TestMerge1(t *testing.T) {
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

	diff4 := `diff --cc main.go
index 1dbddb0,4cd4b38..904d55b
--- a/main.go
+++ b/main.go
@@@ -1,5 -1,5 +1,6 @@@
  package main
  
  func main(){
	+// M
+ // A
  }`

	c1base := "c1base"
	c2branch := "c2branch"
	c3master := "c3master"
	c4merge := "c4merge"

	f1base := Apply(nil, tparse(diff1), c1base)
	f2branch := applyOneParent(f1base, tparse(diff2), c2branch)
	f3master := applyOneParent(f1base, tparse(diff3), c3master)
	f4merge := Apply([]Blame{f3master, f2branch}, tparse(diff4), c4merge)

	want := Blame{
		Lines: []Line{
			tl(`package main`, c1base),
			tl(``, c1base),
			tl(`func main(){`, c1base),
			tl(`// M`, c3master),
			tl(`// A`, c2branch),
			tl(`}`, c1base),
		},
	}

	assertEqualFiles(t, f4merge, want)
}

func TestMerge2(t *testing.T) {
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

	diff4 := `diff --combined a.go
index 0bb73a0,7a73855..e5f72e3
--- a/a.go
+++ b/a.go
@@@ -1,3 -1,8 +1,8 @@@
+ 1
+ 2
+ 3
+ 4
+ 5
  q
  w
  e
@@@ -7,21 -12,17 +12,25 @@@
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
+ 1
+ 2
+ 3
+ 4
+ 5
  d
  f
  g
  h
- j
- k
- l
+ z
+ x`

	c1base := "c1base"
	c2branch := "c2branch"
	c3master := "c3master"
	c4merge := "c4merge"

	f1base := Apply(nil, tparse(diff1), c1base)
	f2branch := applyOneParent(f1base, tparse(diff2), c2branch)
	f3master := applyOneParent(f1base, tparse(diff3), c3master)
	f4merge := Apply([]Blame{f3master, f2branch}, tparse(diff4), c4merge)

	want := Blame{
		Lines: []Line{
			tl(`1`, c2branch),
			tl(`2`, c2branch),
			tl(`3`, c2branch),
			tl(`4`, c2branch),
			tl(`5`, c2branch),
			tl(`q`, c1base),
			tl(`w`, c1base),
			tl(`e`, c1base),
			tl(`r`, c1base),
			tl(`t`, c1base),
			tl(`y`, c1base),
			tl(`u`, c1base),
			tl(`i`, c1base),
			tl(`o`, c1base),
			tl(`9`, c3master),
			tl(`9`, c3master),
			tl(`9`, c3master),
			tl(`9`, c3master),
			tl(`9`, c3master),
			tl(`9`, c3master),
			tl(`9`, c3master),
			tl(`9`, c3master),
			tl(`p`, c1base),
			tl(`a`, c1base),
			tl(`s`, c1base),
			tl(`1`, c2branch),
			tl(`2`, c2branch),
			tl(`3`, c2branch),
			tl(`4`, c2branch),
			tl(`5`, c2branch),
			tl(`d`, c1base),
			tl(`f`, c1base),
			tl(`g`, c1base),
			tl(`h`, c1base),
			tl(`z`, c1base),
			tl(`x`, c1base),
		},
	}

	assertEqualFiles(t, f4merge, want)
}

func TestMerge3MultipleParents(t *testing.T) {
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

	diff4m := `
diff --git a/a.go b/a.go
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

	diff5merge := `
diff --combined a.go
index b702c0b,5886731,9102991..7570414
--- a/a.go
+++ b/a.go
@@@@ -1,8 -1,9 -1,9 +1,10 @@@@
+ +1
   q
   w
   e
   r
   t
++ 2
   y
	--u
   i
	++3`

	c1base := "c1base"
	c2a := "c2a"
	c3b := "c3b"
	c4m := "c4m"
	c5merge := "c5merge"

	f1base := Apply(nil, tparse(diff1base), c1base)
	f2a := applyOneParent(f1base, tparse(diff2a), c2a)
	f3b := applyOneParent(f1base, tparse(diff3b), c3b)
	f4m := applyOneParent(f1base, tparse(diff4m), c4m)
	f5merge := Apply([]Blame{f4m, f2a, f3b}, tparse(diff5merge), c5merge)

	want := Blame{
		Lines: []Line{
			tl(`1`, c2a),
			tl(`q`, c1base),
			tl(`w`, c1base),
			tl(`e`, c1base),
			tl(`r`, c1base),
			tl(`t`, c1base),
			tl(`2`, c3b),
			tl(`y`, c1base),
			tl(`i`, c1base),
			tl(`3`, c4m),
		},
	}

	assertEqualFiles(t, f5merge, want)
}
