package branch

import (
	"github.com/pinpt/ripsrc/ripsrc/parentsgraph"
)

type HasCommits map[string]bool

func NewCommits(gr *parentsgraph.Graph, headCommit string) HasCommits {
	s := HasCommits{}
	var rec func(h string)
	done := map[string]bool{}
	rec = func(h string) {
		if done[h] {
			return
		}
		done[h] = true
		s[h] = true
		par, ok := gr.Parents[h]
		if !ok {
			panic("commit not found")
		}
		for _, p := range par {
			rec(p)
		}
	}
	rec(headCommit)
	return s
}
