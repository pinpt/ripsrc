package tests

import (
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/history3/process"
)

func TestMergeSameChanges(t *testing.T) {
	test := NewTest(t, "merge_same_changes")
	got := test.Run()

	c1 := "7b2426009c16e103bed4aaf0ca732f0c3f376026"
	c2 := "6aac6cbfcdae43f0ebd2351b59794e37e6bd6364"
	c3 := "2531fb0b42d18f1dd97e6b1a303f05bf05aef83e"
	c4 := "80775358d2088bb31deedf7d18173ad916b025d3"

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
					line(`b`, c2),
				),
			},
		},
		{
			Commit: c3,
			Files: map[string]*incblame.Blame{
				"a.txt": file(c3,
					line(`b`, c3),
				),
			},
		},
		{
			Commit: c4,
			Files:  map[string]*incblame.Blame{},
		},
	}
	assertResult(t, want, got)
}
