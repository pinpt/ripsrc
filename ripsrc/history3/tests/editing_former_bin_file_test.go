package tests

import (
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/history3/process"
)

// This is a test case for the following condition.
// If the file in repo was a binary at some point and then switched to text and was modified, then git log with patches does not contain the full file content. There are 2 options to fix this, either we ignore all files that at some point in history were binary or retrieve the full file content for these cases separately without using log and patches.
func TestEditingFormerBinFile(t *testing.T) {
	test := NewTest(t, "editing_former_bin_file")
	got := test.Run()

	c1 := "94909fcc06b6a65bf865f94fbc22e5ace8fbbbd6"
	c2 := "199a1819583fc0da098f0d8328acbf43d35f3541"
	c3 := "831589d24aba19a83aa080194ba335a67da0413e"

	want := []process.Result{
		{
			Commit: c1,
			Files: map[string]*incblame.Blame{
				"main.go": incblame.BlameBinaryFile(c1),
			},
		},
		{
			Commit: c2,
			Files: map[string]*incblame.Blame{
				"main.go": incblame.BlameBinaryFile(c2),
			},
		},
		{
			Commit: c3,
			Files: map[string]*incblame.Blame{
				"main.go": file(c3,
					line(`package main`, c1),
					line(``, c1),
					line(`func main(){`, c1),
					line(`}`, c1),
				),
			},
		},
	}

	assertResult(t, want, got)
}
