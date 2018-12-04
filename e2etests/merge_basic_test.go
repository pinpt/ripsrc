package e2etests

import (
	"testing"

	"github.com/pinpt/ripsrc/ripsrc"
)

func TestMergeBasic(t *testing.T) {
	test := NewTest(t, "merge_basic")
	got := test.Run()

	u1n := "User1"
	u1e := "user1@example.com"

	c1d := parseGitDate("Tue Dec 4 17:33:31 2018 +0100")

	f1 := ripsrc.CommitFile{
		Filename:  "main.go",
		Status:    ripsrc.GitFileCommitStatusAdded,
		Additions: 4,
	}

	commit1 := ripsrc.Commit{
		Dir:            "",
		SHA:            "cb78f81991af4120b649c5e2ae18cceba598220a",
		AuthorName:     u1n,
		AuthorEmail:    u1e,
		CommitterName:  u1n,
		CommitterEmail: u1e,
		Files: map[string]*ripsrc.CommitFile{
			"main.go": &f1,
		},
		Message: "base",
		Date:    c1d,
		Parent:  nil,
		Signed:  false,
		//Previous: nil,
	}

	c2d := parseGitDate("Tue Dec 4 17:42:29 2018 +0100")

	f2 := ripsrc.CommitFile{
		Filename:  "main.go",
		Status:    ripsrc.GitFileCommitStatusModified,
		Additions: 1,
	}

	commit2 := ripsrc.Commit{
		Dir:            "",
		SHA:            "3219b85f18fad2aa802344a2275bd8288916f4ee",
		AuthorName:     u1n,
		AuthorEmail:    u1e,
		CommitterName:  u1n,
		CommitterEmail: u1e,
		Files: map[string]*ripsrc.CommitFile{
			"main.go": &f2,
		},
		Message: "m",
		Date:    c2d,
		//Parent:   nil,
		Signed: false,
		//Previous: &commit1,
	}

	want := []ripsrc.BlameResult{
		{
			Commit:   commit1,
			Language: "Go",
			Filename: "main.go",
			Lines: []*ripsrc.BlameLine{
				/*
					package main

					func main(){
					}
				*/
				line(u1n, u1e, c1d, false, true, false),
				line(u1n, u1e, c1d, false, false, true),
				line(u1n, u1e, c1d, false, true, false),
				line(u1n, u1e, c1d, false, true, false),
			},
			Size:               28,
			Loc:                4,
			Sloc:               3,
			Comments:           0,
			Blanks:             1,
			Complexity:         0,
			WeightedComplexity: 0,
			Skipped:            "",
			License:            nil,
			Status:             ripsrc.GitFileCommitStatusAdded,
		},
		{
			Commit:   commit2,
			Language: "Go",
			Filename: "main.go",
			Lines: []*ripsrc.BlameLine{
				/*
					package main

					func main(){
					// M
					}
				*/
				line(u1n, u1e, c1d, false, true, false),
				line(u1n, u1e, c1d, false, false, true),
				line(u1n, u1e, c1d, false, true, false),
				line(u1n, u1e, c2d, true, false, false),
				line(u1n, u1e, c1d, false, true, false),
			},
			Size:               33,
			Loc:                5,
			Sloc:               3,
			Comments:           1,
			Blanks:             1,
			Complexity:         0,
			WeightedComplexity: 0,
			Skipped:            "",
			License:            nil,
			Status:             ripsrc.GitFileCommitStatusModified,
		},
	}

	assertResult(t, want, got)
}
