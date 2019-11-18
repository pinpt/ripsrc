package e2etests

import (
	"testing"
	"time"

	"github.com/pinpt/ripsrc/ripsrc/branchmeta"
)

func TestBranchesBasic1(t *testing.T) {
	test := NewTest(t, "basic1", nil)
	got := test.Run()

	want := []branchmeta.BranchWithCommitTime{
		{
			Name:                "a",
			Commit:              "9b39087654af70197f68d0b3d196a4a20d987cd6",
			CommitCommitterTime: parseTime("2019-02-07T20:17:34+01:00"),
		},
	}

	assertResult(t, want, got)
}

func TestBranchesIncludeDefault(t *testing.T) {
	test := NewTest(t, "basic1", &branchmeta.Opts{
		IncludeDefault: true,
	})
	got := test.Run()

	want := []branchmeta.BranchWithCommitTime{
		{
			Name:                "a",
			Commit:              "9b39087654af70197f68d0b3d196a4a20d987cd6",
			CommitCommitterTime: parseTime("2019-02-07T20:17:34+01:00"),
		},
		{
			Name:                "master",
			Commit:              "33e223d1fd8393dc98596727d370e51e7b3b7fba",
			CommitCommitterTime: parseTime("2019-02-07T20:17:18+01:00"),
		},
	}
	assertResult(t, want, got)
}

func parseTime(s string) time.Time {
	res, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return res
}
