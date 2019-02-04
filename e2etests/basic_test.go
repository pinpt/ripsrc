package e2etests

import (
	"testing"

	"github.com/pinpt/ripsrc/ripsrc"
)

func TestBasic(t *testing.T) {
	test := NewTest(t, "basic")
	got := test.Run(nil)

	u1n := "User1"
	u1e := "user1@example.com"
	c1d := parseGitDate("Tue Nov 27 21:55:36 2018 +0100")

	u2n := "User2"
	u2e := "user2@example.com"
	c2d := parseGitDate("Tue Nov 27 21:56:11 2018 +0100")

	f1 := ripsrc.CommitFile{
		Filename:  "main.go",
		Status:    ripsrc.GitFileCommitStatusAdded,
		Additions: 8,
	}

	f2 := ripsrc.CommitFile{
		Filename:  "main.go",
		Status:    ripsrc.GitFileCommitStatusModified,
		Additions: 1,
		Deletions: 3,
	}

	commit1 := ripsrc.Commit{
		SHA:            "b4dadc54e312e976694161c2ac59ab76feb0c40d",
		AuthorName:     u1n,
		AuthorEmail:    u1e,
		CommitterName:  u1n,
		CommitterEmail: u1e,
		Files: map[string]*ripsrc.CommitFile{
			"main.go": &f1,
		},
		Message: "c1",
		Date:    c1d,
		//Parent:  nil,
		Signed: false,
		//Previous: nil,
		Ordinal: 1,
	}

	commit2 := ripsrc.Commit{
		SHA:            "69ba50fff990c169f80de96674919033a0a9b66d",
		AuthorName:     u2n,
		AuthorEmail:    u2e,
		CommitterName:  u2n,
		CommitterEmail: u2e,
		Files: map[string]*ripsrc.CommitFile{
			"main.go": &f2,
		},
		Message: "c2",
		Date:    c2d,
		//Parent:   nil,
		Signed: false,
		//Previous: &commit1,
		Ordinal: 2,
	}

	want := []ripsrc.BlameResult{
		{
			Commit:   commit1,
			Language: "Go",
			Filename: "main.go",
			Lines: []*ripsrc.BlameLine{
				/*
				   package main

				   import "github.com/pinpt/ripsrc/cmd"

				   func main() {
				   	cmd.Execute()
				   }
				*/
				line(u1n, u1e, c1d, false, true, false),
				line(u1n, u1e, c1d, false, false, true),
				line(u1n, u1e, c1d, false, true, false),
				line(u1n, u1e, c1d, false, false, true),
				line(u1n, u1e, c1d, false, true, false),
				line(u1n, u1e, c1d, false, true, false),
				line(u1n, u1e, c1d, false, true, false),
				line(u1n, u1e, c1d, false, false, true),
			},
			Size:               84,
			Loc:                8,
			Sloc:               5,
			Comments:           0,
			Blanks:             3,
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

					func main() {
					  // do nothing
					}
				*/
				line(u1n, u1e, c1d, false, true, false),
				line(u1n, u1e, c1d, false, false, true),
				line(u1n, u1e, c1d, false, true, false),
				line(u2n, u2e, c2d, true, false, false),
				line(u1n, u1e, c1d, false, true, false),
				line(u1n, u1e, c1d, false, false, true),
			},
			Size:               47,
			Loc:                6,
			Sloc:               3,
			Comments:           1,
			Blanks:             2,
			Complexity:         0,
			WeightedComplexity: 0,
			Skipped:            "",
			License:            nil,
			Status:             ripsrc.GitFileCommitStatusModified,
		},
	}

	assertResult(t, want, got)
}
