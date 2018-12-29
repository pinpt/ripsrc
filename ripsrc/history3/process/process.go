package process

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/history3/process/parser"
)

type Process struct {
	repoDir    string
	gitCommand string

	// map[commitHash]map[filePath]*incblame.Blame
	repo map[string]map[string]*incblame.Blame
}

type Result struct {
	Commit string
	Files  map[string]*incblame.Blame
}

func New(repoDir string) *Process {
	s := &Process{}
	s.repoDir = repoDir
	s.gitCommand = "git"
	s.repo = map[string]map[string]*incblame.Blame{}
	return s
}

func (s *Process) Run(resChan chan Result) error {
	r, err := s.gitLog()
	if err != nil {
		return err
	}

	commits := make(chan parser.Commit)
	p := parser.New(r)
	go func() {
		err := p.Run(commits)
		if err != nil {
			panic(err)
		}

	}()
	for commit := range commits {
		res := Result{}
		res.Commit = commit.Hash
		res.Files = map[string]*incblame.Blame{}

		for _, ch := range commit.Changes {
			diff := incblame.Parse(ch.Diff)
			if diff.Path == "" {
				// file removed
				res.Files[diff.PathPrev] = &incblame.Blame{Commit: commit.Hash}
				continue
			}
			// TODO: test renames here as well

			var parents []incblame.Blame
			if diff.PathPrev == "" {
				// TODO: add unit test for this cond
				// file added
				// no parents
			} else {
				for _, p := range commit.Parents {
					pb, ok := s.repo[p][diff.PathPrev]
					if !ok {
						filesAtParent := []string{}
						for f := range s.repo[p] {
							filesAtParent = append(filesAtParent, f)
						}
						panic(fmt.Errorf("could not find reference for commit %v parent %v, path %v, pathPrev %v, files at parent\n%v", commit.Hash, p, diff.Path, diff.PathPrev, filesAtParent))
					}
					parents = append(parents, *pb)
				}
			}
			blame := incblame.Apply(parents, diff, commit.Hash)
			s.repoSave(commit.Hash, diff.Path, &blame)
			res.Files[diff.Path] = &blame
		}

		// copy unchanged file references from first parent
		if len(commit.Parents) >= 1 {
			p := commit.Parents[0]
			files := s.repo[p]
			for path, blame := range files {

				// was in the diff changes, nothing to do
				if _, ok := res.Files[path]; ok {
					continue
				}
				// copy reference
				s.repoSave(commit.Hash, path, blame)

				// No need to send the file to result. We only need blame info for changed files.
				//res.Files[diff.Path] = &blame
			}
		}

		resChan <- res
	}

	close(resChan)

	return nil
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

func (s *Process) gitLog() (io.Reader, error) {

	args := []string{
		"log",
		"-p",
		"-c",
		"--reverse",
		"--no-abbrev-commit",
		"--pretty=format:!Hash: %H%n!Parents: %P",
	}

	ctx := context.Background()
	c := exec.CommandContext(ctx, s.gitCommand, args...)
	stdout := bytes.NewBuffer(nil)
	c.Dir = s.repoDir
	c.Stderr = os.Stderr
	c.Stdout = stdout
	if err := c.Run(); err != nil {
		return nil, fmt.Errorf("failed executing git log -p %v", err)
	}
	return stdout, nil
}