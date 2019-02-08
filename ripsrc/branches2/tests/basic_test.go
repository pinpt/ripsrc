package e2etests

import (
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/branches2"
)

func TestBasic1(t *testing.T) {
	test := NewTest(t, "basic1", nil)
	got := test.Run()

	c1 := "33e223d1fd8393dc98596727d370e51e7b3b7fba"
	c2 := "9b39087654af70197f68d0b3d196a4a20d987cd6"

	want := []branches2.Branch{
		{
			IsMerged:            false,
			Name:                "a",
			Commits:             []string{c2},
			BranchedFromCommits: []string{c1},
		},
	}

	assertResult(t, want, got)
}
