package branches2

import (
	"sort"

	"github.com/pinpt/ripsrc/ripsrc/parentsgraph"
)

type branchCommitsCache struct {
	// reachableFromHead is a map that has true for all commits that belong to head
	// (in default ripsrc config, that will be the default branch)
	// map[commitSha]isReachableFromHead
	reachableFromHead map[string]bool
}

func newBranchCommitsCache(gr *parentsgraph.Graph, defaultHead string) *branchCommitsCache {
	s := &branchCommitsCache{}
	s.reachable(gr, defaultHead)
	return s
}

func (s *branchCommitsCache) reachable(gr *parentsgraph.Graph, defaultHead string) {
	s.reachableFromHead = map[string]bool{}
	var rec func(string)
	rec = func(hash string) {
		s.reachableFromHead[hash] = true
		for _, p := range gr.Parents[hash] {
			rec(p)
		}
	}
	rec(defaultHead)
}

func branchCommits(
	gr *parentsgraph.Graph,
	defaultHead string,
	cache *branchCommitsCache,
	branchHead string) (commits []string, branchedFrom []string) {

	reachableFromHead := cache.reachableFromHead

	if reachableFromHead[branchHead] {
		// this is a merged commit, we would need to recreate reachableFromHead without merge commit
		// this is an expensive operation
		reachableFromHead = map[string]bool{}
		var rec func(string)
		rec = func(hash string) {
			reachableFromHead[hash] = true
			if hash == branchHead {
				// remove merge commit to branch head
				return
			}
			par, ok := gr.Parents[hash]
			if !ok {
				panic("commit not found in tree")
			}
			for _, p := range par {
				rec(p)
			}
		}
		rec(defaultHead)
	}
	var rec func(string)
	rec = func(hash string) {
		commits = append(commits, hash)
		par, ok := gr.Parents[hash]
		if !ok {
			panic("commit not found in tree")
		}
		// reverse order for better result ordering (see tests)
		for i := len(par) - 1; i >= 0; i-- {
			p := par[i]
			if reachableFromHead[p] {
				branchedFrom = append(branchedFrom, p)
				continue
			}
			rec(p)
		}
	}
	rec(branchHead)
	reverseStrings(commits)

	branchedFrom = dedupLinearFromHead(gr, branchedFrom, branchHead)
	return
}

func dedupLinearFromHead(gr *parentsgraph.Graph, commits []string, defaultHead string) []string {
	commitsHash := toSet(commits)
	dup := map[string]bool{}
	var rec func(string, string)
	rec = func(hash, active string) {
		if commitsHash[hash] {
			if active != "" {
				dup[active] = true
			}
			active = hash
		}
		for _, p := range gr.Parents[hash] {
			rec(p, active)
		}
	}
	rec(defaultHead, "")

	var res []string
	for c := range commitsHash {
		if !dup[c] {
			res = append(res, c)
		}
	}
	sort.Strings(res) // to have consistent order
	return res
}

func toSet(arr []string) map[string]bool {
	res := map[string]bool{}
	for _, v := range arr {
		res[v] = true
	}
	return res
}

func reverseStrings(arr []string) {
	for i, j := 0, len(arr)-1; i < j; i, j = i+1, j-1 {
		arr[i], arr[j] = arr[j], arr[i]
	}
}
