package e2etests

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/pinpt/ripsrc/ripsrc"
	"github.com/pinpt/ripsrc/ripsrc/pkg/testutil"
)

type Test struct {
	t        *testing.T
	repoName string
	tempDir  string
}

func NewTest(t *testing.T, repoName string) *Test {
	s := &Test{}
	s.t = t
	s.repoName = repoName
	return s
}

func (s *Test) Run(optsp *ripsrc.Opts) []ripsrc.BlameResult {
	t := s.t
	dirs := testutil.UnzipTestRepo(s.repoName)
	defer dirs.Remove()

	opts := ripsrc.Opts{}
	if optsp != nil {
		opts = *optsp
	}
	opts.RepoDir = dirs.RepoDir
	res, err := ripsrc.New(opts).BlameSlice(context.Background())
	if err != nil {
		t.Fatal("Rip returned error", err)
	}
	return res
}

func assertResult(t *testing.T, want, got []ripsrc.BlameResult) {
	t.Helper()
	if len(want) != len(got) {
		t.Fatalf("invalid result count, wanted %v, got %v", len(want), len(got))
	}
	for i := range want {

		if !assertBlame(t, want[i], got[i]) {
			t.Fatalf("invalid blame, wanted\n%+v\ngot\n%+v", want[i], got[i])
		}
	}
}

// needed because BlameResult has private fields
func assertBlame(t *testing.T, b1, b2 ripsrc.BlameResult) bool {
	if !assertCommitEqual(t, b1.Commit, b2.Commit) {
		t.Errorf("blame commit does not match, wanted\n%#+v\ngot\n%#+v", b1.Commit, b2.Commit)
		return false
	}
	if b1.Language != b2.Language {
		return false
	}
	if b1.Filename != b2.Filename {
		return false
	}
	if !blameLinesEqual(t, b1.Lines, b2.Lines) {
		t.Error("blame lines do not match, got")
		for _, l := range b2.Lines {
			t.Logf("%+v", l)
		}
		return false
	}
	if b1.Size != b2.Size {
		return false
	}
	if b1.Loc != b2.Loc {
		return false
	}
	if b1.Sloc != b2.Sloc {
		return false
	}
	if b1.Comments != b2.Comments {
		return false
	}
	if b1.Blanks != b2.Blanks {
		return false
	}
	if b1.Complexity != b2.Complexity {
		return false
	}
	if b1.WeightedComplexity != b2.WeightedComplexity {
		return false
	}
	if b1.Skipped != b2.Skipped {
		return false
	}
	if b1.License != b2.License {
		return false
	}
	if b1.Status != b2.Status {
		return false
	}
	return true
}

func blameLinesEqual(t *testing.T, b1, b2 []*ripsrc.BlameLine) bool {
	if len(b1) != len(b2) {
		return false
	}
	for i := range b1 {
		v1 := b1[i]
		v2 := b2[i]
		if !blameLineEqual(t, v1, v2) {
			t.Logf("blame line not equal\n%v\n%v", v1, v2)
			return false
		}
	}
	return true
}

func blameLineEqual(t *testing.T, l1, l2 *ripsrc.BlameLine) bool {
	if l1.Name != l2.Name {
		return false
	}
	if l1.Email != l2.Email {
		return false
	}
	if !l1.Date.Equal(l2.Date) {
		return false
	}
	if l1.Comment != l2.Comment {
		return false
	}
	if l1.Code != l2.Code {
		return false
	}
	if l1.Blank != l2.Blank {
		return false
	}
	return true
}

// needed because Commit has private fields
func assertCommitEqual(t *testing.T, c1, c2 ripsrc.Commit) bool {
	// TODO: commit is using full path including path to repo
	// this is probably a bug
	//if c1.Dir != c2.Dir {
	//	t.Error("commit.Dir mismatch")
	//	return false
	//}
	if c1.SHA != c2.SHA {
		t.Error("commit.SHA mismatch")
		return false
	}
	if c1.AuthorName != c2.AuthorName {
		t.Error("commit.AuthorName mismatch")
		return false
	}
	if c1.AuthorEmail != c2.AuthorEmail {
		t.Error("commit.AuthorEmail mismatch")
		return false
	}
	if c1.CommitterName != c2.CommitterName {
		t.Error("commit.CommitterName mismatch")
		return false
	}
	if c1.CommitterEmail != c2.CommitterEmail {
		t.Error("commit.CommitterEmail mismatch")
		return false
	}
	if !reflect.DeepEqual(c1.Files, c2.Files) {
		t.Errorf("commit.Files mismatch")
		t.Log("got")
		for k, f := range c2.Files {
			t.Logf("%v %+v", k, f)
		}
		return false
	}
	if c1.Date != c2.Date {
		t.Error("commit.Date mismatch")
		return false
	}
	// internally incremented counter, don't need to check
	if c1.Ordinal != c2.Ordinal {
		t.Error("commit.Ordinal mismatch")
		return false
	}
	if c1.Message != c2.Message {
		t.Error("commit.Message mismatch")
		return false
	}
	//if c1.Parent != c2.Parent {
	//	t.Error("commit.Parent mismatch")
	//	return false
	//}
	//if c1.Previous != c2.Previous {
	//	t.Error("commit.Previous mismatch")
	//	return false
	//}
	return true
}

func commitFileEqual(f1, f2 *ripsrc.CommitFile) bool {
	return true
}

func parseGitDate(s string) time.Time {
	//Tue Nov 27 21:55:36 2018 +0100
	r, err := time.Parse("Mon Jan 2 15:04:05 2006 -0700", s)
	if err != nil {
		panic(err)
	}
	return r.UTC()
}

func line(name string, email string, date time.Time, comment, code, blank bool) *ripsrc.BlameLine {
	return &ripsrc.BlameLine{
		Name:    name,
		Email:   email,
		Date:    date,
		Comment: comment,
		Code:    code,
		Blank:   blank}
}

func filep(f ripsrc.CommitFile) *ripsrc.CommitFile {
	return &f
}
