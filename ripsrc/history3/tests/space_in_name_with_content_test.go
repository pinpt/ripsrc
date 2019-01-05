package tests

import (
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/history3/process"
)

func TestSpaceInNameWithContent(t *testing.T) {
	test := NewTest(t, "space_in_name_with_content")
	got := test.Run()

	c1 := "0f801cc6cdb7a11e66e1a615940ed4e74bdbcff6"
	c2 := "66461ee9ddc0dbecbf4a681e0636d266c14a49c8"
	c3 := "b6d9384161ef363a76c82f1abed294184014dadd"

	want := []process.Result{
		{
			Commit: c1,
			Files: map[string]*incblame.Blame{
				"a a.txt": file(c1,
					line("a", c1),
				),
			},
		},
		{
			Commit: c2,
			Files: map[string]*incblame.Blame{
				"a a.txt": file(c2,
					line("a", c1),
					line("a", c2),
				),
			},
		},
		{
			Commit: c3,
			Files: map[string]*incblame.Blame{
				"a a.txt": file(c3,
					line("a", c1),
				),
			},
		},
	}
	assertResult(t, want, got)
}
