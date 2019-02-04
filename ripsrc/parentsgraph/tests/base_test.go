package tests

import (
	"context"
	"os"
	"reflect"
	"sort"
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/pkg/logger"

	"github.com/pinpt/ripsrc/ripsrc/parentsgraph"

	"github.com/pinpt/ripsrc/ripsrc/gitexec"
	"github.com/pinpt/ripsrc/ripsrc/pkg/testutil"
)

type Test struct {
	t        *testing.T
	repoName string
	opts     *parentsgraph.Opts
}

func NewTest(t *testing.T, repoName string, opts *parentsgraph.Opts) *Test {
	s := &Test{}
	s.t = t
	s.repoName = repoName
	s.opts = opts
	return s
}

func (s *Test) Run() *parentsgraph.Graph {
	t := s.t
	dirs := testutil.UnzipTestRepo(s.repoName)
	defer dirs.Remove()

	ctx := context.Background()
	err := gitexec.Prepare(ctx, "git", dirs.RepoDir)
	if err != nil {
		t.Fatal(err)
	}
	opts := s.opts
	if opts == nil {
		opts = &parentsgraph.Opts{}
	}
	opts.Logger = logger.NewDefaultLogger(os.Stdout)
	opts.RepoDir = dirs.RepoDir
	pg := parentsgraph.New(*opts)
	err = pg.Read()
	if err != nil {
		t.Fatal(err)
	}
	return pg
}

func assertResult(t *testing.T, pg *parentsgraph.Graph, wantParents, wantChildren map[string][]string) {
	t.Helper()
	assertEqualMaps(t, wantParents, pg.Parents, "parents")
	assertEqualMaps(t, wantChildren, pg.Children, "children")
}

func assertEqualMaps(t *testing.T, wantMap, gotMap map[string][]string, label string) {
	t.Helper()
	for _, data := range wantMap {
		sort.Strings(data)
	}
	if !reflect.DeepEqual(wantMap, gotMap) {
		t.Errorf("invalid map %v\ngot\n%v\nwanted\n%v", label, gotMap, wantMap)
	}
}
