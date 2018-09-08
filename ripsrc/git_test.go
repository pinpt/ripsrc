package ripsrc

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStreamCommitsEmpty(t *testing.T) {
	assert := assert.New(t)
	commits := make(chan *Commit, 1000)
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
			assert.NoError(err)
		}
	default:
		break
	}
}

func TestStreamCommitsNotEmpty(t *testing.T) {
	assert := assert.New(t)
	commits := make(chan *Commit, 1)
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
