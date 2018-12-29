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

	commit1 := ripsrc.Commit{
		SHA:            "cb78f81991af4120b649c5e2ae18cceba598220a",
		AuthorName:     u1n,
		AuthorEmail:    u1e,
		CommitterName:  u1n,
		CommitterEmail: u1e,
		Files: map[string]*ripsrc.CommitFile{
			"main.go": filep(ripsrc.CommitFile{
				Filename:  "main.go",
				Status:    ripsrc.GitFileCommitStatusAdded,
				Additions: 4,
			}),
		},
		Message: "base",
		Date:    c1d,
	}

	c2d := parseGitDate("Tue Dec 4 17:42:10 2018 +0100")

	commit2 := ripsrc.Commit{
		SHA:            "a08d204ee5913986294000e1280e7ad3484098e3",
		AuthorName:     u1n,
		AuthorEmail:    u1e,
		CommitterName:  u1n,
		CommitterEmail: u1e,
		Files: map[string]*ripsrc.CommitFile{
			"main.go": filep(ripsrc.CommitFile{
				Filename:  "main.go",
				Status:    ripsrc.GitFileCommitStatusModified,
				Additions: 1,
			}),
		},
		Message: "a",
		Date:    c2d,
	}

	c3d := parseGitDate("Tue Dec 4 17:42:29 2018 +0100")

	commit3 := ripsrc.Commit{
		SHA:            "3219b85f18fad2aa802344a2275bd8288916f4ee",
		AuthorName:     u1n,
		AuthorEmail:    u1e,
		CommitterName:  u1n,
		CommitterEmail: u1e,
		Files: map[string]*ripsrc.CommitFile{
			"main.go": filep(ripsrc.CommitFile{
				Filename:  "main.go",
				Status:    ripsrc.GitFileCommitStatusModified,
				Additions: 1,
			}),
		},
		Message: "m",
		Date:    c3d,
		//Parent:   nil,
		Signed: false,
		//Previous: &commit1,
	}

	c4d := parseGitDate("Tue Dec 4 17:42:55 2018 +0100")

	commit4 := ripsrc.Commit{
		SHA:            "49dd6946d595ae6cd51fb228f14c799b749ea3a4",
		AuthorName:     u1n,
		AuthorEmail:    u1e,
		CommitterName:  u1n,
		CommitterEmail: u1e,
		Files: map[string]*ripsrc.CommitFile{
			"main.go": filep(ripsrc.CommitFile{
				Filename:  "main.go",
				Status:    ripsrc.GitFileCommitStatusModified,
				Additions: 1,
			}),
		},
		Message: "merge",
		Date:    c4d,
		Signed:  false,
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
			Size:               29,
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
					// A
					}
				*/
				line(u1n, u1e, c1d, false, true, false),
				line(u1n, u1e, c1d, false, false, true),
				line(u1n, u1e, c1d, false, true, false),
				line(u1n, u1e, c2d, true, false, false),
				line(u1n, u1e, c1d, false, true, false),
			},
			Size:               34,
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
		{
			Commit:   commit3,
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
				line(u1n, u1e, c3d, true, false, false),
				line(u1n, u1e, c1d, false, true, false),
			},
			Size:               34,
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
		{
			Commit:   commit4,
			Language: "Go",
			Filename: "main.go",
			Lines: []*ripsrc.BlameLine{
				/*
					package main

					func main(){
					// A
					// M
					}
				*/
				line(u1n, u1e, c1d, false, true, false),
				line(u1n, u1e, c1d, false, false, true),
				line(u1n, u1e, c1d, false, true, false),
				line(u1n, u1e, c3d, true, false, false),
				line(u1n, u1e, c2d, true, false, false),
				line(u1n, u1e, c1d, false, true, false),
			},
			Size:               39,
			Loc:                6,
			Sloc:               3,
			Comments:           2,
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
