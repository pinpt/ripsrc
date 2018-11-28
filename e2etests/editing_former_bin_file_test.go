package e2etests

import (
	"testing"
)

func TestEditingFormerBinFile(t *testing.T) {
	test := NewTest(t, "editing_former_bin_file")
	_ = test.Run()

	/*
		got := test.Run()

		u1n := "User1"
		u1e := "user1@example.com"

		c1d := parseGitDate("Wed Nov 28 20:12:51 2018 +0100")
		c2d := parseGitDate("Wed Nov 28 20:13:11 2018 +0100")
		//c3d := parseGitDate("Wed Nov 28 20:13:20 2018 +0100")

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
			Dir:            "",
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
			Parent:  nil,
			Signed:  false,
			//Previous: nil,
		}

		commit2 := ripsrc.Commit{
			Dir:            "",
			SHA:            "69ba50fff990c169f80de96674919033a0a9b66d",
			AuthorName:     u1n,
			AuthorEmail:    u1e,
			CommitterName:  u1n,
			CommitterEmail: u1e,
			Files: map[string]*ripsrc.CommitFile{
				"main.go": &f2,
			},
			Message: "c2",
			Date:    c2d,
			//Parent:   nil,
			Signed: false,
			//Previous: &commit1,
		}

		want := []ripsrc.BlameResult{
			{
				Commit:             commit1,
				Language:           "Go",
				Filename:           "main.go",
				Lines:              []*ripsrc.BlameLine{},
				Size:               83,
				Loc:                7,
				Sloc:               5,
				Comments:           0,
				Blanks:             2,
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
					line(u1n, u1e, c1d, false, true, false),
				},
				Size:               46,
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

		assertResult(t, want, got)*/
}
