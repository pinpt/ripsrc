package tests

import (
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/history3/process"
)

func TestBasicAddFile(t *testing.T) {
	test := NewTest(t, "basic_add_file")
	got := test.Run()

	c1 := "dfc1c9aede4d1e85843da10950bc13015c133a90"
	c2 := "8758ef8c99c61d0ad117b295c377b144dd1ef3be"

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
	}
	assertResult(t, want, got)
}
