package branches

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/pinpt/ripsrc/ripsrc/parentsgraph"
	"github.com/pinpt/ripsrc/ripsrc/pkg/logger"
)

type Process struct {
	opts Opts

	graph *parentsgraph.Graph

	Branches   []Branch
	branchToID map[string]int

	// map[commit][]branchID
	data map[string][]int
}

type Opts struct {
	Logger logger.Logger

	RepoDir string
	// ParentsGraph is optional graph of commits. Pass to reuse, if not passed will be created.
	ParentsGraph *parentsgraph.Graph
}

func New(opts Opts) *Process {
	s := &Process{}
	s.opts = opts
	s.branchToID = map[string]int{}
	s.data = map[string][]int{}
	return s
}

func (s *Process) Run() error {
	s.graph = s.opts.ParentsGraph
	if s.graph == nil {
		s.graph = parentsgraph.New(parentsgraph.Opts{
			RepoDir:     s.opts.RepoDir,
			AllBranches: true,
			Logger:      s.opts.Logger,
		})
		err := s.graph.Read()
		if err != nil {
			return err
		}
	}
	var err error
	s.Branches, err = s.getBranches()
	if err != nil {
		return err
	}
	for i, b := range s.Branches {
		s.branchToID[b.Name] = i
		err := s.attributeCommits(i, b.Commit)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Process) BranchesThatIncludeCommit(commit string) (res []string) {
	for _, id := range s.data[commit] {
		res = append(res, s.Branches[id].Name)
	}
	return
}

func (s *Process) CommitsToBranches() map[string][]string {
	res := map[string][]string{}
	for c := range s.data {
		res[c] = s.BranchesThatIncludeCommit(c)
	}
	return res
}

func (s *Process) attributeCommits(id int, fromCommit string) error {
	c := fromCommit
	s.data[c] = append(s.data[c], id)
	parents, ok := s.graph.Parents[c]
	if !ok {
		return fmt.Errorf("commit not found in graph: %v", c)
	}
	for _, p := range parents {
		err := s.attributeCommits(id, p)
		if err != nil {
			return err
		}
	}
	return nil
}

type Branch struct {
	Name   string
	Commit string
}

func (s *Process) getBranches() (res []Branch, _ error) {
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
		b := Branch{}
		b.Commit = parts[0]
		b.Name = parts[1]
		res = append(res, b)
	}
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
