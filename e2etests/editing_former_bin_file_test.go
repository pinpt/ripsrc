package e2etests

/*

// This is a test case for the following condition.
// If the file in repo was a binary at some point and then switched to text and was modified, then git log with patches does not contain the full file content. There are 2 options to fix this, either we ignore all files that at some point in history were binary or retrieve the full file content for these cases separately without using log and patches.
func TestEditingFormerBinFile(t *testing.T) {
	test := NewTest(t, "editing_former_bin_file")
	got := test.Run()

	{
		u1n := "User1"
		u1e := "user1@example.com"

		c1d := parseGitDate("Wed Nov 28 20:12:51 2018 +0100")

		f1 := ripsrc.CommitFile{
			Filename: "main.go",
			Status:   ripsrc.GitFileCommitStatusAdded,
			Binary:   true,
		}

		commit1 := ripsrc.Commit{
			Dir:            "",
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
			Parent:  nil,
			Signed:  false,
			//Previous: nil,
		}

		want := []ripsrc.BlameResult{
			{
				Commit:             commit1,
				Language:           "",
				Filename:           "main.go",
				Lines:              nil,
				Size:               0,
				Loc:                0,
				Sloc:               0,
				Comments:           0,
				Blanks:             0,
				Complexity:         0,
				WeightedComplexity: 0,
				Skipped:            "File was binary",
				License:            nil,
				Status:             ripsrc.GitFileCommitStatusAdded,
			},
		}

		assertResult(t, want, got)
	}

	return

	// TODO: we should retain history after file switches from binary to text

	fmt.Println(got)

	u1n := "User1"
	u1e := "user1@example.com"

	//c1d := parseGitDate("Wed Nov 28 20:12:51 2018 +0100")
	c2d := parseGitDate("Wed Nov 28 20:13:11 2018 +0100")

	f2 := ripsrc.CommitFile{
		Filename:  "main.go",
		Status:    ripsrc.GitFileCommitStatusModified,
		Additions: 1,
		Deletions: 3,
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
			Commit:   commit2,
			Language: "Go",
			Filename: "main.go",
			Lines:    []*ripsrc.BlameLine{
				//line(u1n, u1e, c1d, false, true, false),
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

	assertResult(t, want, got)
}
*/
