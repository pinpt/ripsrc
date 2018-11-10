package ripsrc

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStreamCommitsEmpty(t *testing.T) {
	assert := assert.New(t)
	commits := make(chan Commit, 1000)
	errors := make(chan error, 1)
	cwd := os.TempDir()
	assert.NoError(streamCommits(context.Background(), cwd, "", 0, commits, errors))
	select {
	case <-commits:
		{
			assert.Fail("found an unexpected commit")
		}
	case err := <-errors:
		{
			assert.Error(err)
			assert.Contains(err.Error(), "exit status 128. fatal: not a git repository (or any of the parent directories): .git")
		}
	default:
		break
	}
}

func TestStreamCommitsNotEmpty(t *testing.T) {
	assert := assert.New(t)
	commits := make(chan Commit, 1)
	errors := make(chan error, 1)
	c := exec.Command("git", "log", "-n", "2", "--format=%H", "--no-merges")
	var buf strings.Builder
	cwd, _ := os.Getwd()
	c.Dir = filepath.Join(cwd, "..")
	c.Stdout = &buf
	assert.NoError(c.Run())
	shas := strings.Split(strings.TrimSpace(buf.String()), "\n")
	assert.NoError(streamCommits(context.Background(), c.Dir, shas[1], 0, commits, errors))
	select {
	case commit := <-commits:
		{
			assert.Equal(shas[0], commit.SHA)
		}
	case err := <-errors:
		{
			assert.NoError(err)
		}
	default:
		assert.Fail("should have found a commit and didn't")
	}
}

func TestToCommitStatus(t *testing.T) {
	assert := assert.New(t)
	tt := []struct {
		data   []byte
		answer string
	}{
		{[]byte(""), ""},
		{[]byte("A"), "added"},
		{[]byte("D"), "removed"},
		{[]byte("M"), "modified"},
	}
	for _, v := range tt {
		response := toCommitStatus(v.data)
		assert.Contains(response, v.answer)
	}
}

func TestParseDate(t *testing.T) {
	assert := assert.New(t)
	tt := []struct {
		data   string
		answer string
	}{
		{"2018-09-26", ""},
	}
	for _, v := range tt {
		_, err := parseDate(v.data)
		assert.Error(err)
	}
}

func TestParseEmail(t *testing.T) {
	assert := assert.New(t)
	tt := []struct {
		data   string
		answer string
	}{
		{"^cf89534 <jhaynie@pinpt.com> 2018-06-30 17:06:10 -0700   1", "jhaynie@pinpt.com"},
		{"<someone@somewhere.com>", "someone@somewhere.com"},
		{"<[someone@somewhere.com]>", "someone@somewhere.com"},
		{"\\", ""},
		{"", ""},
	}
	for _, v := range tt {
		response := parseEmail(v.data)
		assert.Equal(response, v.answer)
	}
}

func TestGetFilename(t *testing.T) {
	assert := assert.New(t)
	tt := []struct {
		data   string
		answer string
	}{
		{"/somewhere/somewhere/somewhere/something.txt => /somewhere/somewhere/something.txt", "/somewhere/somewhere/something.txt"},
		{`file.go\{file.go => newfile.go\}newfile.go`, `file.go\/newfile.go\/newfile.go`},
		{"", ""},
		{"/", "/"},
	}
	for _, v := range tt {
		response, _, _ := getFilename(v.data)
		assert.Equal(response, v.answer)
	}
}

func TestStreamCommitsWithAnError(t *testing.T) {
	assert := assert.New(t)
	commits := make(chan Commit, 1000)
	errors := make(chan error, 1)
	tmpdir := os.TempDir()
	defer os.RemoveAll(tmpdir)
	fn := filepath.Join(tmpdir, "rungit.sh")
	gfn := filepath.Join(tmpdir, "mockgit.go")
	gout := `package main

import "fmt"
func main() {
	fmt.Println("hi")
	panic(1)
}
	`
	ioutil.WriteFile(gfn, []byte(gout), 0644)
	bout := fmt.Sprintf(`#!/bin/sh
echo running...
go run %s
	`, gfn)
	ioutil.WriteFile(fn, []byte(bout), 0644)
	exec.Command("chmod", "+x", fn).Run()
	oldGit := gitCommand
	defer func() { gitCommand = oldGit }()
	gitCommand = fn
	cwd, _ := os.Getwd()
	cwd = filepath.Join(cwd, "..")
	err := streamCommits(context.Background(), cwd, "", 0, commits, errors)
	assert.NotNil(err)
	assert.True(strings.Contains(err.Error(), "error streaming commits from"))
}
