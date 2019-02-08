package branches2

import "github.com/pinpt/ripsrc/ripsrc/parentsgraph"

func getMergeCommit(gr *parentsgraph.Graph, cache *branchCommitsCache, branchHead string) string {
	children, ok := gr.Children[branchHead]
	if !ok {
		panic("commit not found in tree")
	}
	for _, ch := range children {
		if cache.reachableFromHead[ch] {
			return ch
		}
	}
	return ""
}
