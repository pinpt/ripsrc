package tests

import (
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/history3/process"
)

func TestMergeBasic2(t *testing.T) {
	test := NewTest(t, "merge_basic2")
	got := test.Run()

	c1 := "8594b52d09d3df10fb392a38caa4805a756c8b26"
	c2 := "3c025938d44df68a47075364059ac9f293467826" // merge parent 2
	c3 := "de3309c8562d543650fe18a31fed15660e363fba" // merge parent 1
	c4 := "f9cd5f4d55d6ff11ed502c0a595c03617590a5a5"

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
				"c.txt": file(c3,
					line(`c`, c3),
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
