package incblame

import (
	"testing"
)

const basicDiff1 = `diff --git a/main.go b/main.go
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

const basicDiff2 = `diff --git a/main.go b/main.go
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
	}
`

func TestApplyBasic1(t *testing.T) {
	diff := Parse([]byte(basicDiff1))
	c1 := "c1"
	f2 := Apply(Blame{}, diff, c1)

	want := Blame{
		Lines: []Line{
			tl(`package main`, c1),
			tl(``, c1),
			tl(`import "github.com/pinpt/ripsrc/cmd"`, c1),
			tl(``, c1),
			tl(`func main() {`, c1),
			tl(`	cmd.Execute()`, c1),
			tl(`}`, c1),
			tl(``, c1),
		},
	}

	assertEqualFiles(t, f2, want)
}

func TestApplyBasic2Regular(t *testing.T) {
	c1 := "c1"
	c2 := "c2"

	diff := Parse([]byte(basicDiff1))
	f := Apply(Blame{}, diff, c1)
	diff = Parse([]byte(basicDiff2))
	f = applyOneParent(f, diff, c2)

	want := Blame{
		Lines: []Line{
			tl(`package main`, c1),
			tl(``, c1),
			tl(`func main() {`, c1),
			tl(`  // do nothing`, c2),
			tl(`}`, c1),
			tl(``, c1),
		},
	}

	assertEqualFiles(t, f, want)
}
