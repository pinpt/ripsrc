package tests

import (
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/history3/process"
)

func TestMergeBranchSelect(t *testing.T) {
	test := NewTest(t, "merge_branch_select")
	got := test.Run()

	c1 := "34bc624cce01e974960ff11aaaf7b1bcb1cac189"
	c2 := "b6f00a92a3eb5b5df5574a192853ac68b9d05280"
	c3 := "56527ebac87e58ac53f545b7c9737f29f254623a"
	c4 := "454010bb96cc39c4f41ccf0ba782539e949e388e"
	c5 := "274ce02011864284bc2a178cf210556cc1a8819b"

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
					line(`a+`, c2),
				),
			},
		},
		{
			Commit: c3,
			Files: map[string]*incblame.Blame{
				"b.txt": file(c3,
					line(`b+`, c3),
				),
			},
		},
		{
			Commit: c4,
			Files:  map[string]*incblame.Blame{},
		},
		{
			Commit: c5,
			Files: map[string]*incblame.Blame{
				"a.txt": file(c5,
					line(`a+`, c2),
					line(`1`, c5),
				),
				"b.txt": file(c5,
					line(`b+`, c3),
					line(`1`, c5),
				),
			},
		},
	}
	assertResult(t, want, got)
}
