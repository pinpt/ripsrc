package repo

import (
	"container/list"
	"fmt"

	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
)

type Repo map[string]map[string]*incblame.Blame

func New() Repo {
	return Repo{}
}

func (s Repo) CommitsInMemory() int {
	return len(s)
}

func (s Repo) AddCommit(commitHash string) {
	s[commitHash] = map[string]*incblame.Blame{}
}

func (s Repo) GetCommitMust(commitHash string) map[string]*incblame.Blame {
	res, ok := s[commitHash]
	if !ok {
		panic(fmt.Errorf("commit not found: %v", commitHash))
	}
	return res
}

func (s Repo) GetFileOptional(commitHash string, filePath string) *incblame.Blame {
	c, ok := s[commitHash]
	if !ok {
		panic(fmt.Errorf("commit not found: %v when looking for file: %v", commitHash, filePath))
	}
	return c[filePath]
}

func (s Repo) GetFileMust(commitHash string, filePath string) *incblame.Blame {
	c, ok := s[commitHash]
	if !ok {
		panic(fmt.Errorf("commit not found: %v when looking for file: %v", commitHash, filePath))
	}
	res, ok := c[filePath]
	if !ok {
		panic(fmt.Errorf("file is missing in commit. commit: %v file: %v", commitHash, filePath))
	}
	return res
}

type Unloader struct {
	repo     Repo
	toUnload *list.List
}

func NewUnloader(repo Repo) *Unloader {
	s := &Unloader{}
	s.repo = repo
	s.toUnload = list.New()
	return s
}

const maxCommitsInCheckpoint = 1000

func (s *Unloader) Unload(commitHash string) {
	s.toUnload.PushFront(commitHash)

	if s.toUnload.Len() > maxCommitsInCheckpoint {
		last := s.toUnload.Back()
		s.toUnload.Remove(last)
		delete(s.repo, last.Value.(string))
	}
}
