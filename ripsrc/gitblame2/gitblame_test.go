package gitblame2

import (
	"testing"
)

func TestBasic(t *testing.T) {
	test := NewTest(t, "basic")
	hash := "69ba50fff990c169f80de96674919033a0a9b66d"
	got, err := test.Run(hash, "main.go")
	if err != nil {
		t.Fatal(err)
	}
	c1 := "b4dadc54e312e976694161c2ac59ab76feb0c40d"
	c2 := "69ba50fff990c169f80de96674919033a0a9b66d"
	want := []Line{
		ml("package main", c1),
		ml("", c1),
		ml("func main() {", c1),
		ml("  // do nothing", c2),
		ml("}", c1),
		ml("", c1),
	}
	assertEqualLines(t, Result{Lines: want}, got)
}

func ml(content string, commitHash string) Line {
	return Line{Content: content, CommitHash: commitHash}
}

func assertEqualLines(t *testing.T, wantRes, gotRes Result) {
	want := wantRes.Lines
	got := gotRes.Lines

	t.Helper()
	if len(want) != len(got) {
		t.Fatalf("got %v lines", len(got))
	}
	for i := range want {
		if want[i] != got[i] {
			t.Errorf("line %v wanted %#+v got %#+v", i, want[i], got[i])
		}
	}
}

func TestNonLatestCommit(t *testing.T) {
	test := NewTest(t, "basic")
	hash := "b4dadc54e312e976694161c2ac59ab76feb0c40d"
	got, err := test.Run(hash, "main.go")
	if err != nil {
		t.Fatal(err)
	}
	c1 := "b4dadc54e312e976694161c2ac59ab76feb0c40d"
	want := []Line{
		ml(`package main`, c1),
		ml(``, c1),
		ml(`import "github.com/pinpt/ripsrc/cmd"`, c1),
		ml(``, c1),
		ml(`func main() {`, c1),
		ml("	cmd.Execute()", c1),
		ml("}", c1),
		ml("", c1),
	}
	assertEqualLines(t, Result{Lines: want}, got)
}
func TestFormerBinData(t *testing.T) {
	test := NewTest(t, "editing_former_bin_file")
	hash := "831589d24aba19a83aa080194ba335a67da0413e"
	got, err := test.Run(hash, "main.go")
	if err != nil {
		t.Fatal(err)
	}
	c1 := "94909fcc06b6a65bf865f94fbc22e5ace8fbbbd6"
	want := []Line{
		ml("package main", c1),
		ml("", c1),
		ml("func main(){", c1),
		ml("}", c1),
	}
	assertEqualLines(t, Result{Lines: want}, got)
}
