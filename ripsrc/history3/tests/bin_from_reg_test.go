package tests

import (
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/history3/process"
)

func TestBinFromReg(t *testing.T) {
	test := NewTest(t, "bin_from_reg")
	got := test.Run()

	c1 := "4100c0083e5d67800ed2353d26ec40fa040c4dca"
	c2 := "e82bcbdee6ea5093f1cb04a80ffcd3177a399cdb"

	want := []process.Result{
		{
			Commit: c1,
			Files: map[string]*incblame.Blame{
				"basic.zip": file(c1,
					line(`a`, c1),
				),
			},
		},
		{
			Commit: c2,
			Files: map[string]*incblame.Blame{
				"basic.zip": incblame.BlameBinaryFile(c2),
			},
		},
	}
	assertResult(t, want, got)
}
