package tests

import (
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/history3/process"
)

// When looking at merges using -m option it generated diff for each parent. But if the parent diff == 0 then it is omitted. Make sure we actually check which parent was used, instead of relying on order.
// Not doing it correctly does not lead to errors in this case.
// TODO: need are repro
func TestMergeMDiffNotShownForPrim(t *testing.T) {
	t.Skip("test is not correct WIP")
	test := NewTest(t, "merge_m_diff_not_shown_for_prim")
	got := test.Run()

	c1 := "f41bd930ef17b6d2f06eb22847bd78943464df65"
	c2 := "13dcd0e9ca3d4ed8fe62ec5ad46295eafe9a14ea"
	c3 := "4c99a9e681b9c699ee100a4311ba0a0e329ab834"
	c4 := "6b0276de4b0599f7caec246517ceeef2a58e6378"

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
				"c.txt": file(c2,
					line(`c`, c2),
				),
			},
		},
		{
			Commit: c3,
			Files: map[string]*incblame.Blame{
				"c.txt": file(c3,
					line(`c`, c3),
				),
			},
		},
		{
			Commit: c4,
			Files: map[string]*incblame.Blame{
				"c.txt": file(c4,
					line(`c`, c2),
					line(`d`, c4),
				),
			},
		},
	}
	assertResult(t, want, got)
}
