package branches2

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"runtime"
	"strings"
	"sync"

	"github.com/pinpt/ripsrc/ripsrc/branchmeta"
	"github.com/pinpt/ripsrc/ripsrc/pkg/logger"

	"github.com/pinpt/ripsrc/ripsrc/parentsgraph"
)

// Branch contains information about the branch or pull request, for example commits.
type Branch struct {
	// ID of the branch hash(Commits[0] + Name)
	// More stable id that allows name reuse after deletes.
	// Only set for branches.
	BranchID string

	// Name of the branch
	Name string

	// IsPullRequest is set to true if this is created based on passed pull request sha instead of repo branch.
	IsPullRequest bool

	// HeadSHA is the sha of the head commit.
	HeadSHA string

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

	// FirstCommit is the first commit on this branch
	FirstCommit string
}

type Opts struct {
	// Logger outputs logs.
	Logger logger.Logger
	// IncludeDefaultBranch if default branch should be included in results.
	IncludeDefaultBranch bool
	// Concurrency sets number of goroutines that process tree.
	Concurrency int
	// RepoDir is location of git repo.
	RepoDir string
	// CommitGraph is the full graph of commits.
	CommitGraph *parentsgraph.Graph
	// UseOrigin set to true to use branches with origin/ prefix instead of default.
	UseOrigin bool
	// PullRequestSHAs is a list of custom sha references to process similar to branches returned from the repo.
	PullRequestSHAs []string
	// PullRequestsOnly skips branch data output, only using passed PullRequestSHAs
	PullRequestsOnly bool
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

func (s *Process) getFirstCommit() (string, error) {
	buf, err := execCommand("git", s.opts.RepoDir, []string{"rev-list", "--max-parents=0", "HEAD"})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(buf)), nil
}

func (s *Process) Run(ctx context.Context, res chan Branch) error {
	defer close(res)

	defaultBranch, err := branchmeta.GetDefault(ctx, s.opts.RepoDir)
	if err != nil {
		return err
	}
	s.defaultBranch = nameAndHash{Name: defaultBranch.Name, Commit: defaultBranch.Commit}

	if !s.opts.PullRequestsOnly && s.opts.IncludeDefaultBranch {
		firstCommit, err := s.getFirstCommit()
		if err != nil {
			return err
		}
		res <- Branch{
			BranchID:    branchID(s.defaultBranch.Name, nil),
			Name:        s.defaultBranch.Name,
			HeadSHA:     s.defaultBranch.Commit,
			IsDefault:   true,
			Commits:     getAllCommits(s.opts.CommitGraph, s.defaultBranch.Commit),
			FirstCommit: firstCommit,
		}
	}

	s.reachableFromHead = newReachableFromHead(s.opts.CommitGraph, s.defaultBranch.Commit)

	var namesAndHashes namesAndHashes

	if !s.opts.PullRequestsOnly {
		namesAndHashes, err = s.getNamesAndHashes()
		if err != nil {
			return err
		}
	}

	for _, sha := range uniqueStrings(s.opts.PullRequestSHAs) {
		namesAndHashes = append(namesAndHashes, nameAndHash{Commit: sha})
	}

	workCh := namesAndHashes.Chan()
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

func uniqueStrings(arr1 []string) (res []string) {
	m := map[string]bool{}
	for _, s := range arr1 {
		m[s] = true
	}
	for s := range m {
		res = append(res, s)
	}
	return
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

func (s *Process) processBranch(ctx context.Context, nameAndHash nameAndHash, resChan chan Branch) error {
	name := nameAndHash.Name
	if name == "" { // this is a passed pr
		s.opts.Logger.Info("processing pr", "head", nameAndHash.Commit)
	} else {
		s.opts.Logger.Info("processing branch", "name", nameAndHash.Name, "commit", nameAndHash.Commit)
	}
	gr := s.opts.CommitGraph

	res := Branch{}
	if name == "" {
		res.IsPullRequest = true

		// passed sha not found in the tree at all, skip it
		if _, ok := s.opts.CommitGraph.Parents[nameAndHash.Commit]; !ok {
			return nil
		}
	}
	res.HeadSHA = nameAndHash.Commit
	res.Name = name
	defaultHead := s.defaultBranch.Commit
	res.Commits, res.BranchedFromCommits = branchCommits(gr, defaultHead, s.reachableFromHead, nameAndHash.Commit)
	if name != "" {
		res.BranchID = branchID(res.Name, res.BranchedFromCommits)
	}
	if s.reachableFromHead[nameAndHash.Commit] {
		res.IsMerged = true
		res.MergeCommit = getMergeCommit(gr, s.reachableFromHead, nameAndHash.Commit)
	} else {
		if len(res.BranchedFromCommits) >= 1 {
			res.BehindDefaultCount = behindBranch(gr, s.reachableFromHead, nameAndHash.Commit, defaultHead)
		}
	}
	res.AheadDefaultCount = len(res.Commits)
	res.FirstCommit = res.Commits[0]
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
