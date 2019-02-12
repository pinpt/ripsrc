package branches2

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"runtime"
	"strings"
	"sync"

	"github.com/pinpt/ripsrc/ripsrc/pkg/logger"

	"github.com/pinpt/ripsrc/ripsrc/parentsgraph"
)

// Branch contains information about the branch and commits on that branch.
type Branch struct {
	// ID of the branch hash(Commits[0] + Name)
	// More stable id that allows name reuse after deletes.
	ID string

	// Name of the branch
	Name string

	// IsMerged is true if this branch was merged into default branch
	IsMerged bool

	// MergeCommit is the hash of the merged commit. Set if IsMerged=true
	MergeCommit string

	// BranchedFromCommits are the branch points where the branch was originally created from master.
	// If the master was merged into branch multiple times, only oldest commit intersecting with master is used.
	// If the branch was started from multiple branches that are now the current master, this will contain array of them.
	// Normally this should be len(1)
	// In case the branch is completely separate and does not have commit commits with master it will be len(0)
	BranchedFromCommits []string

	// Commits is the list of hashes.
	// Does not include commits that were on master before it was created.
	// Could be empty.
	//
	// Commits[0] is the first commit on this branch
	// Commits[len-1] is the last commit on this branch
	Commits []string
}

type Opts struct {
	Logger      logger.Logger
	Concurrency int
	RepoDir     string
	CommitGraph *parentsgraph.Graph
	UseOrigin   bool
}

type Process struct {
	opts Opts

	branchCommitsCache *branchCommitsCache

	defaultHead string
}

func New(opts Opts) *Process {
	if opts.Concurrency == 0 {
		opts.Concurrency = runtime.NumCPU() * 2
	}
	s := &Process{}
	s.opts = opts
	return s
}

func (s *Process) Run(ctx context.Context, res chan Branch) error {
	defer close(res)

	err := s.markReachableFromHead(ctx)
	if err != nil {
		return err
	}

	nameAndHashes, err := s.getNamesAndHashes()
	if err != nil {
		return err
	}

	workCh := nameAndHashes.Chan()
	wg := sync.WaitGroup{}
	var lastErr error
	var lastErrMu sync.Mutex
	for i := 0; i < s.opts.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for nameAndHash := range workCh {
				lastErrMu.Lock()
				err := lastErr
				lastErrMu.Unlock()
				if err != nil {
					return
				}
				err = s.processBranch(ctx, nameAndHash, res)
				if err != nil {
					lastErrMu.Lock()
					lastErr = err
					lastErrMu.Unlock()
				}
			}
		}()
	}
	wg.Wait()
	return lastErr
}

func (s *Process) RunSlice(ctx context.Context) (res []Branch, _ error) {
	resChan := make(chan Branch)
	done := make(chan bool)
	go func() {
		for r := range resChan {
			res = append(res, r)
		}
		done <- true
	}()
	err := s.Run(ctx, resChan)
	<-done
	return res, err
}

func (s *Process) markReachableFromHead(ctx context.Context) error {
	head, err := headCommit(ctx, "git", s.opts.RepoDir)
	if err != nil {
		return err
	}
	s.defaultHead = head

	s.branchCommitsCache = newBranchCommitsCache(s.opts.CommitGraph, head)
	return nil
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

func (s *Process) processBranch(ctx context.Context, nameAndHash nameAndHash, resChan chan Branch) error {
	s.opts.Logger.Info("processing branch", "name", nameAndHash.Name, "commit", nameAndHash.Commit)
	res := Branch{}
	res.Name = nameAndHash.Name
	res.Commits, res.BranchedFromCommits = branchCommits(s.opts.CommitGraph, s.defaultHead, s.branchCommitsCache, nameAndHash.Commit)
	res.ID = branchID(res.Name, res.BranchedFromCommits)
	if s.branchCommitsCache.reachableFromHead[nameAndHash.Commit] {
		res.IsMerged = true
		res.MergeCommit = getMergeCommit(s.opts.CommitGraph, s.branchCommitsCache, nameAndHash.Commit)
	}
	resChan <- res
	return nil
}

func branchID(name string, branchedFrom []string) string {
	parts := []string{name}
	if len(branchedFrom) > 0 {
		parts = append(parts, branchedFrom[0])
	}
	return hash(strings.Join(parts, ""))
}

func hash(s string) string {
	h := sha256.New()
	_, err := h.Write([]byte(s))
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(h.Sum(nil))
}
