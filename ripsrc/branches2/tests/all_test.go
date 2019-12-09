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
			HeadSHA:             c2,
			Commits:             []string{c2},
			BranchedFromCommits: []string{c1},
			BehindDefaultCount:  0,
			AheadDefaultCount:   1,
			FirstCommit:         c2,
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
			Name:        "master",
			HeadSHA:     c1,
			IsDefault:   true,
			Commits:     []string{c1},
			FirstCommit: c1,
		},
		{
			IsMerged:            false,
			Name:                "a",
			HeadSHA:             c2,
			Commits:             []string{c2},
			BranchedFromCommits: []string{c1},
			BehindDefaultCount:  0,
			AheadDefaultCount:   1,
			FirstCommit:         c2,
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
			HeadSHA:             c2,
			MergeCommit:         c3,
			Name:                "a",
			Commits:             []string{c2},
			BranchedFromCommits: []string{c1},
			BehindDefaultCount:  0,
			AheadDefaultCount:   1,
			FirstCommit:         c2,
		},
	}

	assertResult(t, want, got)
}

func TestBranchesBehindMaster1(t *testing.T) {
	test := NewTest(t, "behindmaster1", nil)
	got := test.Run()

	c1 := "33e223d1fd8393dc98596727d370e51e7b3b7fba"
	c2 := "9b39087654af70197f68d0b3d196a4a20d987cd6"

	want := []branches2.Branch{
		{
			IsMerged:            false,
			Name:                "a",
			HeadSHA:             c2,
			Commits:             []string{c2},
			BranchedFromCommits: []string{c1},
			BehindDefaultCount:  2,
			AheadDefaultCount:   1,
			FirstCommit:         c2,
		},
	}

	assertResult(t, want, got)
}

// Check ordering of commits of master branch
func TestBranchesMasterOrder(t *testing.T) {
	test := NewTest(t, "master_order", &branches2.Opts{
		IncludeDefaultBranch: true,
	})
	got := test.Run()

	c1 := "88c671630d8c0a8d36e5c9037d461910297ac59b"
	c2 := "5336c030caa3aaaae9d19b86a883e42040f5f539"
	c3 := "7276ced6e6b938ac550949cb7d1d1446d286da97"
	c4 := "871024b23514ed4023d6ec7743ff108905722c09"

	want := []branches2.Branch{
		{
			Name:      "master",
			HeadSHA:   c4,
			IsDefault: true,
			// NOTE: order of commits in master branch is not based on date or topology. First we get all commits for master, then new commits per branch.
			// For this reason we have separate FirstCommit field
			Commits:     []string{c2, c1, c3, c4},
			FirstCommit: c1,
		},
		{
			IsMerged:            true,
			MergeCommit:         c4,
			Name:                "b1",
			HeadSHA:             c3,
			Commits:             []string{c3},
			BranchedFromCommits: []string{c1},
			AheadDefaultCount:   1,
			FirstCommit:         c3,
		},
	}

	assertResult(t, want, got)
}

func TestPullRequestsBasic1(t *testing.T) {

	test := NewTest(t, "basic1", nil)
	c1 := "33e223d1fd8393dc98596727d370e51e7b3b7fba"
	c2 := "9b39087654af70197f68d0b3d196a4a20d987cd6"

	test.opts = &branches2.Opts{}
	test.opts.PullRequestSHAs = []string{c2}
	test.opts.PullRequestsOnly = true

	got := test.Run()

	want := []branches2.Branch{
		{
			IsPullRequest:       true,
			HeadSHA:             c2,
			Commits:             []string{c2},
			BranchedFromCommits: []string{c1},
			BehindDefaultCount:  0,
			AheadDefaultCount:   1,
			FirstCommit:         c2,
		},
	}

	assertResult(t, want, got)
}

func TestPullRequestsDuplicates1(t *testing.T) {

	test := NewTest(t, "basic1", nil)
	c1 := "33e223d1fd8393dc98596727d370e51e7b3b7fba"
	c2 := "9b39087654af70197f68d0b3d196a4a20d987cd6"

	test.opts = &branches2.Opts{}
	test.opts.PullRequestSHAs = []string{c2, c2}
	test.opts.PullRequestsOnly = true

	got := test.Run()

	want := []branches2.Branch{
		{
			IsPullRequest:       true,
			HeadSHA:             c2,
			Commits:             []string{c2},
			BranchedFromCommits: []string{c1},
			BehindDefaultCount:  0,
			AheadDefaultCount:   1,
			FirstCommit:         c2,
		},
	}

	assertResult(t, want, got)
}

func TestPullRequestsNotExisting1(t *testing.T) {

	test := NewTest(t, "basic1", nil)

	test.opts = &branches2.Opts{}
	test.opts.PullRequestSHAs = []string{"xxx"}
	test.opts.PullRequestsOnly = true

	got := test.Run()

	want := []branches2.Branch{}

	assertResult(t, want, got)
}
