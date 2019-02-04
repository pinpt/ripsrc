package tests

import (
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/commitmeta"
)

func TestMultipleBranches1(t *testing.T) {
	test := NewTest(t, "multiple_branches")
	got := test.Run(&commitmeta.Opts{AllBranches: true})

	name := "none"
	email := "none"

	f1 := commitmeta.CommitFile{
		Filename:  "a.txt",
		Status:    commitmeta.GitFileCommitStatusAdded,
		Additions: 1,
	}

	commit1 := commitmeta.Commit{
		SHA:            "bdf8c8cfa9c027e58f1aea5c532ba0e9ef74bc4c",
		AuthorName:     name,
		AuthorEmail:    email,
		CommitterName:  name,
		CommitterEmail: email,
		Files: map[string]*commitmeta.CommitFile{
			"a.txt": &f1,
		},
		Message: "c1",
		Date:    parseGitDate("Mon Feb 4 12:58:55 2019 +0100"),
		Ordinal: 1,
	}

	f2 := commitmeta.CommitFile{
		Filename:  "a.txt",
		Status:    commitmeta.GitFileCommitStatusModified,
		Additions: 1,
	}

	commit2 := commitmeta.Commit{
		SHA:            "d3a93f475772c90918ebc34e144e1c3554163a9f",
		Parents:        []string{"bdf8c8cfa9c027e58f1aea5c532ba0e9ef74bc4c"},
		AuthorName:     name,
		AuthorEmail:    email,
		CommitterName:  name,
		CommitterEmail: email,
		Files: map[string]*commitmeta.CommitFile{
			"a.txt": &f2,
		},
		Message: "c2",
		Date:    parseGitDate("Mon Feb 4 12:59:28 2019 +0100"),
		Ordinal: 2,
	}

	f3 := commitmeta.CommitFile{
		Filename:  "a.txt",
		Status:    commitmeta.GitFileCommitStatusModified,
		Additions: 1,
		Deletions: 1,
	}

	commit3 := commitmeta.Commit{
		SHA:            "7c6eba56ba8616ee903f2394553c022d6d3046bf",
		Parents:        []string{"bdf8c8cfa9c027e58f1aea5c532ba0e9ef74bc4c"},
		AuthorName:     name,
		AuthorEmail:    email,
		CommitterName:  name,
		CommitterEmail: email,
		Files: map[string]*commitmeta.CommitFile{
			"a.txt": &f3,
		},
		Message: "c3",
		Date:    parseGitDate("Mon Feb 4 12:59:42 2019 +0100"),
		Ordinal: 3,
	}

	f4 := commitmeta.CommitFile{
		Filename:  "a.txt",
		Status:    commitmeta.GitFileCommitStatusModified,
		Additions: 1,
	}

	commit4 := commitmeta.Commit{
		SHA:            "3f18a2ea07832a18d0645df2aa666b339cee1a06",
		Parents:        []string{"bdf8c8cfa9c027e58f1aea5c532ba0e9ef74bc4c"},
		AuthorName:     name,
		AuthorEmail:    email,
		CommitterName:  name,
		CommitterEmail: email,
		Files: map[string]*commitmeta.CommitFile{
			"a.txt": &f4,
		},
		Message: "c4",
		Date:    parseGitDate("Mon Feb 4 13:00:29 2019 +0100"),
		Ordinal: 4,
	}

	want := []commitmeta.Commit{commit1, commit2, commit3, commit4}

	assertCommits(t, want, got)
}

func TestMultipleBranchesDisabled(t *testing.T) {
	test := NewTest(t, "multiple_branches_disabled")
	got := test.Run(nil)

	name := "none"
	email := "none"

	f1 := commitmeta.CommitFile{
		Filename:  "a.txt",
		Status:    commitmeta.GitFileCommitStatusAdded,
		Additions: 1,
	}

	commit1 := commitmeta.Commit{
		SHA:            "bba6ce31b58bd8b864b0c3eb4fb8856b2dcc0297",
		AuthorName:     name,
		AuthorEmail:    email,
		CommitterName:  name,
		CommitterEmail: email,
		Files: map[string]*commitmeta.CommitFile{
			"a.txt": &f1,
		},
		Message: "c1",
		Date:    parseGitDate("Mon Feb 4 13:06:01 2019 +0100"),
		Ordinal: 1,
	}

	want := []commitmeta.Commit{commit1}

	assertCommits(t, want, got)
}
