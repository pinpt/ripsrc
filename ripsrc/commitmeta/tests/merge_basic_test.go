package tests

import (
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/commitmeta"
)

func TestMergeBasic(t *testing.T) {
	test := NewTest(t, "merge_basic")
	got := test.Run(nil)

	u1n := "User1"
	u1e := "user1@example.com"

	c1d := parseGitDate("Tue Dec 4 17:33:31 2018 +0100")

	f1 := commitmeta.CommitFile{
		Filename:  "main.go",
		Status:    commitmeta.GitFileCommitStatusAdded,
		Additions: 4,
	}

	commit1 := commitmeta.Commit{
		SHA:            "cb78f81991af4120b649c5e2ae18cceba598220a",
		AuthorName:     u1n,
		AuthorEmail:    u1e,
		CommitterName:  u1n,
		CommitterEmail: u1e,
		Files: map[string]*commitmeta.CommitFile{
			"main.go": &f1,
		},
		Message: "base",
		Date:    c1d,
		Signed:  false,
		Ordinal: 1,
	}

	c2d := parseGitDate("Tue Dec 4 17:42:10 2018 +0100")

	f2 := commitmeta.CommitFile{
		Filename:  "main.go",
		Status:    commitmeta.GitFileCommitStatusModified,
		Additions: 1,
	}

	commit2 := commitmeta.Commit{
		SHA:            "a08d204ee5913986294000e1280e7ad3484098e3",
		AuthorName:     u1n,
		AuthorEmail:    u1e,
		CommitterName:  u1n,
		CommitterEmail: u1e,
		Files: map[string]*commitmeta.CommitFile{
			"main.go": &f2,
		},
		Message: "a",
		Date:    c2d,
		Parents: []string{"cb78f81991af4120b649c5e2ae18cceba598220a"},
		Signed:  false,
		Ordinal: 2,
	}

	c3d := parseGitDate("Tue Dec 4 17:42:29 2018 +0100")

	f3 := commitmeta.CommitFile{
		Filename:  "main.go",
		Status:    commitmeta.GitFileCommitStatusModified,
		Additions: 1,
	}

	commit3 := commitmeta.Commit{
		SHA:            "3219b85f18fad2aa802344a2275bd8288916f4ee",
		AuthorName:     u1n,
		AuthorEmail:    u1e,
		CommitterName:  u1n,
		CommitterEmail: u1e,
		Files: map[string]*commitmeta.CommitFile{
			"main.go": &f3,
		},
		Message: "m",
		Date:    c3d,
		Parents: []string{"cb78f81991af4120b649c5e2ae18cceba598220a"},
		Signed:  false,
		Ordinal: 3,
	}

	c4d := parseGitDate("Tue Dec 4 17:42:55 2018 +0100")

	f4 := commitmeta.CommitFile{
		Filename:  "main.go",
		Status:    commitmeta.GitFileCommitStatusModified,
		Additions: 1,
	}

	commit4 := commitmeta.Commit{
		SHA:            "49dd6946d595ae6cd51fb228f14c799b749ea3a4",
		AuthorName:     u1n,
		AuthorEmail:    u1e,
		CommitterName:  u1n,
		CommitterEmail: u1e,
		Files: map[string]*commitmeta.CommitFile{
			"main.go": &f4,
		},
		Message: "merge",
		Date:    c4d,
		Parents: []string{"3219b85f18fad2aa802344a2275bd8288916f4ee", "a08d204ee5913986294000e1280e7ad3484098e3"},
		Signed:  false,
		Ordinal: 4,
	}

	want := []commitmeta.Commit{commit1, commit2, commit3, commit4}

	assertCommits(t, want, got)
}
