package tests

import (
	"archive/zip"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/pinpt/ripsrc/ripsrc/commitmeta"
	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
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

func (s *Test) Run() []commitmeta.Commit {
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

	p := commitmeta.New(repoDir)
	res, err := p.RunSlice()
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

func assertCommits(t *testing.T, want, got []commitmeta.Commit) {
	t.Helper()
	if len(want) != len(got) {
		t.Fatalf("invalid number of entries, got %v, wanted %v, got\n%v", len(got), len(want), got)
	}
	for i := range want {
		w := want[i]
		g := got[i]
		if !assert.Equal(t, w, g) {
			t.Fatal()
		}
	}
}

func file(hash string, lines ...incblame.Line) *incblame.Blame {
	return &incblame.Blame{Commit: hash, Lines: lines}
}

func line(buf string, commit string) incblame.Line {
	return incblame.Line{Line: []byte(buf), Commit: commit}
}

func parseGitDate(s string) time.Time {
	//Tue Nov 27 21:55:36 2018 +0100
	r, err := time.Parse("Mon Jan 2 15:04:05 2006 -0700", s)
	if err != nil {
		panic(err)
	}
	return r.UTC()
}

func strp(s string) *string {
	return &s
}
