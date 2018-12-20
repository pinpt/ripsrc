package tests

import (
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/history3/process"
)

func TestDeletedFiles(t *testing.T) {
	test := NewTest(t, "deleted_files")
	got := test.Run()

	c1 := "624f3a74bf727e365cfbd090b9b993ddded0e1ea"
	c2 := "9c7629df59b283bdec8b9705cb17c822652f6fae"

	want := []process.Result{
		{
			Commit: c1,
			Files: map[string]*incblame.Blame{
				"a.go": file(c1,
					line(`package main`, c1),
					line(``, c1),
					line(`func main(){`, c1),
					line(`}`, c1),
				),
			},
		},
		{
			Commit: c2,
			Files: map[string]*incblame.Blame{
				"a.go": file(c2),
			},
		},
	}
	assertResult(t, want, got)

}
