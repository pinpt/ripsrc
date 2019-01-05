package tests

import (
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/history3/process"
)

// This is a test case for the following condition.
// If the file in repo was a binary at some point and then switched to text and was modified, then git log with patches does not contain the full file content.
func TestBinMergeToReg(t *testing.T) {
	test := NewTest(t, "bin_merge_to_reg")
	got := test.Run()

	c1 := "a9bef252ea7a3795be2c2207939633ac6f09865d"
	c2 := "0a42017d020ad6cb63b56b14e8570d3e98ddb7e3"
	c3 := "d3134fac036c11def147d5405c5f4b37216c42af"

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
				"basic.zip": incblame.BlameBinaryFile(c2),
			},
		},
		{
			Commit: c3,
			Files: map[string]*incblame.Blame{
				"basic.zip": incblame.BlameBinaryFile(c3),
				//"basic.zip": file(c3,
				//	line(`a`, c2),
				//	line(`A`, c3),
				//),
			},
		},
	}
	assertResult(t, want, got)
}
