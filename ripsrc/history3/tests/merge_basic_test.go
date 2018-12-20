package tests

import (
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/history3/process"
)

func TestMergeBasic(t *testing.T) {
	test := NewTest(t, "merge_basic")
	got := test.Run()

	c1 := "cb78f81991af4120b649c5e2ae18cceba598220a"
	c2 := "a08d204ee5913986294000e1280e7ad3484098e3"
	c3 := "3219b85f18fad2aa802344a2275bd8288916f4ee"
	c4 := "49dd6946d595ae6cd51fb228f14c799b749ea3a4"

	want := []process.Result{
		{
			Commit: c1,
			Files: map[string]*incblame.Blame{
				"main.go": file(c1,
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
				"main.go": file(c2,
					line(`package main`, c1),
					line(``, c1),
					line(`func main(){`, c1),
					line(`// A`, c2),
					line(`}`, c1),
				),
			},
		},
		{
			Commit: c3,
			Files: map[string]*incblame.Blame{
				"main.go": file(c3,
					line(`package main`, c1),
					line(``, c1),
					line(`func main(){`, c1),
					line(`// M`, c3),
					line(`}`, c1),
				),
			},
		},
		{
			Commit: c4,
			Files: map[string]*incblame.Blame{
				"main.go": file(c4,
					line(`package main`, c1),
					line(``, c1),
					line(`func main(){`, c1),
					line(`// M`, c3),
					line(`// A`, c2),
					line(`}`, c1),
				),
			},
		},
	}
	assertResult(t, want, got)
}
