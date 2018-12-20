package e2etests

import (
	"testing"

	"github.com/pinpt/ripsrc/ripsrc"
)

func TestDeletedFiles(t *testing.T) {
	test := NewTest(t, "deleted_files")
	got := test.Run()

	u1n := "User1"
	u1e := "user1@example.com"
	c1d := parseGitDate("Wed Dec 12 18:11:52 2018 +0100")
	c2d := parseGitDate("Wed Dec 12 18:12:05 2018 +0100")

	f1 := ripsrc.CommitFile{
		Filename:  "a.go",
		Status:    ripsrc.GitFileCommitStatusAdded,
		Additions: 4,
	}

	f2 := ripsrc.CommitFile{
		Filename:  "a.go",
		Status:    ripsrc.GitFileCommitStatusRemoved,
		Deletions: 4,
	}

	commit1 := ripsrc.Commit{
		Dir:            "",
		SHA:            "624f3a74bf727e365cfbd090b9b993ddded0e1ea",
		AuthorName:     u1n,
		AuthorEmail:    u1e,
		CommitterName:  u1n,
		CommitterEmail: u1e,
		Files: map[string]*ripsrc.CommitFile{
			"a.go": &f1,
		},
		Message: "c1",
		Date:    c1d,
		Parent:  nil,
		Signed:  false,
		//Previous: nil,
	}

	commit2 := ripsrc.Commit{
		Dir:            "",
		SHA:            "9c7629df59b283bdec8b9705cb17c822652f6fae",
		AuthorName:     u1n,
		AuthorEmail:    u1e,
		CommitterName:  u1n,
		CommitterEmail: u1e,
		Files: map[string]*ripsrc.CommitFile{
			"a.go": &f2,
		},
		Message: "c2",
		Date:    c2d,
		//Parent:   nil,
		Signed: false,
		//Previous: &commit1,
	}

	want := []ripsrc.BlameResult{
		{
			Commit:   commit1,
			Language: "Go",
			Filename: "a.go",
			Lines: []*ripsrc.BlameLine{
				/*
				   package main

				   func main() {
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
			Commit:             commit2,
			Language:           "",
			Filename:           "a.go",
			Lines:              []*ripsrc.BlameLine{},
			Size:               0,
			Loc:                0,
			Sloc:               0,
			Comments:           0,
			Blanks:             0,
			Complexity:         0,
			WeightedComplexity: 0,
			Skipped:            "File was removed",
			License:            nil,
			Status:             ripsrc.GitFileCommitStatusRemoved,
		},
	}

	assertResult(t, want, got)
}
