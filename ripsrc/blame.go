package ripsrc

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"

	"github.com/pinpt/ripsrc/ripsrc/branch"

	"github.com/pinpt/ripsrc/ripsrc/commitmeta"
	"github.com/pinpt/ripsrc/ripsrc/fileinfo"
	"github.com/pinpt/ripsrc/ripsrc/history3/process"
)

// Commit is a specific detail around a commit
//type Commit = commitmeta.Commit

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

	// OnDefaultBranch is set to true when commit is from the default branch. When AllBranches=true some commits could be from unmerged branches, in that case OnDefaultBranch=false
	OnDefaultBranch bool
}

func commitFromMeta(c commitmeta.Commit, onDefaultBranch bool) (res Commit) {
	res.OnDefaultBranch = onDefaultBranch

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

// CommitFile is a specific detail around a file in a commit
type CommitFile = commitmeta.CommitFile

// BlameResult holds details about the blame result
type BlameResult struct {
	Commit             Commit
	Language           string
	Filename           string
	Lines              []*BlameLine
	Size               int64
	Loc                int64
	Sloc               int64
	Comments           int64
	Blanks             int64
	Complexity         int64
	WeightedComplexity float64
	Skipped            string
	License            *License
	Status             CommitStatus
}

// BlameLine is a single line entry in blame
type BlameLine struct {
	Name    string
	Email   string
	Date    time.Time
	Comment bool
	Code    bool
	Blank   bool
}

// License holds details about detected license
type License = fileinfo.License

// CommitStatus is a commit status type
type CommitStatus = commitmeta.CommitStatus

const (
	// GitFileCommitStatusAdded is the added status
	GitFileCommitStatusAdded = commitmeta.GitFileCommitStatusAdded
	// GitFileCommitStatusModified is the modified status
	GitFileCommitStatusModified = commitmeta.GitFileCommitStatusModified
	// GitFileCommitStatusRemoved is the removed status
	GitFileCommitStatusRemoved = commitmeta.GitFileCommitStatusRemoved
)

func (s *Ripsrc) Blame(ctx context.Context, res chan BlameResult) error {
	defer close(res)

	err := s.prepareGitExec(ctx)
	if err != nil {
		return err
	}

	err = s.buildCommitGraph(ctx)
	if err != nil {
		return err
	}

	if s.opts.AllBranches {
		head, err := headCommit(ctx, gitCommand, s.opts.RepoDir)
		if err != nil {
			return err
		}

		s.defaultBranchCommits = branch.NewCommits(s.commitGraph, head)
	}

	err = s.getCommitInfo(ctx)
	if err != nil {
		panic(err)
	}

	gitRes := make(chan process.Result)
	done := make(chan bool)
	go func() {
		for r1 := range gitRes {
			rs, err := s.codeInfoFiles(r1)
			if err != nil {
				panic(err)
			}
			for _, r := range rs {
				res <- r
			}
		}
		done <- true
	}()

	processOpts := process.Opts{
		Logger:         s.opts.Logger,
		RepoDir:        s.opts.RepoDir,
		CheckpointsDir: s.opts.CheckpointsDir,
		NoStrictResume: s.opts.NoStrictResume,
		CommitFromIncl: s.opts.CommitFromIncl,
		AllBranches:    s.opts.AllBranches,
		ParentsGraph:   s.commitGraph,
	}
	gitProcessor := process.New(processOpts)
	err = gitProcessor.Run(gitRes)
	if err != nil {
		return err
	}

	<-done

	s.GitProcessTimings = gitProcessor.Timing()

	return nil
}

func (s *Ripsrc) BlameSlice(ctx context.Context) (res []BlameResult, _ error) {
	resChan := make(chan BlameResult)
	done := make(chan bool)
	go func() {
		for r := range resChan {
			res = append(res, r)
		}
		done <- true
	}()
	err := s.Blame(ctx, resChan)
	<-done
	return res, err
}

func headCommit(ctx context.Context, gitCommand string, repoDir string) (string, error) {
	data, err := execCommand(gitCommand, repoDir, []string{"rev-parse", "HEAD"})
	if err != nil {
		return "", err
	}
	res := strings.TrimSpace(string(data))
	if len(res) != 40 {
		return "", errors.New("unexpected output from git rev-parse HEAD")
	}
	return res, nil
}

func execCommand(command string, dir string, args []string) ([]byte, error) {
	out := bytes.NewBuffer(nil)
	c := exec.Command(command, args...)
	c.Dir = dir
	c.Stdout = out
	err := c.Run()
	if err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
