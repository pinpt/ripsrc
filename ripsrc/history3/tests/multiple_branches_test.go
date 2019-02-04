package tests

import (
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/history3/process"
)

func TestMultipleBranches1(t *testing.T) {
	test := NewTest(t, "multiple_branches")
	got := test.Run(&process.Opts{AllBranches: true})

	c1 := "bdf8c8cfa9c027e58f1aea5c532ba0e9ef74bc4c"
	c2 := "d3a93f475772c90918ebc34e144e1c3554163a9f"
	c3 := "7c6eba56ba8616ee903f2394553c022d6d3046bf"
	c4 := "3f18a2ea07832a18d0645df2aa666b339cee1a06"

	want := []process.Result{
		{
			Commit: c1,
			Files: map[string]*incblame.Blame{
				"a.txt": file(c1,
					line(`a`, c1),
				),
			},
		},
		{
			Commit: c2,
			Files: map[string]*incblame.Blame{
				"a.txt": file(c2,
					line(`a`, c1),
					line(`b`, c2),
				),
			},
		},
		{
			Commit: c3,
			Files: map[string]*incblame.Blame{
				"a.txt": file(c3,
					line(`aa`, c3),
				),
			},
		},
		{
			Commit: c4,
			Files: map[string]*incblame.Blame{
				"a.txt": file(c4,
					line(`a`, c1),
					line(`c`, c4),
				),
			},
		},
	}
	assertResult(t, want, got)
}

func TestMultipleBranchesDisabled(t *testing.T) {
	test := NewTest(t, "multiple_branches_disabled")
	got := test.Run(nil)

	c1 := "bba6ce31b58bd8b864b0c3eb4fb8856b2dcc0297"

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
