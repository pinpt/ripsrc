package parentsgraph

import (
	"context"
	"io"
	"sort"
	"time"

	"github.com/pinpt/ripsrc/ripsrc/gitexec"
	"github.com/pinpt/ripsrc/ripsrc/parentsgraph/parentsp"
	"github.com/pinpt/ripsrc/ripsrc/pkg/logger"
)

type Graph struct {
	opts     Opts
	Parents  map[string][]string
	Children map[string][]string
}

type Opts struct {
	RepoDir     string
	AllBranches bool
	Logger      logger.Logger
}

func New(opts Opts) *Graph {
	s := &Graph{}
	s.opts = opts
	return s
}

func (s *Graph) Read() error {
	start := time.Now()
	s.opts.Logger.Info("parentsgraph: starting reading")
	defer func() {
		s.opts.Logger.Info("parentsgraph: completed reading", "d", time.Since(start))
	}()
	err := s.retrieveParents()
	if err != nil {
		return err
	}
	s.Children = map[string][]string{}
	for commit, parents := range s.Parents {
		if _, ok := s.Children[commit]; !ok {
			// make sure that even if commit does not have any children we have a map key for it
			s.Children[commit] = nil
		}
		for _, p := range parents {
			s.Children[p] = append(s.Children[p], commit)
		}
	}
	for _, data := range s.Children {
		sort.Strings(data)
	}
	return nil
}

func (s *Graph) retrieveParents() error {
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

	s.Parents = res
	return nil
}

func (s *Graph) gitLogParents() (io.ReadCloser, error) {
	args := []string{
		"log",
		"-m",
		"--reverse",
		"--no-abbrev-commit",
		"--pretty=format:%H@%P",
	}

	if s.opts.AllBranches {
		args = append(args, "--all")
	}

	ctx := context.Background()
	return gitexec.ExecPiped(ctx, "git", s.opts.RepoDir, args)
}
