package ripsrc

import (
	"context"

	"github.com/pinpt/ripsrc/ripsrc/commitmeta"
)

func (s *Ripsrc) getCommitInfo(ctx context.Context) error {
	copts := commitmeta.Opts{}
	copts.CommitFromIncl = s.opts.CommitFromIncl
	copts.AllBranches = s.opts.AllBranches
	copts.Logger = s.opts.Logger
	cm := commitmeta.New(s.opts.RepoDir, copts)
	res, err := cm.RunMap()
	if err != nil {
		return err
	}
	s.commitMeta = res
	return nil
}
