package incblame

import "fmt"

// ApplyMerge creates a new blame data for file based on parent blame data and merge diff for each parent (generated using git -m option)
func ApplyMerge(parents []Blame, diffs []Diff, commit string) Blame {
	//fmt.Println("apply")
	//for i, p := range parents {
	//	fmt.Println("parent", i)
	//	fmt.Println(p)
	//}

	// different view for each parent
	var cand []Blame
	for i, p := range parents {
		res := applyOneParent(p, diffs[i], commit)
		cand = append(cand, res)
	}
	// check that cand lines are eq
	var lenLines []int
	for _, c := range cand {
		lenLines = append(lenLines, len(c.Lines))
	}
	for _, c := range cand {
		if len(c.Lines) != len(cand[0].Lines) {
			panic(fmt.Errorf("not all resulting blames have the same num of lines %v", lenLines))
		}
	}

	// now use the source for each
	res := cand[0]
	for i := range res.Lines {
		for j := range parents {
			line := cand[j].Lines[i]
			// if commit is not the merge commit that means the line appeared from that parent use it in res, in case multiple sources, first will be used
			if line.Commit != commit {
				res.Lines[i].Commit = line.Commit
				break
			}
		}
		// if the line originated from none of the parents it will be set to commit, because it was created here
	}
	return res
}
