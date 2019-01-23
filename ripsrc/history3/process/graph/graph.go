package graph

// Graph represets the graph of commits in git repo
// map[commit.sha]commit.sha_of_parents
type Graph map[string][]string

type trace struct {
	headInd int
	commit  string
}

func (g Graph) LastCommonParent(heads []string) string {
	var curr []trace
	for i, h := range heads {
		curr = append(curr, trace{i, h})
	}
	reached := map[string]map[int]bool{}
	markReach := func(tr trace) (done bool) {
		commit := tr.commit
		fromHead := tr.headInd
		if _, ok := reached[commit]; !ok {
			reached[commit] = map[int]bool{}
		}
		reached[commit][fromHead] = true
		if len(reached[commit]) == len(heads) {
			return true
		}
		return false
	}
	for {
		var ntr []trace
		for _, tr := range curr {
			done := markReach(tr)
			if done {
				return tr.commit
			}
			parents := g[tr.commit]
			for _, p := range parents {
				ntr = append(ntr, trace{headInd: tr.headInd, commit: p})
			}
		}
		if len(ntr) == 0 {
			panic("all roots reached")
		}
		curr = ntr
	}
}
