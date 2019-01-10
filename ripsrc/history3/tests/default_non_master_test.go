package tests

import (
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/history3/process"
)

// Checks that it works when repos is checked out on any branch.
// It should use that branch. Needed to support repos, which use
// a different branch instead of master.
func TestDefaultNonMaster(t *testing.T) {
	test := NewTest(t, "default_non_master")
	got := test.Run()

	c1 := "6342b6a4efeb897f54d60bcca426af843ac85c16"

	want := []process.Result{
		{
			Commit: c1,
			Files: map[string]*incblame.Blame{
				"a.txt": file(c1,
					line(`a`, c1),
				),
			},
		},
	}
	assertResult(t, want, got)
}
