package incblame

import (
	"strconv"
	"testing"
)

func makeLongDiff(c int) Diff {
	diffPrefix := `diff --git a/main.go b/main.go
new file mode 100644
index 0000000..43f9419
--- /dev/null
+++ b/main.go
@@ -0,0 +1,` + strconv.Itoa(c) + ` @@` + "\n"

	diffBytes := []byte{}
	diffBytes = append(diffBytes, diffPrefix...)
	for i := 0; i < c; i++ {
		diffBytes = append(diffBytes, "+a"...)
		diffBytes = append(diffBytes, '\n')
	}

	return Parse(diffBytes)
}

func TestApplyGeneratedFile(t *testing.T) {
	const lines = 3
	diff := makeLongDiff(lines)

	f := Apply(Blame{}, diff, "c1", "")
	want := Blame{}
	for i := 0; i < lines; i++ {
		want.Lines = append(want.Lines, tl("a", "c1"))
	}
	assertEqualFiles(t, f, want)
}

func BenchmarkApplyNewLargeFile(b *testing.B) {
	const lines = 10000
	diff := makeLongDiff(lines)

	for i := 0; i < b.N; i++ {
		Apply(Blame{}, diff, "c1", "")
	}
}
