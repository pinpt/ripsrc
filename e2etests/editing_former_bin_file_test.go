package e2etests

import (
	"testing"

	"github.com/pinpt/ripsrc/ripsrc"
)

// This is a test case for the following condition.
// If the file in repo was a binary at some point and then switched to text and was modified, then git log with patches does not contain the full file content. There are 2 options to fix this, either we ignore all files that at some point in history were binary or retrieve the full file content for these cases separately without using log and patches.
func TestEditingFormerBinFile(t *testing.T) {
	test := NewTest(t, "editing_former_bin_file")
	got := test.Run(nil)

	u1n := "User1"
	u1e := "user1@example.com"

	c1d := parseGitDate("Wed Nov 28 20:12:51 2018 +0100")
	c2d := parseGitDate("Wed Nov 28 20:13:11 2018 +0100")
	c3d := parseGitDate("Wed Nov 28 20:13:20 2018 +0100")

	f1 := ripsrc.CommitFile{
		Filename: "main.go",
		Status:   ripsrc.GitFileCommitStatusAdded,
		Binary:   true,
	}

	commit1 := ripsrc.Commit{
		SHA:            "94909fcc06b6a65bf865f94fbc22e5ace8fbbbd6",
		AuthorName:     u1n,
		AuthorEmail:    u1e,
		CommitterName:  u1n,
		CommitterEmail: u1e,
		Files: map[string]*ripsrc.CommitFile{
			"main.go": &f1,
		},
		Message: "c1",
		Date:    c1d,
		Ordinal: 1,
	}

	f2 := ripsrc.CommitFile{
		Filename: "main.go",
		Status:   ripsrc.GitFileCommitStatusModified,
		Binary:   true,
	}

	commit2 := ripsrc.Commit{
		SHA:            "199a1819583fc0da098f0d8328acbf43d35f3541",
		AuthorName:     u1n,
		AuthorEmail:    u1e,
		CommitterName:  u1n,
		CommitterEmail: u1e,
		Files: map[string]*ripsrc.CommitFile{
			"main.go": &f2,
		},
		Message: "c2",
		Date:    c2d,
		Ordinal: 2,
	}

	f3 := ripsrc.CommitFile{
		Filename:  "main.go",
		Status:    ripsrc.GitFileCommitStatusModified,
		Deletions: 1,
	}

	commit3 := ripsrc.Commit{
		SHA:            "831589d24aba19a83aa080194ba335a67da0413e",
		AuthorName:     u1n,
		AuthorEmail:    u1e,
		CommitterName:  u1n,
		CommitterEmail: u1e,
		Files: map[string]*ripsrc.CommitFile{
			"main.go": &f3,
		},
		Message: "c3",
		Date:    c3d,
		Ordinal: 3,
	}

	want := []ripsrc.BlameResult{
		{
			Commit:   commit1,
			Language: "Go",
			Filename: "main.go",
			Lines:    []*ripsrc.BlameLine{},
			Skipped:  "",
			License:  nil,
			Status:   ripsrc.GitFileCommitStatusAdded,
		},
		{
			Commit:   commit2,
			Language: "Go",
			Filename: "main.go",
			Lines:    []*ripsrc.BlameLine{},
			License:  nil,
			Status:   ripsrc.GitFileCommitStatusModified,
		},
		{
			Commit:   commit3,
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
			Size:               29,
			Loc:                4,
			Sloc:               3,
			Comments:           0,
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
