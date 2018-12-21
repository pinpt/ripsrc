package tests

import (
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/history3/process"
)

// TestGapBasic checks that blame works correctly for files not touched for a few commits.
func TestGapBasic(t *testing.T) {
	test := NewTest(t, "gap_basic")
	got := test.Run()

	c1 := "054bd3ca722948e5299bfda4d7c96f312f9c3b39"
	c2 := "06c4e4db2f222a663da1a5bb5afb1fa41e075f50"
	c3 := "ce8ed5092477e4730c2b76630675e1809deca103"
	c4 := "432c422013568e28107194044760f002b4082427"

	want := []process.Result{
		{
			Commit: c1,
			Files: map[string]*incblame.Blame{
				"a.txt": file(c1,
					line(`a`, c1),
				),
				"b.txt": file(c1,
					line(`b`, c1),
				),
			},
		},
		{
			Commit: c2,
			Files: map[string]*incblame.Blame{
				"a.txt": file(c2,
					line(`A`, c2),
				),
			},
		},
		{
			Commit: c3,
			Files: map[string]*incblame.Blame{
				"a.txt": file(c3,
					line(`AA`, c3),
				),
			},
		},
		{
			Commit: c4,
			Files: map[string]*incblame.Blame{
				"b.txt": file(c4,
					line(`B`, c4),
				),
			},
		},
	}
	assertResult(t, want, got)
}
