package ripsrc

import (
	"time"

	"github.com/pinpt/ripsrc/ripsrc/commitmeta"
)

// Commit is a specific detail around a commit
type Commit struct {
	// Fields from commitmeta.Commit.
	// Definition copy to allow extra fields. Not using embedding to allow initialization without the following error:
	// cannot use promoted field .... in struct literal of type

	SHA            string
	AuthorName     string
	AuthorEmail    string
	CommitterName  string
	CommitterEmail string
	Files          map[string]*CommitFile
	Date           time.Time
	Ordinal        int64
	Message        string
	Parents        []string
	Signed         bool

	// Extra fields fields

	// Branches from which this commit is reachable. Uses current branch references.
	// This field is only populated when AllBranches=true.
	Branches []string
}

// CommitFile is a specific detail around a file in a commit
type CommitFile = commitmeta.CommitFile

func commitFromMeta(c commitmeta.Commit, branches []string) (res Commit) {
	res.Branches = branches

	res.SHA = c.SHA
	res.AuthorName = c.AuthorName
	res.AuthorEmail = c.AuthorEmail
	res.CommitterName = c.CommitterName
	res.CommitterEmail = c.CommitterEmail
	res.Files = c.Files
	res.Date = c.Date
	res.Ordinal = c.Ordinal
	res.Message = c.Message
	res.Parents = c.Parents
	res.Signed = c.Signed
	return
}

// Author returns either the author name (preference) or the email if not found
func (c Commit) Author() string {
	if c.AuthorName != "" {
		return c.AuthorName
	}
	return c.AuthorEmail
}
