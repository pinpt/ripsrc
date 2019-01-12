package tests

import (
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/history3/process"
)

// Check that processing ignores any settings set in use .gitattributes file or inside the repo.
func TestCustomGitAttr1(t *testing.T) {
	t.Skip("not supported, need correct global settings and in repo settings")
	test := NewTest(t, "custom_git_attr")
	got := test.Run()

	c1 := "74866c1398bbe2498c1f136ae1ad78c6376d3df7"
	c2 := "43f304b0ce38f8b42a61a1552207d6f78f14eb52"

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
	}
	assertResult(t, want, got)
}
