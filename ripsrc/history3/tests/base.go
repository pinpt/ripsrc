package tests

import (
	"context"
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/gitexec"
	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/history3/process"
	"github.com/pinpt/ripsrc/ripsrc/pkg/testutil"
)

var gitCommand = "git"

type Test struct {
	t        *testing.T
	repoName string
	tempDir  string
}

func NewTest(t *testing.T, repoName string) *Test {
	s := &Test{}
	s.t = t
	s.repoName = repoName
	return s
}

func (s *Test) Run(opts *process.Opts) []process.Result {
	t := s.t
	dirs := testutil.UnzipTestRepo(s.repoName)
	defer dirs.Remove()

	ctx := context.Background()
	err := gitexec.Prepare(ctx, gitCommand, dirs.RepoDir)
	if err != nil {
		t.Fatal(err)
	}

	if opts == nil {
		opts = &process.Opts{}
	}
	opts.RepoDir = dirs.RepoDir
	opts.DisableCache = true

	p := process.New(*opts)
	res, err := p.RunGetAll()
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func assertResult(t *testing.T, want, got []process.Result) {
	t.Helper()
	if len(want) != len(got) {
		t.Fatalf("invalid number of results %v, got\n%v", len(got), got)
	}

	for i := range want {
		w := want[i]
		g := got[i]
		if w.Commit != g.Commit {
			t.Fatalf("invalid commit hash %v at pos %v", w.Commit, i)
		}
		commit := w.Commit
		if len(w.Files) != len(g.Files) {
			t.Fatalf("invalid number of entries %v for commit %v, got\n%v", len(g.Files), commit, g.Files)
		}
		for filePath := range w.Files {
			gf := g.Files[filePath]
			if gf == nil {
				t.Fatalf("missing file %v commit %v\nwanted\n%v", filePath, commit, w.Files[filePath])
			}
			if !w.Files[filePath].Eq(gf) {
				t.Fatalf("invalid patch for file %v commit %v", filePath, commit)
				//t.Fatalf("invalid patch for file %v commit %v, got\n%v\nwanted\n%v", filePath, commit, g.Files[filePath], w.Files[filePath])
			}
		}
	}
}

func file(hash string, lines ...*incblame.Line) *incblame.Blame {
	return &incblame.Blame{Commit: hash, Lines: lines}
}

func line(buf string, commit string) *incblame.Line {
	return &incblame.Line{Line: []byte(buf), Commit: commit}
}
