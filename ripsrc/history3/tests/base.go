package tests

import (
	"archive/zip"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/history3/process"
)

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

func (s *Test) Run() []process.Result {
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

	p := process.New(repoDir)
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
		t.Fatalf("invalid number of entries %v, got\n%v", len(got), got)
	}
	for i := range want {
		w := want[i]
		g := got[i]
		if w.Commit != g.Commit {
			t.Fatalf("invalid commit hash %v at pos %v", w.Commit, i)
		}
		commit := w.Commit
		if len(w.Files) != len(g.Files) {
			t.Fatalf("invalid number of entries %v for commit %v, got\n%v", len(w.Files), commit, g.Files)
		}
		for filePath := range w.Files {
			if !reflect.DeepEqual(w.Files[filePath], g.Files[filePath]) {
				t.Fatalf("invalid patch for file %v commit %v, got\n%v\nwanted\n%v", filePath, commit, g.Files[filePath], w.Files[filePath])
			}
		}
	}
}

func file(hash string, lines ...incblame.Line) *incblame.Blame {
	return &incblame.Blame{Commit: hash, Lines: lines}
}

func line(buf string, commit string) incblame.Line {
	return incblame.Line{Line: []byte(buf), Commit: commit}
}
