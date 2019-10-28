package ripsrc

import (
	"context"
	"errors"

	"github.com/pinpt/ripsrc/ripsrc/branches2"
)

// Branch contains information about the branch and commits on that branch.
type Branch = branches2.Branch

func (s *Ripsrc) Branches(ctx context.Context, res chan Branch) error {
	defer close(res)
	if !s.opts.AllBranches {
		return errors.New("Branches call is only allowed when AllBranches=true")
	}

	err := s.prepareGitExec(ctx)
	if err != nil {
		return err
	}

	err = s.buildCommitGraph(ctx)
	if err != nil {
		return err
	}

	res2 := make(chan Branch)
	done := make(chan bool)
	go func() {
		for r := range res2 {
			res <- r
		}
		done <- true
	}()
	opts := branches2.Opts{}
	opts.Logger = s.opts.Logger
	opts.UseOrigin = s.opts.BranchesUseOrigin
	opts.CommitGraph = s.commitGraph
	opts.RepoDir = s.opts.RepoDir
	opts.IncludeDefaultBranch = true
	opts.PullRequestSHAs = s.opts.PullRequestSHAs
	pr := branches2.New(opts)
	err = pr.Run(ctx, res2)
	<-done
	return err
}

func (s *Ripsrc) BranchesSlice(ctx context.Context) (res []Branch, _ error) {
	resChan := make(chan Branch)
	done := make(chan bool)
	go func() {
		for r := range resChan {
			res = append(res, r)
		}
		done <- true
	}()
	err := s.Branches(ctx, resChan)
	<-done
	return res, err
}
