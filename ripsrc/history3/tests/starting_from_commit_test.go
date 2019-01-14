package tests

import (
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/history3/process"
)

func TestStartFromCommit1(t *testing.T) {
	test := NewTest(t, "starting_from_commit")
	got := test.Run(&process.Opts{CommitFromIncl: "2124437117d4fe0ac185993e103627bc1d3d848c"})

	c1 := "10d919f0ecc372cefb7603f093b5afc473d8d61a"
	c2 := "f108c129fb0f24b15a3f9ca2b6c1d457089e7f2d"
	c3 := "2124437117d4fe0ac185993e103627bc1d3d848c"
	c4 := "d1780d62ab1b3dfe72c0f937d93fdb65ad4d6958"

	want := []process.Result{
		{
			Commit: c3,
			Files: map[string]*incblame.Blame{
				"a.txt": file(c3,
					line(`a`, c1),
					line(`b`, c2),
					line(`c`, c3),
				),
				"b.txt": file(c3,
					line(`b`, c2),
					line(`c`, c3),
				),
			},
		},
		{
			Commit: c4,
			Files: map[string]*incblame.Blame{
				"a.txt": file(c4,
					line(`a`, c1),
					line(`b`, c2),
					line(`c`, c3),
					line(`d`, c4),
				),
				"b.txt": file(c4,
					line(`b`, c2),
					line(`c`, c3),
					line(`d`, c4),
				),
			},
		},
	}
	assertResult(t, want, got)
}
