package ripsrc

import (
	"context"

	"github.com/pinpt/ripsrc/ripsrc/commitmeta"
)

func (s *Ripper) getCommitInfo(ctx context.Context, repoDir string) error {
	cm := commitmeta.New(repoDir)
	res, err := cm.RunMap()
	if err != nil {
		return err
	}
	s.commitMeta = res
	return nil
}
