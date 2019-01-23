package tests

import (
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/history3/process"
)

// This is a test case for the following condition.
// If the file in repo was a binary at some point and then switched to text and was modified, then git log with patches does not contain the full file content.
func TestBinToReg(t *testing.T) {
	test := NewTest(t, "bin_to_reg")
	got := test.Run(nil)

	c1 := "906dc31df31485790f2c63f94cf13404897ab0f3"
	c2 := "cb569181179ab53951f30cf9cd447f2f615b2439"
	c3 := "72cf857547cd7ca00f8389f994fa0b4f6da96713"

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
				"basic.zip": file(c3,
					line(`Regular file`, c2),
					line(`Add`, c3),
				),
			},
		},
	}
	assertResult(t, want, got)
}
