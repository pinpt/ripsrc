package e2etests

import (
	"testing"

	"github.com/pinpt/ripsrc/ripsrc"
)

func TestBasic1(t *testing.T) {
	test := NewTest(t, "basic1", nil)
	got := test.Run()

	c1 := "33e223d1fd8393dc98596727d370e51e7b3b7fba"
	c2 := "9b39087654af70197f68d0b3d196a4a20d987cd6"

	want := []ripsrc.Branch{
		{
			ID:                  "f45c6fa79ef6f5641148aef7f6c2ea71dd74bc207011e3947817d4a8ef4b0ff8",
			IsMerged:            false,
			Name:                "a",
			Commits:             []string{c2},
			BranchedFromCommits: []string{c1},
		},
	}

	assertResult(t, want, got)
}
