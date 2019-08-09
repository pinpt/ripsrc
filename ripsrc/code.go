package ripsrc

import (
	"context"
	"time"

	"github.com/pinpt/ripsrc/ripsrc/branchmeta"

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
	SHA     string
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

// Code returns code information using one record per file and commit
func (s *Ripsrc) Code(ctx context.Context, res chan BlameResult) error {
	defer close(res)

	res2 := make(chan CommitCode)
	done := make(chan bool)

	go func() {
		for r := range res2 {
			for f := range r.Files {
				res <- f
			}
		}
		done <- true
	}()

	err := s.CodeByCommit(ctx, res2)
	<-done
	if err != nil {
		return err
	}

	return nil
}

type CommitCode struct {
	SHA   string
	Files chan BlameResult
}

// CodeByCommit returns code information using one record per commit that includes records by file
func (s *Ripsrc) CodeByCommit(ctx context.Context, res chan CommitCode) error {
	defer close(res)

	err := s.prepareGitExec(ctx)
	if err != nil {
		return err
	}

	err = s.buildCommitGraph(ctx)
	if err != nil {
		return err
	}

	var wantedBranchRefs []string
	var wantedBranchNames []string

	if s.opts.CommitFromIncl != "" && s.opts.AllBranches {
		allBranches, err := branchmeta.Get(ctx, branchmeta.Opts{
			Logger:    s.opts.Logger,
			RepoDir:   s.opts.RepoDir,
			UseOrigin: s.opts.BranchesUseOrigin,
		})

		if err != nil {
			return err
		}

		deadline := s.opts.IncrementalIgnoreBranchesOlderThan
		if deadline.IsZero() {
			deadline = time.Now().Add(3 * 30 * 24 * time.Hour)
		}
		for _, b := range allBranches {
			if b.CommitCommitterTime.After(deadline) {
				wantedBranchRefs = append(wantedBranchRefs, b.Commit)
				wantedBranchNames = append(wantedBranchNames, b.Name)
			}
		}
	}
	if len(wantedBranchRefs) != 0 {
		s.opts.Logger.Debug("processing additional branches", "branches", wantedBranchNames)
	}

	err = s.getCommitInfo(ctx, wantedBranchRefs)
	if err != nil {
		return err
	}

	gitRes := make(chan process.Result)
	done := make(chan bool)
	go func() {
		for r1 := range gitRes {
			rc := CommitCode{}
			rc.SHA = r1.Commit
			rc.Files = make(chan BlameResult)

			rs, err := s.codeInfoFiles(r1)
			if err != nil {
				panic(err)
			}
			res <- rc
			for _, r := range rs {
				rc.Files <- r
			}
			close(rc.Files)
		}
		done <- true
	}()

	processOpts := process.Opts{
		Logger:           s.opts.Logger,
		RepoDir:          s.opts.RepoDir,
		CheckpointsDir:   s.opts.CheckpointsDir,
		NoStrictResume:   s.opts.NoStrictResume,
		CommitFromIncl:   s.opts.CommitFromIncl,
		AllBranches:      s.opts.AllBranches,
		ParentsGraph:     s.commitGraph,
		WantedBranchRefs: wantedBranchRefs,
	}
	gitProcessor := process.New(processOpts)
	err = gitProcessor.Run(gitRes)
	<-done

	if err != nil {
		return err
	}

	s.GitProcessTimings = gitProcessor.Timing()

	return nil
}

func (s *Ripsrc) CodeSlice(ctx context.Context) (res []BlameResult, _ error) {
	resChan := make(chan BlameResult)
	done := make(chan bool)
	go func() {
		for r := range resChan {
			res = append(res, r)
		}
		done <- true
	}()
	err := s.Code(ctx, resChan)
	<-done
	return res, err
}
