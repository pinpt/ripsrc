package e2etests

import (
	"context"
	"reflect"
	"testing"

	"github.com/pinpt/ripsrc/ripsrc"
	"github.com/pinpt/ripsrc/ripsrc/pkg/testutil"
)

type Test struct {
	t        *testing.T
	repoName string
	opts     *ripsrc.Opts
}

func NewTest(t *testing.T, repoName string, opts *ripsrc.Opts) *Test {
	s := &Test{}
	s.t = t
	s.repoName = repoName
	s.opts = opts
	return s
}

func (s *Test) Run() []ripsrc.Branch {
	t := s.t
	dirs := testutil.UnzipTestRepo(s.repoName)
	defer dirs.Remove()

	opts := ripsrc.Opts{}
	if s.opts != nil {
		opts = *s.opts
	}
	opts.AllBranches = true
	opts.RepoDir = dirs.RepoDir
	res, err := ripsrc.New(opts).BranchesSlice(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func assertResult(t *testing.T, want, got []ripsrc.Branch) {
	t.Helper()
	if len(want) != len(got) {
		t.Fatalf("invalid result count, wanted %v, got %v", len(want), len(got))
	}
	for i := range want {
		if !reflect.DeepEqual(want[i], got[i]) {
			t.Fatalf("invalid branch, wanted\n%+v\ngot\n%+v", want[i], got[i])
		}
	}
}
