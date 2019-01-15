package ripsrc

import (
	"context"

	"github.com/pinpt/ripsrc/ripsrc/commitmeta"
)

func (s *Ripper) getCommitInfo(ctx context.Context, repoDir string, opts *RipOpts) error {
	copts := commitmeta.Opts{}
	copts.CommitFromIncl = opts.CommitFromIncl
	cm := commitmeta.New(repoDir, copts)
	res, err := cm.RunMap()
	if err != nil {
		return err
	}
	s.commitMeta = res
	return nil
}
