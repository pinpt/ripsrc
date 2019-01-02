package tests

import (
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/history3/process"
)

func TestMergeNewWithChange(t *testing.T) {
	test := NewTest(t, "merge_new_with_change")
	got := test.Run()

	c1 := "1ca5f7fbd44d45078b33c0f32e42b6c77b05708e"
	c2 := "ea8e44f305cd844d81f2052892da2e0e56f3450d"
	c3 := "c33e8efe7f29dfde0045a872472bc24ad323a9b3"
	c4 := "7e6dbe576c19c4f549056547b1d3679c90597d8c"

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
				"a.txt": file(c3,
					line(`a`, c1),
					line(`a`, c3),
				),
			},
		},
		{
			Commit: c4,
			Files: map[string]*incblame.Blame{
				"b.txt": file(c4,
					line(`b`, c2),
					line(`b`, c4),
				),
			},
		},
	}
	assertResult(t, want, got)
}
