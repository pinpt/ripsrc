package process

import (
	"context"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/pinpt/ripsrc/ripsrc/gitblame2"

	"github.com/pinpt/ripsrc/ripsrc/history3/process/parentsp"

	"github.com/pinpt/ripsrc/ripsrc/gitexec"
	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/history3/process/graph"
	"github.com/pinpt/ripsrc/ripsrc/history3/process/parser"
)

type Process struct {
	opts       Opts
	gitCommand string

	// map[commitHash]map[filePath]*incblame.Blame
	repo map[string]map[string]*incblame.Blame

	commitParents      graph.Graph
	commitChildren     map[string][]string
	childrenProcessed  map[string]int
	maxLenOfStoredTree int

	mergePartsCommit string
	// map[parent_diffed]parser.Commit
	mergeParts map[string]parser.Commit

	timing *Timing
}

type Opts struct {
	RepoDir      string
	DisableCache bool
}

type Result struct {
	Commit string
	Files  map[string]*incblame.Blame
}

func New(opts Opts) *Process {
	s := &Process{}
	s.opts = opts
	s.gitCommand = "git"
	s.timing = &Timing{}
	s.repo = map[string]map[string]*incblame.Blame{}
	return s
}

func (s *Process) Timing() Timing {
	return *s.timing
}

func (s *Process) Run(resChan chan Result) error {
	defer func() {
		close(resChan)
	}()

	err := s.retrieveParents()
	if err != nil {
		return err
	}

	s.commitChildren = map[string][]string{}
	for commit, parents := range s.commitParents {
		for _, p := range parents {
			s.commitChildren[p] = append(s.commitChildren[p], commit)
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

	go func() {
		err := p.Run(commits)
		if err != nil {
			panic(err)
		}
	}()

	for commit := range commits {
		commit.Parents = s.commitParents[commit.Hash]
		s.processCommit(resChan, commit)
	}

	if len(s.mergeParts) > 0 {
		s.processGotMergeParts(resChan)
	}

	//fmt.Println("max len of stored tree", s.maxLenOfStoredTree)
	//fmt.Println("repo len", len(s.repo))

	return nil
}

func (s *Process) trimGraphAfterCommitProcessed(commit string) {
	parents := s.commitParents[commit]
	for _, p := range parents {
		s.childrenProcessed[p]++ // mark commit as processed
		siblings := s.commitChildren[p]
		if s.childrenProcessed[p] == len(siblings) {
			// done with parent, can delete it
			delete(s.repo, p)
		}
	}
	if len(s.repo) > s.maxLenOfStoredTree {
		s.maxLenOfStoredTree = len(s.repo)
	}
}

func (s *Process) retrieveParents() error {
	r, err := s.gitLogParents()
	if err != nil {
		return err
	}
	defer r.Close()

	pp := parentsp.New(r)
	res, err := pp.Run()
	if err != nil {
		return err
	}

	s.commitParents = graph.Graph(res)
	return nil
}

func (s *Process) processCommit(resChan chan Result, commit parser.Commit) {
	if len(s.mergeParts) > 0 {
		// continuing with merge
		if s.mergePartsCommit == commit.Hash {
			s.mergeParts[commit.MergeDiffFrom] = commit
			// still same
			return
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
		return
	}

	res, err := s.processRegularCommit(commit)
	if err != nil {
		panic(err)
	}
	s.trimGraphAfterCommitProcessed(commit.Hash)
	resChan <- res
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

func (s *Process) processRegularCommit(commit parser.Commit) (res Result, _ error) {
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
				s.repoSave(commit.Hash, p, bl)
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
				pb, ok := s.repo[parent][diff.PathPrev]
				if !ok {
					filesInParent := []string{}
					for f := range s.repo[parent] {
						filesInParent = append(filesInParent, f)
					}
					panic(fmt.Errorf("regular commit %v, file rename from %v to %v, file not found in parent %v files in parent: %v", commit.Hash, diff.PathPrev, diff.Path, parent, filesInParent))
				}
				if pb.IsBinary {
					s.repoSave(commit.Hash, diff.Path, pb)
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
				pc, ok := s.repo[parentHash]
				if !ok {
					panic("parent commit not found")
				}
				pb, ok := pc[diff.PathPrev]
				// file may not be in parent if this is create
				if ok {
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

		s.repoSave(commit.Hash, diff.Path, &blame)
		res.Files[diff.Path] = &blame
	}

	if len(commit.Parents) == 0 {
		// no need to copy files from prev
		return
	}

	// copy unchanged from prev
	p := commit.Parents[0]
	files := s.repo[p]
	for path, blame := range files {
		// was in the diff changes, nothing to do
		if _, ok := res.Files[path]; ok {
			continue
		}

		// copy reference
		s.repoSave(commit.Hash, path, blame)
	}

	return
}

const deletedPrefix = "@@@del@@@"

func (s *Process) processMergeCommit(commitHash string, parts map[string]parser.Commit) (res Result, _ error) {
	start := time.Now()
	defer func() {
		dur := time.Since(start)
		s.timing.UpdateSlowestCommitsWith(commitHash, dur)
		s.timing.MergesTime += dur
		s.timing.MergesCount++
	}()

	//fmt.Println("processing merge commit", commitHash)

	parentHashes := s.commitParents[commitHash]
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
			pb, ok := s.repo[parent][diff.PathPrev]
			if !ok {
				panic("parent not found")
			}
			if pb.IsBinary {
				binParentsWithDiffs++
			}
		}

		// do not try to resolve the diffs for binary files in merge commits
		if binaryDiffs != 0 || binParentsWithDiffs != 0 {
			bl := incblame.BlameBinaryFile(commitHash)
			s.repoSave(commitHash, k, bl)
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
				pb, ok := s.repo[parent][k]
				if ok {
					// exacly the same as parent, no changes
					s.repoSave(commitHash, k, pb)
					continue EACHFILE
				}
			}
		}

		parents := []incblame.Blame{}
		for i, diff := range diffs {
			if diff == nil {
				// no change use prev
				parentHash := parentHashes[i]
				parentBlame, ok := s.repo[parentHash][k]
				if !ok {
					panic(fmt.Errorf("merge: no change for file recorded, but parent does not contain blame information file:%v merge:%v parent:%v", k, commitHash, parentHash))
				}
				parents = append(parents, *parentBlame)
				continue
			}

			pathPrev := diff.PathPrev
			if pathPrev == "" {
				// this is create, no parent blame
				parents = append(parents, incblame.Blame{})
				//fmt.Println("create for paretnt", i)
				continue
			}

			parentHash := parentHashes[i]
			parentBlame, ok := s.repo[parentHash][pathPrev]
			if !ok {
				panic("parent blame not found")
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
		s.repoSave(commitHash, k, &blame)

		// only showing deletes and files changed in merge comparent to at least one parent
		res.Files[k] = &blame
	}

	// for merge commits we need to use the most updated copy

	// get a list of all files in all parents
	files = map[string]bool{}
	for _, p := range parentHashes {
		for f := range s.repo[p] {
			files[f] = true
		}
	}

	root := ""

	for f := range files {
		// already added above
		if _, ok := s.repo[commitHash][f]; ok {
			continue
		}

		var candidates []*incblame.Blame
		for _, p := range parentHashes {
			if b, ok := s.repo[p][f]; ok {
				candidates = append(candidates, b)
			}
		}

		// only one branch has the file
		if len(candidates) == 1 {
			// copy reference
			s.repoSave(commitHash, f, candidates[0])
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
				root = s.commitParents.LastCommonParent(parentHashes)
				dur := time.Since(ts)
				if dur > time.Second {
					fmt.Printf("took %v to find last common parent for %v res: %v", dur, parentHashes, root)
				}
			}*/

		var res *incblame.Blame
		for _, c := range candidates {
			// unchanged
			//if c.Commit == root {
			//	continue
			//}
			res = c
		}
		if res == nil {
			// all are unchanged
			res = s.repo[root][f]
		}
		s.repoSave(commitHash, f, res)

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
		l2 := incblame.Line{}
		l2.Commit = l.CommitHash
		l2.Line = []byte(l.Content)
		res.Lines = append(res.Lines, l2)
	}
	return
}

func (s *Process) repoSave(commit, path string, blame *incblame.Blame) {
	if _, ok := s.repo[commit]; !ok {
		s.repo[commit] = map[string]*incblame.Blame{}
	}
	s.repo[commit][path] = blame
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

func (s *Process) gitLogParents() (io.ReadCloser, error) {
	args := []string{
		"log",
		"-m",
		"--reverse",
		"--no-abbrev-commit",
		"--pretty=format:%H@%P",
	}

	ctx := context.Background()
	if s.opts.DisableCache {

		return gitexec.Exec(ctx, s.gitCommand, s.opts.RepoDir, args)
	}
	return gitexec.ExecWithCache(ctx, s.gitCommand, s.opts.RepoDir, args)
}

func (s *Process) gitLogPatches() (io.ReadCloser, error) {
	args := []string{
		"log",
		"-p",
		"-m",
		"--date-order",
		"--reverse",
		"--no-abbrev-commit",
		"--pretty=short",
	}

	ctx := context.Background()
	if s.opts.DisableCache {

		return gitexec.Exec(ctx, s.gitCommand, s.opts.RepoDir, args)
	}
	return gitexec.ExecWithCache(ctx, s.gitCommand, s.opts.RepoDir, args)
}
