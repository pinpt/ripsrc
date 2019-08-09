package e2etests

import (
	"context"
	"os"
	"reflect"
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/branchmeta"
	"github.com/pinpt/ripsrc/ripsrc/gitexec"
	"github.com/pinpt/ripsrc/ripsrc/pkg/logger"
	"github.com/pinpt/ripsrc/ripsrc/pkg/testutil"
)

type Test struct {
	t        *testing.T
	repoName string
	opts     *branchmeta.Opts
}

func NewTest(t *testing.T, repoName string, opts *branchmeta.Opts) *Test {
	s := &Test{}
	s.t = t
	s.repoName = repoName
	s.opts = opts
	return s
}

func (s *Test) Run() []branchmeta.Branch {
	t := s.t
	dirs := testutil.UnzipTestRepo(s.repoName)
	defer dirs.Remove()

	ctx := context.Background()
	repoDir := dirs.RepoDir
	gitexec.Prepare(ctx, "git", repoDir)

	opts := branchmeta.Opts{}
	if s.opts != nil {
		opts = *s.opts
	}
	opts.Logger = logger.NewDefaultLogger(os.Stdout)
	opts.RepoDir = repoDir
	res, err := branchmeta.Get(ctx, opts)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func assertResult(t *testing.T, want, got []branchmeta.Branch) {
	t.Helper()
	if len(want) != len(got) {
		t.Fatalf("invalid result count, wanted %v, got %v", len(want), len(got))
	}
	gotCopy := make([]branchmeta.Branch, len(got))
	copy(gotCopy, got)

	for i := range want {
		g := gotCopy[i]
		if !reflect.DeepEqual(want[i], g) {
			t.Fatalf("invalid branch, wanted\n%+v\ngot\n%+v", want[i], got[i])
		}
	}
}
