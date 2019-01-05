package tests

import (
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/history3/process"
)

func TestRenameWithSpaces(t *testing.T) {
	test := NewTest(t, "rename_with_spaces")
	got := test.Run()

	c1 := "550a9f28d5f740e599c2a8c213d9370be528936a"
	c2 := "5696bd8f1ee3e5979a5d36ae17039a7e24084abb"

	want := []process.Result{
		{
			Commit: c1,
			Files: map[string]*incblame.Blame{
				"a a.txt": file(c1),
			},
		},
		{
			Commit: c2,
			Files: map[string]*incblame.Blame{
				"b b.txt": file(c2),
			},
		},
	}
	assertResult(t, want, got)
}
