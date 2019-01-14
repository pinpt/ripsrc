package tests

import (
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/history3/process"
)

func TestBinCrud(t *testing.T) {
	test := NewTest(t, "bin_crud")
	got := test.Run(nil)

	c1 := "2928e513831e71431d6b8003f38873a69c738a57"
	c2 := "3a92708066c7e6994c3f4184071a33c120dee75d"
	c3 := "b06f8cbac16a5f5a07560e56a59fd8088b815fa1"

	want := []process.Result{
		{
			Commit: c1,
			Files: map[string]*incblame.Blame{
				"basic.zip": incblame.BlameBinaryFile(c1),
			},
		},
		{
			Commit: c2,
			Files: map[string]*incblame.Blame{
				"n2.zip": incblame.BlameBinaryFile(c1),
			},
		},
		{
			Commit: c3,
			Files: map[string]*incblame.Blame{
				"n2.zip": incblame.BlameBinaryFile(c3),
			},
		},
	}
	assertResult(t, want, got)
}
