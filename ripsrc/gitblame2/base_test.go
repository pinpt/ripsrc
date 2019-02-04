package gitblame2

import (
	"testing"

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

func (s *Test) Run(hash, filePath string) (Result, error) {
	dirs := testutil.UnzipTestRepo(s.repoName)
	defer dirs.Remove()

	return Run(dirs.RepoDir, hash, filePath)
}
