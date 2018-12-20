package tests

import (
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/history3/process"
)

func TestBasic(t *testing.T) {
	test := NewTest(t, "basic")
	got := test.Run()

	c1 := "b4dadc54e312e976694161c2ac59ab76feb0c40d"
	c2 := "69ba50fff990c169f80de96674919033a0a9b66d"

	want := []process.Result{
		{
			Commit: c1,
			Files: map[string]*incblame.Blame{
				"main.go": file(c1,
					line(`package main`, c1),
					line(``, c1),
					line(`import "github.com/pinpt/ripsrc/cmd"`, c1),
					line(``, c1),
					line(`func main() {`, c1),
					line(`	cmd.Execute()`, c1),
					line(`}`, c1),
					line(``, c1),
				),
			},
		},
		{
			Commit: c2,
			Files: map[string]*incblame.Blame{
				"main.go": file(c2,
					line(`package main`, c1),
					line(``, c1),
					line(`func main() {`, c1),
					line(`  // do nothing`, c2),
					line(`}`, c1),
					line(``, c1),
				),
			},
		},
	}
	assertResult(t, want, got)
}
