package diff

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

	f1base := applySingleParent(NewNilFile(), tparse(diff1), c1base)
	f2branch := applySingleParent(f1base, tparse(diff2), c2branch)
	f3master := applySingleParent(f1base, tparse(diff3), c3master)
	f4merge := ApplyMerge([]File{f3master, f2branch}, tparse(diff4), c4merge)

	want := File{
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
