package e2etests

import (
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

// cb callback to defer dirs.Remove()
func (s *Test) Run(optsp *ripsrc.Opts, cb func(*ripsrc.Ripsrc)) {
	dirs := testutil.UnzipTestRepo(s.repoName)
	defer dirs.Remove()

	opts := ripsrc.Opts{}
	if optsp != nil {
		opts = *optsp
	}
	opts.RepoDir = dirs.RepoDir
	cb(ripsrc.New(opts))
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
func assertBlame(t *testing.T, want, got ripsrc.BlameResult) bool {
	if !assertCommitEqual(t, want.Commit, got.Commit) {
		t.Errorf("blame commit does not match, wanted\n%#+v\ngot\n%#+v", want.Commit, got.Commit)
		return false
	}
	if want.Language != got.Language {
		return false
	}
	if want.Filename != got.Filename {
		return false
	}
	if !blameLinesEqual(t, want.Lines, got.Lines) {
		t.Error("blame lines do not match, got")
		for _, l := range got.Lines {
			t.Logf("%+v", l)
		}
		return false
	}
	if want.Size != got.Size {
		return false
	}
	if want.Loc != got.Loc {
		return false
	}
	if want.Sloc != got.Sloc {
		return false
	}
	if want.Comments != got.Comments {
		return false
	}
	if want.Blanks != got.Blanks {
		return false
	}
	if want.Complexity != got.Complexity {
		return false
	}
	if want.WeightedComplexity != got.WeightedComplexity {
		return false
	}
	if want.Skipped != got.Skipped {
		return false
	}
	if want.License != got.License {
		return false
	}
	if want.Status != got.Status {
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
	if l1.SHA != "" {
		if l1.SHA != l2.SHA {
			return false
		}
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
	return r
}

func line(name string, email string, date time.Time, comment, code, blank bool, sha string) *ripsrc.BlameLine {
	return &ripsrc.BlameLine{
		Name:    name,
		Email:   email,
		Date:    date,
		Comment: comment,
		Code:    code,
		Blank:   blank,
		SHA:     sha,
	}
}

func filep(f ripsrc.CommitFile) *ripsrc.CommitFile {
	return &f
}
