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

	// IsDefault is true if this is default branch of the repo. Typically true for master.
	IsDefault bool

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

	// BehindDefaultCount is the number of commits on master that are not in this branch.
	// If the branch isMerges = true the value is 0.
	BehindDefaultCount int

	// AheadDefaultCount is the number of commits made to this branch, not including commits on master.
	// Same as len(Commits)
	AheadDefaultCount int
}

type Opts struct {
	Logger               logger.Logger
	IncludeDefaultBranch bool
	Concurrency          int
	RepoDir              string
	CommitGraph          *parentsgraph.Graph
	UseOrigin            bool
}

type Process struct {
	opts Opts

	defaultBranch nameAndHash

	reachableFromHead reachableFromHead
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

	var err error
	s.defaultBranch, err = getDefaultBranch(ctx, "git", s.opts.RepoDir)
	if err != nil {
		return err
	}

	if s.opts.IncludeDefaultBranch {
		res <- Branch{
			ID:        branchID(s.defaultBranch.Name, nil),
			Name:      s.defaultBranch.Name,
			IsDefault: true,
			Commits:   getAllCommits(s.opts.CommitGraph, s.defaultBranch.Commit),
		}
	}

	s.reachableFromHead = newReachableFromHead(s.opts.CommitGraph, s.defaultBranch.Commit)

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

func getAllCommits(gr *parentsgraph.Graph, head string) (res []string) {
	done := map[string]bool{}
	var rec func(string)
	rec = func(h string) {
		if done[h] {
			return
		}
		done[h] = true
		res = append(res, h)
		par, ok := gr.Parents[h]
		if !ok {
			panic("commit not found in tree")
		}
		// reverse order for better result ordering
		for i := len(par) - 1; i >= 0; i-- {
			rec(par[i])
		}
	}
	rec(head)
	reverseStrings(res)
	return
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

func getDefaultBranch(ctx context.Context, gitCommand string, repoDir string) (res nameAndHash, _ error) {
	name, err := headBranch(ctx, gitCommand, repoDir)
	if err != nil {
		return res, err
	}
	commit, err := headCommit(ctx, gitCommand, repoDir)
	if err != nil {
		return res, err
	}
	res.Name = name
	res.Commit = commit
	return res, nil
}

func headBranch(ctx context.Context, gitCommand string, repoDir string) (string, error) {
	data, err := execCommand(gitCommand, repoDir, []string{"rev-parse", "--abbrev-ref", "HEAD"})
	if err != nil {
		return "", err
	}
	res := strings.TrimSpace(string(data))
	if res == "HEAD" {
		return "", errors.New("cound not retrieve the name of the default branch")
	}
	return res, nil
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
	gr := s.opts.CommitGraph
	res := Branch{}
	res.Name = nameAndHash.Name
	defaultHead := s.defaultBranch.Commit
	res.Commits, res.BranchedFromCommits = branchCommits(gr, defaultHead, s.reachableFromHead, nameAndHash.Commit)
	res.ID = branchID(res.Name, res.BranchedFromCommits)
	if s.reachableFromHead[nameAndHash.Commit] {
		res.IsMerged = true
		res.MergeCommit = getMergeCommit(gr, s.reachableFromHead, nameAndHash.Commit)
	} else {
		if len(res.BranchedFromCommits) >= 1 {
			res.BehindDefaultCount = behindBranch(gr, s.reachableFromHead, nameAndHash.Commit, defaultHead)
		}
	}
	res.AheadDefaultCount = len(res.Commits)
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
