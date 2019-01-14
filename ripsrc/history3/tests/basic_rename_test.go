package tests

import (
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/history3/process"
)

func TestBasicRename(t *testing.T) {
	test := NewTest(t, "basic_rename")
	got := test.Run(nil)

	c1 := "f4ffbf5c5bfa147bd3792f4b3062802c8eaf65e2"
	c2 := "a6f2b499898c44372395878fdb527e028f63244b"

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
					line(`a`, c1),
				),
			},
		},
	}
	assertResult(t, want, got)
}
