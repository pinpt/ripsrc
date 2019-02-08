package ripsrc

import (
	"context"
	"time"

	"github.com/pinpt/ripsrc/ripsrc/commitmeta"
	"github.com/pinpt/ripsrc/ripsrc/fileinfo"
	"github.com/pinpt/ripsrc/ripsrc/history3/process"
)

// Commit is a specific detail around a commit
type Commit = commitmeta.Commit

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
	if err != nil {
		return nil, err
	}
	<-done
	return res, nil
}
