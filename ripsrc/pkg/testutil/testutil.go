package testutil

import (
	"archive/zip"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

type TestRepoDirs struct {
	TempWrapper string
	RepoDir     string
}

func (s TestRepoDirs) Remove() {
	err := os.RemoveAll(s.TempWrapper)
	if err != nil {
		panic(err)
	}
}

func UnzipTestRepo(repoName string) TestRepoDirs {
	return UnzipTestRepoLoc(filepath.Join(".", "testdata", repoName+".zip"))
}

func UnzipTestRepoLoc(zipLoc string) (res TestRepoDirs) {
	tempDir, err := ioutil.TempDir("", "ripsrc-test-")
	if err != nil {
		panic(err)
	}
	res.TempWrapper = tempDir
	repoDirWrapper := filepath.Join(tempDir, "repo")
	unzip(zipLoc, repoDirWrapper)
	res.RepoDir = filepath.Join(repoDirWrapper, firstDir(repoDirWrapper))
	return
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
