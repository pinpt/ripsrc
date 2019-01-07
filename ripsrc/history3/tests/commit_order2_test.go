package tests

import (
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/history3/process"
)

// This test fails if --date-order is not passed in process. The child commit is shown before parent in this case.
// Commit 2 was commited using date in the future.
func TestCommitOrder2(t *testing.T) {
	test := NewTest(t, "commit_order2")
	got := test.Run()

	c1 := "d1ae279323d0c7bc9fe9ee101edeccdf9d992412"
	c2 := "781215b9c139709e2d21130ddeb2e2ff8c2bbf9a"
	c3 := "a4e51b6d862f44e3674df0e6279eb60dd544d2f5"

	want := []process.Result{
		{
			Commit: c1,
			Files: map[string]*incblame.Blame{
				"a.txt": file(c1,
					line(`a`, c1),
					line(`a`, c1),
				),
			},
		},
		{
			Commit: c2,
			Files: map[string]*incblame.Blame{
				"a.txt": file(c2,
					line(`a`, c1),
				),
			},
		},
		{
			Commit: c3,
			Files:  map[string]*incblame.Blame{},
		},
	}
	assertResult(t, want, got)
}
