package repo

import (
	"container/list"
	"fmt"
	"sort"
	"strings"

	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
)

type Repo map[string]map[string]*incblame.Blame

func New() Repo {
	return Repo{}
}

func (s Repo) Debug() string {
	res := []string{}
	type KV struct {
		K string
		V map[string]*incblame.Blame
	}
	var arr []KV
	for k, v := range s {
		arr = append(arr, KV{k, v})
	}
	sort.Slice(arr, func(i, j int) bool {
		a := arr[i]
		b := arr[j]
		return a.K < b.K
	})
	line := func(str string) {
		res = append(res, str)
		res = append(res, "\n")
	}
	line("")
	for _, v := range arr {
		commit := v.K
		line("commit:" + commit)
		type KV struct {
			K string
			V *incblame.Blame
		}
		var arr []KV
		for k, v := range v.V {
			arr = append(arr, KV{k, v})
		}
		sort.Slice(arr, func(i, j int) bool {
			a := arr[i]
			b := arr[j]
			return a.K < b.K
		})
		for _, v := range arr {
			line("commit:" + commit)
			line("file:" + v.K)
			line(v.V.String())
			line("")
		}
	}
	line("")
	return strings.Join(res, "")
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
