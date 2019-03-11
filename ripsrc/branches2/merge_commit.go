package branches2

import "github.com/pinpt/ripsrc/ripsrc/parentsgraph"

func getMergeCommit(gr *parentsgraph.Graph, reachableFromHead reachableFromHead, branchHead string) string {
	children, ok := gr.Children[branchHead]
	if !ok {
		panic("commit not found in tree")
	}
	for _, ch := range children {
		if reachableFromHead[ch] {
			return ch
		}
	}
	return ""
}
