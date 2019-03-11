package branches2

import "github.com/pinpt/ripsrc/ripsrc/parentsgraph"

func reachableFromBranchStopOnDefault(gr *parentsgraph.Graph, reachableFromDefault reachableFromHead, branchHead string) map[string]bool {
	res := map[string]bool{}
	done := map[string]bool{}
	var rec func(string)
	rec = func(hash string) {
		if done[hash] {
			return
		}
		done[hash] = true
		res[hash] = true
		if reachableFromDefault[hash] {
			return
		}
		for _, p := range gr.Parents[hash] {
			rec(p)
		}
	}
	rec(branchHead)
	return res
}

func behindBranch(
	gr *parentsgraph.Graph,
	reachableFromDefault reachableFromHead,
	branchHead string,
	defaultHead string) (res int) {

	rfb := reachableFromBranchStopOnDefault(gr, reachableFromDefault, branchHead)

	done := map[string]bool{}
	var rec func(string)
	rec = func(hash string) {
		if done[hash] {
			return
		}
		done[hash] = true
		if rfb[hash] {
			return
		}
		res++
		for _, p := range gr.Parents[hash] {
			rec(p)
		}
	}
	rec(defaultHead)
	return res
}
