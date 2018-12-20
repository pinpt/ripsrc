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

			var parents []incblame.Blame
			for _, p := range commit.Parents {
				pb, ok := s.repo[p][diff.PathPrev]
				if !ok {
					panic(fmt.Errorf("could not find reference for commit %v, path %v", p, diff.PathPrev))
				}
				parents = append(parents, *pb)
			}

			blame := incblame.Apply(parents, diff, commit.Hash)
			s.repoSave(commit.Hash, diff.Path, blame)
			res.Files[diff.Path] = &blame
		}
		resChan <- res
	}

	close(resChan)

	return nil
}

func (s *Process) repoSave(commit, path string, blame incblame.Blame) {
	if _, ok := s.repo[commit]; !ok {
		s.repo[commit] = map[string]*incblame.Blame{}
	}
	s.repo[commit][path] = &blame
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
