package tests

import (
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/commitmeta"
)

func TestBasic(t *testing.T) {
	test := NewTest(t, "basic")
	got := test.Run()

	u1n := "User1"
	u1e := "user1@example.com"
	c1d := parseGitDate("Tue Nov 27 21:55:36 2018 +0100")

	u2n := "User2"
	u2e := "user2@example.com"
	c2d := parseGitDate("Tue Nov 27 21:56:11 2018 +0100")

	f1 := commitmeta.CommitFile{
		Filename:  "main.go",
		Status:    commitmeta.GitFileCommitStatusAdded,
		Additions: 8,
	}

	f2 := commitmeta.CommitFile{
		Filename:  "main.go",
		Status:    commitmeta.GitFileCommitStatusModified,
		Additions: 1,
		Deletions: 3,
	}

	commit1 := commitmeta.Commit{
		SHA:            "b4dadc54e312e976694161c2ac59ab76feb0c40d",
		AuthorName:     u1n,
		AuthorEmail:    u1e,
		CommitterName:  u1n,
		CommitterEmail: u1e,
		Files: map[string]*commitmeta.CommitFile{
			"main.go": &f1,
		},
		Message: "c1",
		Date:    c1d,
		//Parent:   nil,
		Signed: false,
		//Previous: nil,
	}

	commit2 := commitmeta.Commit{
		SHA:            "69ba50fff990c169f80de96674919033a0a9b66d",
		AuthorName:     u2n,
		AuthorEmail:    u2e,
		CommitterName:  u2n,
		CommitterEmail: u2e,
		Files: map[string]*commitmeta.CommitFile{
			"main.go": &f2,
		},
		Message: "c2",
		Date:    c2d,
		//Parent:   strp("b4dadc54e312e976694161c2ac59ab76feb0c40d"),
		Signed: false,
		//Previous: &commit1,
	}

	want := []commitmeta.Commit{commit1, commit2}

	assertCommits(t, want, got)
}
