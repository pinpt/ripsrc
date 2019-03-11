package e2etests

import (
	"context"
	"os"
	"reflect"
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/branches2"
	"github.com/pinpt/ripsrc/ripsrc/gitexec"
	"github.com/pinpt/ripsrc/ripsrc/parentsgraph"
	"github.com/pinpt/ripsrc/ripsrc/pkg/logger"
	"github.com/pinpt/ripsrc/ripsrc/pkg/testutil"
)

type Test struct {
	t        *testing.T
	repoName string
	opts     *branches2.Opts
}

func NewTest(t *testing.T, repoName string, opts *branches2.Opts) *Test {
	s := &Test{}
	s.t = t
	s.repoName = repoName
	s.opts = opts
	return s
}

func (s *Test) Run() []branches2.Branch {
	t := s.t
	dirs := testutil.UnzipTestRepo(s.repoName)
	defer dirs.Remove()

	ctx := context.Background()
	repoDir := dirs.RepoDir
	log := logger.NewDefaultLogger(os.Stdout)
	gitexec.Prepare(ctx, "git", repoDir)

	commitGraph := parentsgraph.New(parentsgraph.Opts{
		RepoDir:     repoDir,
		AllBranches: true,
		Logger:      log,
	})
	err := commitGraph.Read()
	if err != nil {
		t.Fatal(err)
	}

	opts := branches2.Opts{}
	if s.opts != nil {
		opts = *s.opts
	}
	opts.Logger = logger.NewDefaultLogger(os.Stdout)
	opts.Concurrency = 1
	opts.RepoDir = repoDir
	opts.CommitGraph = commitGraph
	res, err := branches2.New(opts).RunSlice(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func assertResult(t *testing.T, want, got []branches2.Branch) {
	t.Helper()
	if len(want) != len(got) {
		t.Fatalf("invalid result count, wanted %v, got %v", len(want), len(got))
	}
	gotCopy := make([]branches2.Branch, len(got))
	copy(gotCopy, got)

	for i := range want {
		g := gotCopy[i]
		g.ID = "" // do not compare id
		if !reflect.DeepEqual(want[i], g) {
			t.Fatalf("invalid branch, wanted\n%+v\ngot\n%+v", want[i], got[i])
		}
	}
}
