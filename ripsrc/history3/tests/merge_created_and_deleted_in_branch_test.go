package tests

import (
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/history3/process"
)

func TestMergeCreatedAndDeletedInBranch(t *testing.T) {
	test := NewTest(t, "merge_created_and_deleted_in_branch")
	got := test.Run()

	c1 := "bed98c3d7630be04af2ed51f81c6c01ded6a735f"
	c2 := "1e186471594426718f664f3cc252f72161427e31"
	c3 := "3f22c703895df8ff3127bc9bf281f648b71aad78"

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
				"b.txt": file(c2,
					line(`b`, c2),
				),
			},
		},
		{
			Commit: c3,
			Files: map[string]*incblame.Blame{
				"b.txt": file(c3),
			},
		},
	}
	assertResult(t, want, got)
}
