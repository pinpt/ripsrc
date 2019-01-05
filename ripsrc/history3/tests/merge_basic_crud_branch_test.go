package tests

import (
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/history3/process"
)

func TestMergeBasicCrudBranch(t *testing.T) {
	test := NewTest(t, "merge_basic_crud_branch")
	got := test.Run()

	c1 := "b5c42f70c2bf950cb8c5cbcaff87027b5e30fe67"
	c2 := "142b866db5e367b013b5a8acb5f7a959eb1b7706"
	c3 := "6f4360c2b0565331128c689358e79f192af5effb"

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
					line(`a`, c1),
					line(`a`, c2),
				),
				"b.txt": file(c2),
				"c.txt": file(c2,
					line(`c`, c2),
				),
			},
		},
		{
			Commit: c3,
			Files: map[string]*incblame.Blame{
				// only showing deletes and files changed in merge comparent to at least one parent
				"b.txt": file(c3),
			},
		},
	}
	assertResult(t, want, got)
}
