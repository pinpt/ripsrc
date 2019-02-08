package branches2

import (
	"bytes"
	"errors"
	"os/exec"
	"sort"
	"strings"
)

type nameAndHash struct {
	Name   string
	Commit string
}

type namesAndHashes []nameAndHash

func (s namesAndHashes) Chan() chan nameAndHash {
	res := make(chan nameAndHash)
	go func() {
		for _, v := range s {
			res <- v
		}
		close(res)
	}()
	return res
}

func (s *Process) getDefaultBranch() (string, error) {
	args := []string{
		"symbolic-ref",
		"--short",
		"HEAD",
	}
	data, err := execCommand("git", s.opts.RepoDir, args)
	if err != nil {
		return "", err
	}
	res := strings.TrimSpace(string(data))
	if len(res) == 0 {
		return "", errors.New("could not get the default branch name")
	}
	return res, nil
}

func (s *Process) getNamesAndHashes() (res namesAndHashes, _ error) {
	defaultBranch, err := s.getDefaultBranch()
	if err != nil {
		return nil, err
	}
	args := []string{
		"for-each-ref",
		"--format",
		"%(objectname) %(refname:short)",
		"refs/heads",
	}
	data, err := execCommand("git", s.opts.RepoDir, args)
	if err != nil {
		return nil, err
	}
	lines := bytes.Split(data, []byte("\n"))
	for _, line := range lines {
		line := string(line)
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		if line[0] == '(' {
			// not a branch, but a entry for detached head
			// (HEAD detached at faeab7d)
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			panic("unexpected format")
		}
		b := nameAndHash{}
		b.Commit = parts[0]
		b.Name = parts[1]
		if b.Name == defaultBranch {
			continue
		}
		res = append(res, b)
	}
	sort.Slice(res, func(i, j int) bool {
		a := res[i]
		b := res[j]
		return a.Name < b.Name
	})
	return
}

func execCommand(command string, dir string, args []string) ([]byte, error) {
	out := bytes.NewBuffer(nil)
	c := exec.Command(command, args...)
	c.Dir = dir
	c.Stdout = out
	err := c.Run()
	if err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
