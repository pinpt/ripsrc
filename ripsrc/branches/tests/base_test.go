package tests

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/branches"

	"github.com/pinpt/ripsrc/ripsrc/gitexec"
	"github.com/pinpt/ripsrc/ripsrc/pkg/logger"
	"github.com/pinpt/ripsrc/ripsrc/pkg/testutil"
)

type Test struct {
	t        *testing.T
	repoName string
	opts     *branches.Opts
}

func NewTest(t *testing.T, repoName string, opts *branches.Opts) *Test {
	s := &Test{}
	s.t = t
	s.repoName = repoName
	s.opts = opts
	return s
}

func (s *Test) Run() *branches.Process {
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
		opts = &branches.Opts{}
	}
	opts.Logger = logger.NewDefaultLogger(os.Stdout)
	opts.RepoDir = dirs.RepoDir
	pg := branches.New(*opts)
	err = pg.Run()
	if err != nil {
		t.Fatal(err)
	}
	return pg
}

func assertResult(t *testing.T, br *branches.Process, wantData map[string][]string) {
	t.Helper()
	got := br.CommitsToBranches()
	assertEqualMaps(t, wantData, got, "")
}

func assertEqualMaps(t *testing.T, wantMap, gotMap map[string][]string, label string) {
	t.Helper()
	for _, data := range wantMap {
		sort.Strings(data)
	}
	if !reflect.DeepEqual(wantMap, gotMap) {
		t.Errorf("invalid map %v\ngot\n%v\nwanted\n%v", label, printMap(gotMap), printMap(wantMap))
	}
}

func printMap(m map[string][]string) string {
	type kv struct {
		k string
		v []string
	}
	var kvs []kv
	for k, v := range m {
		kvs = append(kvs, kv{k, v})
	}
	sort.Slice(kvs, func(i, j int) bool {
		a := kvs[i]
		b := kvs[j]
		return a.k < b.k
	})
	res := []string{}
	for _, kv := range kvs {
		res = append(res, fmt.Sprintf("%v %v", kv.k, kv.v))
	}
	return strings.Join(res, "\n")
}
