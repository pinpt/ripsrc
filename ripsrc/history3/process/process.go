package process

import (
	"context"
	"fmt"
	"io"

	"github.com/pinpt/ripsrc/ripsrc/gitexec"
	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/history3/process/graph"
	"github.com/pinpt/ripsrc/ripsrc/history3/process/parser"
)

type Process struct {
	opts       Opts
	gitCommand string

	// map[commitHash]map[filePath]*incblame.Blame
	repo          map[string]map[string]*incblame.Blame
	commitParents graph.Graph

	mergePartsCommit string
	mergeParts       []parser.Commit
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
	s.repo = map[string]map[string]*incblame.Blame{}
	return s
}

func (s *Process) Run(resChan chan Result) error {
	defer func() {
		close(resChan)
	}()

	r, err := s.gitLog()
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

	s.commitParents = graph.Graph{}

	for commit := range commits {
		s.processCommit(resChan, commit)
	}

	if len(s.mergeParts) > 0 {
		s.processGotMergeParts(resChan)
	}

	return nil
}

func (s *Process) processCommit(resChan chan Result, commit parser.Commit) {
	if len(s.mergeParts) > 0 {
		// continuing with merge
		if s.mergePartsCommit == commit.Hash {
			s.mergeParts = append(s.mergeParts, commit)
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
		s.commitParents[commit.Hash] = commit.Parents
		s.mergePartsCommit = commit.Hash
		s.mergeParts = append(s.mergeParts, commit)
		return
	}

	res, err := s.processRegularCommit(commit)
	if err != nil {
		panic(err)
	}
	resChan <- res
}

func (s *Process) processGotMergeParts(resChan chan Result) {
	res, err := s.processMergeCommit(s.mergeParts)
	if err != nil {
		panic(err)
	}
	s.mergeParts = nil
	resChan <- res
}

func (s *Process) processRegularCommit(commit parser.Commit) (res Result, _ error) {

	s.commitParents[commit.Hash] = commit.Parents

	res.Commit = commit.Hash
	res.Files = map[string]*incblame.Blame{}

	for _, ch := range commit.Changes {

		//fmt.Printf("%+v\n", string(ch.Diff))
		diff := incblame.Parse(ch.Diff)
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
		} else {
			// this is an empty file creation
			//if len(diff.Hunks) == 0 {
			//	panic(fmt.Errorf("no changes in commit: %v diff: %v", commit.Hash, string(ch.Diff)))
			//}
		}

		var parents []incblame.Blame
		if diff.PathPrev == "" {
			// file added in this commit, no parent blame for this file
		} else {
			for _, p := range commit.Parents {
				pb, ok := s.repo[p][diff.PathPrev]
				if !ok {
					// use empty file since merge will have line for it
					pb = &incblame.Blame{Commit: p}
					// file is not necessary in all parents if it was created in branch and then modified at merge
					// see merge_new_with_change test
					/*
						filesAtParent := []string{}
						for f := range s.repo[p] {
							filesAtParent = append(filesAtParent, f)
						}
						panic(fmt.Errorf("could not find file reference for commit %v parent %v, path %v, pathPrev %v, files at parent\n%v", commit.Hash, p, diff.Path, diff.PathPrev, filesAtParent))*/
				}

				parents = append(parents, *pb)
			}
		}
		var blame incblame.Blame
		if len(parents) == 0 {
			blame = incblame.Apply(incblame.Blame{}, diff, commit.Hash, diff.PathOrPrev())
		} else {
			blame = incblame.Apply(parents[0], diff, commit.Hash, diff.PathOrPrev())
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

func (s *Process) processMergeCommit(parts []parser.Commit) (res Result, _ error) {
	commitHash := parts[0].Hash
	parentHashes := parts[0].Parents
	parentCount := len(parentHashes)
	res.Commit = commitHash
	res.Files = map[string]*incblame.Blame{}

	// parse and organize all diffs for access
	diffs := map[string][]*incblame.Diff{}

	for parInd, part := range parts {
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
			// file removed, no longer need to keep blame reference, but showcase the file in res.Files using PathPrev
			pathPrev := k[len(deletedPrefix):]
			res.Files[pathPrev] = &incblame.Blame{Commit: commitHash}
			continue
		}
		// below k == new file path

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
					panic("parent blame not found")
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
		res.Files[k] = &blame
	}
	/*
		// for merge commits we need to use the most updated copy

		// get a list of all files in all parents
		files := map[string]bool{}
		for _, p := range commit.Parents {
			for f := range s.repo[p] {
				files[f] = true
			}
		}

		for f := range files {
			// was in the diff changes, nothing to do
			if _, ok := res.Files[f]; ok {
				continue
			}

			var candidates []*incblame.Blame
			for _, p := range commit.Parents {
				if b, ok := s.repo[p][f]; ok {
					candidates = append(candidates, b)
				}
			}

			// only one branch has the file
			if len(candidates) == 1 {
				// copy reference
				s.repoSave(commit.Hash, f, candidates[0])
				continue
			}

			if len(candidates) == 0 {
				panic("no file candidates")
			}

			// find common parent commit for all
			root := s.commitParents.LastCommonParent(commit.Parents)
			var res *incblame.Blame
			for _, c := range candidates {
				// unchanged
				if c.Commit == root {
					continue
				}
				res = c
			}
			if res == nil {
				// all are unchanged
				res = s.repo[root][f]
			}
			s.repoSave(commit.Hash, f, res)

		}
	*/
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

func (s *Process) gitLog() (io.ReadCloser, error) {
	args := []string{
		"log",
		"-p",
		"-m",
		"--reverse",
		"--no-abbrev-commit",
		"--pretty=format:!Hash: %H%n!Parents: %P",
	}

	ctx := context.Background()
	if s.opts.DisableCache {

		return gitexec.Exec(ctx, s.gitCommand, s.opts.RepoDir, args)
	}
	return gitexec.ExecWithCache(ctx, s.gitCommand, s.opts.RepoDir, args)
}
