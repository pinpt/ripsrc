package tests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/pinpt/ripsrc/ripsrc/commitmeta"
	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/pkg/testutil"
)

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

func (s *Test) Run(opts *commitmeta.Opts) []commitmeta.Commit {
	t := s.t
	dirs := testutil.UnzipTestRepo(s.repoName)
	defer dirs.Remove()

	if opts == nil {
		opts = &commitmeta.Opts{}
	}

	p := commitmeta.New(dirs.RepoDir, *opts)
	res, err := p.RunSlice()
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func assertCommits(t *testing.T, want, got []commitmeta.Commit) {
	t.Helper()
	if len(want) != len(got) {
		t.Fatalf("invalid number of entries, got %v, wanted %v, got\n%v", len(got), len(want), got)
	}
	for i := range want {
		w := want[i]
		g := got[i]
		if !assert.Equal(t, w, g) {
			t.Fatal()
		}
	}
}

func file(hash string, lines ...*incblame.Line) *incblame.Blame {
	return &incblame.Blame{Commit: hash, Lines: lines}
}

func line(buf string, commit string) *incblame.Line {
	return &incblame.Line{Line: []byte(buf), Commit: commit}
}

func parseGitDate(s string) time.Time {
	//Tue Nov 27 21:55:36 2018 +0100
	r, err := time.Parse("Mon Jan 2 15:04:05 2006 -0700", s)
	if err != nil {
		panic(err)
	}
	return r
}

func strp(s string) *string {
	return &s
}
