package ripsrc

import (
	"context"
	"errors"
	"time"

	"github.com/pinpt/ripsrc/ripsrc/commitmeta"
	"github.com/pinpt/ripsrc/ripsrc/fileinfo"
	"github.com/pinpt/ripsrc/ripsrc/gitexec"

	"github.com/pinpt/ripsrc/ripsrc/history3/process"
)

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

// Commit is a specific detail around a commit
type Commit = commitmeta.Commit

// CommitFile is a specific detail around a file in a commit
type CommitFile = commitmeta.CommitFile

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

type Ripper struct {
	commitMeta        map[string]Commit
	GitProcessTimings process.Timing
	CodeInfoTimings   *CodeInfoTimings

	fileInfo *fileinfo.Process
}

func New() *Ripper {
	s := &Ripper{}
	s.CodeInfoTimings = &CodeInfoTimings{}
	s.fileInfo = fileinfo.New()
	return s
}

type RipOpts struct {
	// CheckpointsDir is the directory to store incremental data cache for this repo
	// If empty, directory is created inside repoDir
	CheckpointsDir string
	// NoStrictResume forces incremental processing to avoid checking that it continues from the same commit in previously finished on. Since incrementals save a large number of previous commits, it works even starting on another commit.
	NoStrictResume bool
	CommitFromIncl string
}

var ErrNoHeadCommit = errors.New("can't get valid output from git rev-parse HEAD")

var gitCommand = "git"

func (s *Ripper) Rip(ctx context.Context, repoDir string, res chan BlameResult, opts *RipOpts) error {
	defer close(res)

	if opts == nil {
		opts = &RipOpts{}
	}

	err := gitexec.Prepare(ctx, gitCommand, repoDir)
	if err != nil {
		return err
	}

	err = s.getCommitInfo(ctx, repoDir, opts)
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
		RepoDir:        repoDir,
		CheckpointsDir: opts.CheckpointsDir,
		NoStrictResume: opts.NoStrictResume,
	}
	processOpts.CommitFromIncl = opts.CommitFromIncl
	gitProcessor := process.New(processOpts)
	err = gitProcessor.Run(gitRes)
	if err != nil {
		return err
	}

	<-done

	s.GitProcessTimings = gitProcessor.Timing()

	return nil
}

func (s *Ripper) RipSlice(ctx context.Context, repoDir string, opts *RipOpts) (res []BlameResult, _ error) {
	resChan := make(chan BlameResult)
	done := make(chan bool)
	go func() {
		for r := range resChan {
			res = append(res, r)
		}
		done <- true
	}()
	err := s.Rip(ctx, repoDir, resChan, opts)
	<-done
	return res, err
}
