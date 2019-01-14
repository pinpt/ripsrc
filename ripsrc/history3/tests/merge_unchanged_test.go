package tests

import (
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/history3/process"
)

func TestMergeUnchanged(t *testing.T) {
	test := NewTest(t, "merge_unchanged")
	got := test.Run(nil)

	c1 := "c17463b8e0d5890d2a6fc9e20507ab7e985e2007"
	c2 := "34fad1eecfd082fb0845d148696e2993f888e4d2"
	c3 := "9ac941fa5ae894a36cb007d961d0047de670db70"
	c4 := "86d1f67e29e9c5d4de12ee5e21a7694930c9c543"

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
			Files:  map[string]*incblame.Blame{},
		},
		{
			Commit: c4,
			Files: map[string]*incblame.Blame{
				"a.txt": file(c4,
					line(`a`, c1),
					line(`a`, c4),
				),
			},
		},
	}
	assertResult(t, want, got)
}
