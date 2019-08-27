package process

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/pinpt/ripsrc/ripsrc/parentsgraph"

	"github.com/pinpt/ripsrc/ripsrc/gitblame2"
	"github.com/pinpt/ripsrc/ripsrc/pkg/logger"

	"github.com/pinpt/ripsrc/ripsrc/history3/process/repo"

	"github.com/pinpt/ripsrc/ripsrc/gitexec"
	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/history3/process/parser"
)

type Process struct {
	opts       Opts
	gitCommand string

	graph *parentsgraph.Graph

	childrenProcessed  map[string]int
	maxLenOfStoredTree int

	mergePartsCommit string
	// map[parent_diffed]parser.Commit
	mergeParts map[string]parser.Commit

	timing *Timing

	repo     repo.Repo
	unloader *repo.Unloader

	checkpointsDir string

	lastProcessedCommitHash string
}

type Opts struct {
	Logger  logger.Logger
	RepoDir string

	// CheckpointsDir is the directory to store incremental data cache for this repo
	// If empty, directory is created inside repoDir
	CheckpointsDir string

	// NoStrictResume forces incremental processing to avoid checking that it continues from the same commit in previously finished on. Since incrementals save a large number of previous commits, it works even starting on another commit.
	NoStrictResume bool

	// CommitFromIncl process starting from this commit (including this commit).
	CommitFromIncl string

	// CommitFromMakeNonIncl by default we start from passed commit and include it. Set CommitFromMakeNonIncl to true to avoid returning it, and skipping reading/writing checkpoint.
	CommitFromMakeNonIncl bool

	// DisableCache is unused.
	DisableCache bool

	// AllBranches set to true to process all branches. If false, processes commits starting from HEAD only.
	AllBranches bool

	// WantedBranchRefs filter branches.  When CommitFromIncl and AllBranches is set this is required.
	WantedBranchRefs []string

	// ParentsGraph is optional graph of commits. Pass to reuse, if not passed will be created.
	ParentsGraph *parentsgraph.Graph
}

type Result struct {
	Commit string
	Files  map[string]*incblame.Blame
}

func New(opts Opts) *Process {
	s := &Process{}

	if opts.Logger == nil {
		opts.Logger = logger.NewDefaultLogger(os.Stdout)
	}
	s.opts = opts
	s.gitCommand = "git"

	s.timing = &Timing{}

	if opts.CheckpointsDir != "" {
		s.checkpointsDir = filepath.Join(opts.CheckpointsDir, "pp-git-cache")
	} else {
		s.checkpointsDir = filepath.Join(opts.RepoDir, "pp-git-cache")
	}

	return s
}

func (s *Process) Timing() Timing {
	return *s.timing
}

func (s *Process) initCheckpoints() error {

	if s.opts.CommitFromIncl == "" {
		s.repo = repo.New()
	} else {
		expectedCommit := ""
		if s.opts.NoStrictResume {
			// validation disabled
		} else {
			expectedCommit = s.opts.CommitFromIncl
		}
		reader := repo.NewCheckpointReader(s.opts.Logger)
		r, err := reader.Read(s.checkpointsDir, expectedCommit)
		if err != nil {
			panic(err)
		}
		s.repo = r
	}

	s.unloader = repo.NewUnloader(s.repo)
	return nil
}

func (s *Process) Run(resChan chan Result) error {
	defer func() {
		close(resChan)
	}()

	if s.opts.ParentsGraph != nil {
		s.graph = s.opts.ParentsGraph
	} else {
		s.graph = parentsgraph.New(parentsgraph.Opts{
			RepoDir:     s.opts.RepoDir,
			AllBranches: s.opts.AllBranches,
			Logger:      s.opts.Logger,
		})
		err := s.graph.Read()
		if err != nil {
			return err
		}
	}

	s.childrenProcessed = map[string]int{}

	r, err := s.gitLogPatches()
	if err != nil {
		return err
	}

	defer r.Close()

	commits := make(chan parser.Commit)
	p := parser.New(r)

	done := make(chan bool)

	go func() {
		defer func() {
			done <- true
		}()
		err := p.Run(commits)
		if err != nil {
			panic(err)
		}
	}()

	drainAndExit := func() {
		for range commits {
		}
		<-done
	}

	i := 0
	for commit := range commits {
		if i == 0 {
			err := s.initCheckpoints()
			if err != nil {
				drainAndExit()
				return err
			}
		}
		i++
		commit.Parents = s.graph.Parents[commit.Hash]
		err := s.processCommit(resChan, commit)
		if err != nil {
			drainAndExit()
			return err
		}
	}

	if len(s.mergeParts) > 0 {
		s.processGotMergeParts(resChan)
	}

	if i == 0 {
		// there were no items in log, happens when last processed commit was in a branch that is no longer recent and is skipped in incremental
		// no need to write checkpoints
		<-done
		return nil
	}

	writer := repo.NewCheckpointWriter(s.opts.Logger)
	err = writer.Write(s.repo, s.checkpointsDir, s.lastProcessedCommitHash)
	if err != nil {
		<-done
		return err
	}

	//fmt.Println("max len of stored tree", s.maxLenOfStoredTree)
	//fmt.Println("repo len", len(s.repo))
	<-done
	return nil
}

func (s *Process) trimGraphAfterCommitProcessed(commit string) {
	parents := s.graph.Parents[commit]
	for _, p := range parents {
		s.childrenProcessed[p]++ // mark commit as processed
		siblings := s.graph.Children[p]
		if s.childrenProcessed[p] == len(siblings) {
			// done with parent, can delete it
			s.unloader.Unload(p)
		}
	}
	//commitsInMemory := s.repo.CommitsInMemory()
	commitsInMemory := len(s.repo)
	if commitsInMemory > s.maxLenOfStoredTree {
		s.maxLenOfStoredTree = commitsInMemory
	}
}

func (s *Process) processCommit(resChan chan Result, commit parser.Commit) error {
	if len(s.mergeParts) > 0 {
		// continuing with merge
		if s.mergePartsCommit == commit.Hash {
			s.mergeParts[commit.MergeDiffFrom] = commit
			// still same
			return nil
		} else {
			// finished
			s.processGotMergeParts(resChan)
			// new commit
			// continue below
		}
	}

	if len(commit.Parents) > 1 { // this is a merge
		s.mergePartsCommit = commit.Hash
		s.mergeParts = map[string]parser.Commit{}
		s.mergeParts[commit.MergeDiffFrom] = commit
		return nil
	}

	res, err := s.processRegularCommit(commit)
	if err != nil {
		return err
	}
	s.trimGraphAfterCommitProcessed(commit.Hash)
	resChan <- res
	return nil
}

func (s *Process) processGotMergeParts(resChan chan Result) {
	res, err := s.processMergeCommit(s.mergePartsCommit, s.mergeParts)
	if err != nil {
		panic(err)
	}
	s.trimGraphAfterCommitProcessed(s.mergePartsCommit)
	s.mergeParts = nil
	resChan <- res
}

type Timing struct {
	RegularCommitsCount int
	RegularCommitsTime  time.Duration
	MergesCount         int
	MergesTime          time.Duration
	SlowestCommits      []CommitWithDuration
}

type CommitWithDuration struct {
	Commit   string
	Duration time.Duration
}

const maxSlowestCommits = 10

func (s *Timing) UpdateSlowestCommitsWith(commit string, d time.Duration) {
	s.SlowestCommits = append(s.SlowestCommits, CommitWithDuration{Commit: commit, Duration: d})
	sort.Slice(s.SlowestCommits, func(i, j int) bool {
		a := s.SlowestCommits[i]
		b := s.SlowestCommits[j]
		return a.Duration > b.Duration
	})
	if len(s.SlowestCommits) > maxSlowestCommits {
		s.SlowestCommits = s.SlowestCommits[0:maxSlowestCommits]
	}
}

func (s *Timing) SlowestCommitsDur() (res time.Duration) {
	for _, c := range s.SlowestCommits {
		res += c.Duration
	}
	return
}

/*
func (s *Timing) Stats() map[string]interface{} {
	return map[string]interface{}{
		"TotalRegularCommit": s.TotalRegularCommit,
		"TotalMerges":        s.TotalMerges,
		"SlowestCommits":     s.SlowestCommits,
		"SlowestCommitsDur":  s.SlowestCommitsDur(),
	}
}*/

func (s *Timing) OutputStats(wr io.Writer) {
	fmt.Fprintln(wr, "git processor timing")
	fmt.Fprintln(wr, "regular commits", s.RegularCommitsCount)
	fmt.Fprintln(wr, "time in regular commits", s.RegularCommitsTime)
	fmt.Fprintln(wr, "merges", s.MergesCount)
	fmt.Fprintln(wr, "time in merges commits", s.MergesTime)
	fmt.Fprintf(wr, "time in %v slowest commits %v\n", len(s.SlowestCommits), s.SlowestCommitsDur())
	fmt.Fprintln(wr, "slowest commits")
	for _, c := range s.SlowestCommits {
		fmt.Fprintf(wr, "%v %v\n", c.Commit, c.Duration)
	}

}

func (s *Process) processRegularCommit(commit parser.Commit) (res Result, rerr error) {
	s.lastProcessedCommitHash = commit.Hash

	start := time.Now()
	defer func() {
		dur := time.Since(start)
		s.timing.UpdateSlowestCommitsWith(commit.Hash, dur)
		s.timing.RegularCommitsTime += dur
		s.timing.RegularCommitsCount++
	}()

	if len(commit.Parents) > 1 {
		panic("not a regular commit")
	}
	// note that commit exists (important for empty commits)
	s.repo.AddCommit(commit.Hash)

	//fmt.Println("processing regular commit", commit.Hash)
	res.Commit = commit.Hash
	res.Files = map[string]*incblame.Blame{}

	for _, ch := range commit.Changes {

		//fmt.Printf("%+v\n", string(ch.Diff))
		diff := incblame.Parse(ch.Diff)

		if diff.IsBinary {
			// do not keep actual lines, but show in result
			bl := incblame.BlameBinaryFile(commit.Hash)

			if diff.Path == "" {
				p := diff.PathPrev
				res.Files[p] = bl
				// removal
			} else {
				p := diff.Path
				res.Files[p] = bl
				s.repo[commit.Hash][p] = bl
			}
			continue
		}

		//fmt.Printf("diff %+v\n", diff)
		if diff.Path == "" {
			// file removed, no longer need to keep blame reference, but showcase the file in res.Files using PathPrev
			res.Files[diff.PathPrev] = &incblame.Blame{Commit: commit.Hash}
			continue
		}

		// TODO: test renames here as well

		if diff.Path == "" {
			panic(fmt.Errorf("commit diff does not specify Path: %v diff: %v", commit.Hash, string(ch.Diff)))
		}

		// this is a rename
		if diff.PathPrev != "" && diff.PathPrev != diff.Path {
			if len(commit.Parents) != 1 {
				panic(fmt.Errorf("rename with more than 1 parent (merge) not supported: %v diff: %v", commit.Hash, string(ch.Diff)))
			}
			// rename with no patch
			if len(diff.Hunks) == 0 {
				parent := commit.Parents[0]
				pb, err := s.repo.GetFileMust(parent, diff.PathPrev)
				if err != nil {
					rerr = fmt.Errorf("could not get parent file for rename: %v err: %v", commit.Hash, err)
					return
				}
				if pb.IsBinary {
					s.repo[commit.Hash][diff.Path] = pb
					res.Files[diff.Path] = pb
					continue
				}
			}

		} else {
			// this is an empty file creation
			//if len(diff.Hunks) == 0 {
			//	panic(fmt.Errorf("no changes in commit: %v diff: %v", commit.Hash, string(ch.Diff)))
			//}
		}

		var parentBlame *incblame.Blame

		if diff.PathPrev == "" {
			// file added in this commit, no parent blame for this file
		} else {
			switch len(commit.Parents) {
			case 0: // initial commit, no parent
			case 1: // regular commit
				parentHash := commit.Parents[0]
				pb := s.repo.GetFileOptional(parentHash, diff.PathPrev)
				// file may not be in parent if this is create
				if pb != nil {
					parentBlame = pb
				}
			case 2: // merge
				panic("merge passed to regular commit processing")

			}
		}

		var blame incblame.Blame
		if parentBlame == nil {
			blame = incblame.Apply(incblame.Blame{}, diff, commit.Hash, diff.PathOrPrev())
		} else {
			if parentBlame.IsBinary {
				bl, err := s.slowGitBlame(commit.Hash, diff.Path)
				if err != nil {
					return res, err
				}
				blame = bl
			} else {
				blame = incblame.Apply(*parentBlame, diff, commit.Hash, diff.PathOrPrev())
			}
		}
		s.repo[commit.Hash][diff.Path] = &blame
		res.Files[diff.Path] = &blame
	}

	if len(commit.Parents) == 0 {
		// no need to copy files from prev
		return
	}

	// copy unchanged from prev
	p := commit.Parents[0]
	files := s.repo.GetCommitMust(p)
	for fp := range files {
		// was in the diff changes, nothing to do
		if _, ok := res.Files[fp]; ok {
			continue
		}
		blame, err := s.repo.GetFileMust(p, fp)
		if err != nil {
			rerr = fmt.Errorf("could not get parent file for unchanged: %v err: %v", commit.Hash, err)
			return
		}
		// copy reference
		s.repo[commit.Hash][fp] = blame
	}

	return
}

const deletedPrefix = "@@@del@@@"

func (s *Process) processMergeCommit(commitHash string, parts map[string]parser.Commit) (res Result, rerr error) {
	s.lastProcessedCommitHash = commitHash

	start := time.Now()
	defer func() {
		dur := time.Since(start)
		s.timing.UpdateSlowestCommitsWith(commitHash, dur)
		s.timing.MergesTime += dur
		s.timing.MergesCount++
	}()

	// note that commit exists (important for empty commits)
	s.repo.AddCommit(commitHash)

	//fmt.Println("processing merge commit", commitHash)

	parentHashes := s.graph.Parents[commitHash]
	parentCount := len(parentHashes)

	res.Commit = commitHash
	res.Files = map[string]*incblame.Blame{}

	// parse and organize all diffs for access
	diffs := map[string][]*incblame.Diff{}

	hashToParOrd := map[string]int{}
	for i, h := range parentHashes {
		hashToParOrd[h] = i
	}

	for parHash, part := range parts {
		for _, ch := range part.Changes {
			diff := incblame.Parse(ch.Diff)
			key := ""
			if diff.Path != "" {
				key = diff.Path
			} else {
				key = deletedPrefix + diff.PathPrev
			}
			par, ok := diffs[key]
			if !ok {
				par = make([]*incblame.Diff, parentCount, parentCount)
				diffs[key] = par
			}
			parInd := hashToParOrd[parHash]
			par[parInd] = &diff
		}
	}

	// get a list of all files
	files := map[string]bool{}
	for k := range diffs {
		files[k] = true
	}

	// process all files

EACHFILE:
	for k := range files {
		diffs := diffs[k]

		isDelete := true
		for _, diff := range diffs {
			if diff != nil && diff.Path != "" {
				isDelete = false
			}
		}

		//fmt.Println("diffs")
		//for i, d := range diffs {
		//	fmt.Println(i, d)
		//}

		if isDelete {
			// only showing deletes and files changed in merge comparent to at least one parent
			pathPrev := k[len(deletedPrefix):]
			res.Files[pathPrev] = &incblame.Blame{Commit: commitHash}
			continue
		}

		// below k == new file path

		binaryDiffs := 0
		for _, diff := range diffs {
			if diff == nil {
				continue
			}
			if diff.IsBinary {
				binaryDiffs++
			}
		}

		binParentsWithDiffs := 0
		for i, diff := range diffs {
			if diff == nil {
				continue
			}
			if diff.PathPrev == "" {
				// create
				continue
			}
			parent := parentHashes[i]
			pb, err := s.repo.GetFileMust(parent, diff.PathPrev)
			if err != nil {
				rerr = fmt.Errorf("could not get file for merge bin parent. merge: %v %v", commitHash, err)
				return
			}
			if pb.IsBinary {
				binParentsWithDiffs++
			}
		}

		// do not try to resolve the diffs for binary files in merge commits
		if binaryDiffs != 0 || binParentsWithDiffs != 0 {
			bl := incblame.BlameBinaryFile(commitHash)
			s.repo[commitHash][k] = bl
			res.Files[k] = bl
			continue
		}
		/*
			// file is a binary
			if binaryDiffs == validDiffs {
				bl := incblame.BlameBinaryFile(commitHash)
				s.repoSave(commitHash, k, bl)
				res.Files[k] = bl
				continue
			}

			// file is not a binary but one of the parents was a binary, need to use a regular git blame
			if binaryParents != 0 {
				bl, err := s.slowGitBlame(commitHash, k)
				if err != nil {
					return res, err
				}
				s.repoSave(commitHash, k, &bl)
				res.Files[k] = &bl
				continue
			}*/

		for i, diff := range diffs {
			if diff == nil {
				// same as parent
				parent := parentHashes[i]
				pb := s.repo.GetFileOptional(parent, k)
				if pb != nil {
					// exacly the same as parent, no changes
					s.repo[commitHash][k] = pb
					continue EACHFILE
				}
			}
		}

		parents := []incblame.Blame{}
		for i, diff := range diffs {
			if diff == nil {
				// no change use prev
				parentHash := parentHashes[i]
				parentBlame := s.repo.GetFileOptional(parentHash, k)
				if parentBlame == nil {
					panic(fmt.Errorf("merge: no change for file recorded, but parent does not contain file:%v merge commit:%v parent:%v", k, commitHash, parentHash))
				}
				parents = append(parents, *parentBlame)
				continue
			}

			pathPrev := diff.PathPrev
			if pathPrev == "" {
				// this is create, no parent blame
				parents = append(parents, incblame.Blame{})
				continue
			}

			parentHash := parentHashes[i]
			parentBlame, err := s.repo.GetFileMust(parentHash, pathPrev)
			if err != nil {
				rerr = fmt.Errorf("could not get file for unchanged case1 merge file. merge: %v %v", commitHash, err)
				return
			}
			parents = append(parents, *parentBlame)
		}
		//fmt.Println("path", k)
		diffs2 := []incblame.Diff{}
		for _, ob := range diffs {
			if ob == nil {
				ob = &incblame.Diff{}
			}
			diffs2 = append(diffs2, *ob)
		}
		blame := incblame.ApplyMerge(parents, diffs2, commitHash, k)
		s.repo[commitHash][k] = &blame

		// only showing deletes and files changed in merge comparent to at least one parent
		res.Files[k] = &blame
	}

	// for merge commits we need to use the most updated copy

	// get a list of all files in all parents
	files = map[string]bool{}
	for _, p := range parentHashes {
		filesInCommit := s.repo.GetCommitMust(p)
		for f := range filesInCommit {
			files[f] = true
		}
	}

	root := ""

	for f := range files {
		alreadyAddedAbove := false
		{
			bl := s.repo.GetFileOptional(commitHash, f)
			if bl != nil {
				alreadyAddedAbove = true
			}

		}

		if alreadyAddedAbove {
			continue
		}

		var candidates []*incblame.Blame
		for _, p := range parentHashes {
			bl := s.repo.GetFileOptional(p, f)
			if bl != nil {
				candidates = append(candidates, bl)
			}
		}

		// only one branch has the file
		if len(candidates) == 1 {
			// copy reference
			s.repo[commitHash][f] = candidates[0]
			continue
		}

		if len(candidates) == 0 {
			panic("no file candidates")
		}

		// TODO: if more than one candidate we pick at random right now
		// Need to check if this is correct? If no change at merge to any that means they are all the same?
		// Or we need to check the last common parent and see? This was added in the previous design so possible is not needed anymore.

		/*
			if root == "" {
				// TODO: this is not covered by unit tests
				ts := time.Now()
				// find common parent commit for all
				root = s.graph.Parents.LastCommonParent(parentHashes)
				dur := time.Since(ts)
				if dur > time.Second {
					fmt.Printf("took %v to find last common parent for %v res: %v", dur, parentHashes, root)
				}
			}*/

		var res2 *incblame.Blame
		for _, c := range candidates {
			// unchanged
			//if c.Commit == root {
			//	continue
			//}
			res2 = c
		}
		if res2 == nil {
			var err error
			// all are unchanged
			res2, err = s.repo.GetFileMust(root, f)
			if err != nil {
				rerr = fmt.Errorf("could not get file for unchanged case2 merge file. merge: %v %v", commitHash, err)
				return
			}
		}
		s.repo[commitHash][f] = res2

	}

	return
}

func (s *Process) slowGitBlame(commitHash string, filePath string) (res incblame.Blame, _ error) {
	bl, err := gitblame2.Run(s.opts.RepoDir, commitHash, filePath)
	//fmt.Println("running regular blame for file switching from bin mode to regular")
	if err != nil {
		return res, err
	}
	res.Commit = commitHash
	for _, l := range bl.Lines {
		l2 := &incblame.Line{}
		l2.Commit = l.CommitHash
		l2.Line = []byte(l.Content)
		res.Lines = append(res.Lines, l2)
	}
	return
}

func (s *Process) RunGetAll() (_ []Result, err error) {
	res := make(chan Result)
	done := make(chan bool)
	go func() {
		err = s.Run(res)
		done <- true
	}()
	var res2 []Result
	for r := range res {
		res2 = append(res2, r)
	}
	<-done
	return res2, err
}

func (s *Process) gitLogPatches() (io.ReadCloser, error) {
	// empty file at temp location to set an empty attributesFile
	f, err := ioutil.TempFile("", "ripsrc")
	if err != nil {
		return nil, err
	}
	err = f.Close()
	if err != nil {
		return nil, err
	}

	args := []string{
		"-c", "core.attributesFile=" + f.Name(),
		"-c", "diff.renameLimit=10000",
		"log",
		"-p",
		"-m",
		"--date-order",
		"--reverse",
		"--no-abbrev-commit",
		"--pretty=short",
	}

	if s.opts.CommitFromIncl != "" {
		if s.opts.AllBranches {
			for _, c := range s.opts.WantedBranchRefs {
				args = append(args, c)
			}
		}
		pf := ""
		if s.opts.CommitFromMakeNonIncl {
			pf = "..HEAD"
		} else {
			pf = "^..HEAD"
		}
		args = append(args, s.opts.CommitFromIncl+pf)
	} else {
		if s.opts.AllBranches {
			args = append(args, "--all")
		}
	}

	ctx := context.Background()
	//if s.opts.DisableCache {

	return gitexec.ExecPiped(ctx, s.gitCommand, s.opts.RepoDir, args)
	//}
	//return gitexec.ExecWithCache(ctx, s.gitCommand, s.opts.RepoDir, args)
}
