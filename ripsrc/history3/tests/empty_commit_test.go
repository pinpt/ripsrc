package tests

import (
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/history3/process"
)

func TestEmptyCommit1(t *testing.T) {
	test := NewTest(t, "empty_commit")
	got := test.Run(nil)

	c1 := "f95497fbd222cdd6a4fe1e5531b957515dff209f"
	c2 := "fb37735ac46ffa1912b3399b2fdf41853c588547"

	want := []process.Result{
		{
			Commit: c1,
			Files:  map[string]*incblame.Blame{},
		},
		{
			Commit: c2,
			Files: map[string]*incblame.Blame{
				"a.txt": file(c2,
					line(`a`, c2),
				),
			},
		},
	}
	assertResult(t, want, got)
}
