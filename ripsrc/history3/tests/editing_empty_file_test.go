package tests

import (
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/history3/process"
)

func TestEditingEmptyFile(t *testing.T) {
	test := NewTest(t, "editing_empty_file")
	got := test.Run()

	c1 := "394f33a2064f838495e752d5d895c81be476b546"
	c2 := "ede4adefcb25f43033622103f1bf2bf586b4b6f7"

	want := []process.Result{
		{
			Commit: c1,
			Files: map[string]*incblame.Blame{
				"a.txt": file(c1),
			},
		},
		{
			Commit: c2,
			Files: map[string]*incblame.Blame{
				"a.txt": file(c2,
					line(`a`, c2),
				),
			},
		},
	}
	assertResult(t, want, got)
}
