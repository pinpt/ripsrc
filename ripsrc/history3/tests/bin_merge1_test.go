package tests

import (
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/history3/process"
)

func TestBinMerge1(t *testing.T) {
	test := NewTest(t, "bin_merge1")
	got := test.Run()

	c1 := "0fdd9a3d2d11961f864fcbf60fbade8d2046f804"
	c2 := "bb9074ba11ee60b7376db008d7d85409aedf812f"
	c3 := "d32e22a7489865f535a450c426d2c905b49dedba"

	want := []process.Result{
		{
			Commit: c1,
			Files: map[string]*incblame.Blame{
				"f1.zip": incblame.BlameBinaryFile(c1),
				"f2.zip": incblame.BlameBinaryFile(c1),
				"f3.zip": file(c1,
					line(`a`, c1),
				),
			},
		},
		{
			Commit: c2,
			Files: map[string]*incblame.Blame{
				"f1.zip": incblame.BlameBinaryFile(c2),
				"f2.zip": incblame.BlameBinaryFile(c2),
				"f3.zip": incblame.BlameBinaryFile(c2),
			},
		},
		{
			Commit: c3,
			Files: map[string]*incblame.Blame{
				"f1.zip": incblame.BlameBinaryFile(c3),
				"f2.zip": incblame.BlameBinaryFile(c3),
				"f3.zip": incblame.BlameBinaryFile(c3),
			},
		},
	}
	assertResult(t, want, got)
}
