package e2etests

import (
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/branches2"
)

func TestBranchesBasic1(t *testing.T) {
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

func TestBranchesIncludeDefault1(t *testing.T) {
	test := NewTest(t, "basic1", &branches2.Opts{
		IncludeDefaultBranch: true,
	})
	got := test.Run()

	c1 := "33e223d1fd8393dc98596727d370e51e7b3b7fba"
	c2 := "9b39087654af70197f68d0b3d196a4a20d987cd6"

	want := []branches2.Branch{
		{
			Name:      "master",
			IsDefault: true,
			Commits:   []string{c1},
		},
		{
			IsMerged:            false,
			Name:                "a",
			Commits:             []string{c2},
			BranchedFromCommits: []string{c1},
		},
	}

	assertResult(t, want, got)
}

func TestBranchesMerged1(t *testing.T) {
	test := NewTest(t, "merged1", nil)
	got := test.Run()

	c1 := "56a3e281518a6e56de3693ec65348f472275187e"
	c2 := "ac22dfb85417e3d256baeb62fc8b51e33b61ac27"
	c3 := "5ac62691bf584ecee16eb832a4c444aab74d2d27"

	want := []branches2.Branch{
		{
			IsMerged:            true,
			MergeCommit:         c3,
			Name:                "a",
			Commits:             []string{c2},
			BranchedFromCommits: []string{c1},
		},
	}

	assertResult(t, want, got)
}
