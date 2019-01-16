package tests

import (
	"archive/zip"
	"context"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/gitexec"
	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/history3/process"
)

var gitCommand = "git"

type Test struct {
	t        *testing.T
	repoName string
	tempDir  string
}

func NewTest(t *testing.T, repoName string) *Test {
	s := &Test{}
	s.t = t
	s.repoName = repoName
	return s
}

func (s *Test) Run(opts *process.Opts) []process.Result {
	t := s.t
	dir, err := ioutil.TempDir("", "ripsrc-test-")
	if err != nil {
		panic(err)
	}
	s.tempDir = dir
	defer func() {
		os.RemoveAll(s.tempDir)
	}()

	repoDirWrapper := filepath.Join(s.tempDir, "repo")
	unzip(filepath.Join(".", "testdata", s.repoName+".zip"), repoDirWrapper)

	repoDir := filepath.Join(repoDirWrapper, firstDir(repoDirWrapper))

	ctx := context.Background()
	err = gitexec.Prepare(ctx, gitCommand, repoDir)
	if err != nil {
		t.Fatal(err)
	}

	if opts == nil {
		opts = &process.Opts{}
	}
	opts.RepoDir = repoDir
	opts.DisableCache = true

	p := process.New(*opts)
	res, err := p.RunGetAll()
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func firstDir(loc string) string {
	entries, err := ioutil.ReadDir(loc)
	if err != nil {
		panic(err)
	}
	for _, entry := range entries {
		n := entry.Name()
		if n[0] == '_' || n[0] == '.' {
			continue
		}
		if entry.IsDir() {
			return entry.Name()
		}
	}
	panic("no dir in: " + loc)
}

func unzip(archive, dir string) error {
	r, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer r.Close()
	ef := func(f *zip.File) error {
		r, err := f.Open()
		if err != nil {
			return err
		}
		defer r.Close()
		p := filepath.Join(dir, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(p, 0777)
			return nil
		}
		os.MkdirAll(filepath.Dir(p), 0777)
		w, err := os.Create(p)
		if err != nil {
			return err
		}
		defer w.Close()
		_, err = io.Copy(w, r)
		if err != nil {
			return err
		}
		return nil
	}
	for _, f := range r.File {
		err := ef(f)
		if err != nil {
			return err
		}
	}
	return nil
}

func assertResult(t *testing.T, want, got []process.Result) {
	t.Helper()
	if len(want) != len(got) {
		t.Fatalf("invalid number of results %v, got\n%v", len(got), got)
	}

	for i := range want {
		w := want[i]
		g := got[i]
		if w.Commit != g.Commit {
			t.Fatalf("invalid commit hash %v at pos %v", w.Commit, i)
		}
		commit := w.Commit
		if len(w.Files) != len(g.Files) {
			t.Fatalf("invalid number of entries %v for commit %v, got\n%v", len(g.Files), commit, g.Files)
		}
		for filePath := range w.Files {
			gf := g.Files[filePath]
			if gf == nil {
				t.Fatalf("missing file %v commit %v\nwanted\n%v", filePath, commit, w.Files[filePath])
			}
			if !w.Files[filePath].Eq(gf) {
				t.Fatalf("invalid patch for file %v commit %v", filePath, commit)
				//t.Fatalf("invalid patch for file %v commit %v, got\n%v\nwanted\n%v", filePath, commit, g.Files[filePath], w.Files[filePath])
			}
		}
	}
}

func file(hash string, lines ...*incblame.Line) *incblame.Blame {
	return &incblame.Blame{Commit: hash, Lines: lines}
}

func line(buf string, commit string) *incblame.Line {
	return &incblame.Line{Line: []byte(buf), Commit: commit}
}
