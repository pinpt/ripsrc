package ripsrc

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/pinpt/ripsrc/ripsrc/patch"
)

// Filter provides a blacklist (exclude) and/or whitelist (include) filter
type Filter struct {
	Blacklist *regexp.Regexp
	Whitelist *regexp.Regexp
	// the SHA to start streaming from, if not provided will start from the beginning
	SHA string
	// the number of commits to limit, if 0 will include them all
	Limit int
}

func formatBuf(line string) string {
	toks := strings.Split(line, "\n")
	newtoks := []string{}
	for i, tok := range toks {
		newtoks = append(newtoks, fmt.Sprintf("%02d|%s", 1+i, tok))
	}
	return strings.Join(newtoks, "\n")
}

type commitjob struct {
	filename string
	commit   Commit
	file     *patch.File
}

// Rip will rip through the directory provided looking for git directories
// and will stream blame details for each commit back to results
// the results channel will automatically be called once all the commits are
// streamed. this function will block until all results are streamed
func Rip(ctx context.Context, fdir string, results chan<- BlameResult, filter *Filter) error {
	dir, _ := filepath.Abs(fdir)
	gitdir := filepath.Join(dir, ".git")
	if _, err := os.Stat(gitdir); os.IsNotExist(err) {
		return fmt.Errorf("error finding git dir from %v", dir)
	}
	errors := make(chan error, 1)
	size := 10000000 // FIXME once we fix the history.wait below make this reasonable
	var debugdir string
	if RipDebug {
		cwd, _ := os.Getwd()
		debugdir = filepath.Join(cwd, "ripsrc-debug")
		os.RemoveAll(debugdir)
	}
	cachedir := filepath.Join(gitdir, ".ripcache")
	if _, err := os.Stat(cachedir); os.IsNotExist(err) {
		os.MkdirAll(cachedir, 0755)
	}
	history := newCommitFileHistory(cachedir)
	processor := NewBlameProcessor(filter)
	commits := make(chan Commit, size)
	commitjobs := make(chan commitjob, 1000)
	var wg sync.WaitGroup
	var total int
	wg.Add(1)
	// start the goroutine for processing before we start processing
	go func() {
		defer wg.Done()
		// TODO: wait to block below when the file is needed instead of up front
		if err := history.wait(); err != nil {
			errors <- err
			close(commitjobs)
			return
		}
		for commit := range commits {
			for filename, cf := range commit.Files {
				// fmt.Println("@@", commit.SHA, filename)
				file, err := history.Get(filename, commit.SHA)
				if err != nil {
					errors <- err
					close(commitjobs)
					return
				}
				// we skip commits not found since they may not be part of the history as related to merging
				if cf.Status == GitFileCommitStatusModified && (file == nil || file.Empty()) {
					continue
				}
				commitjobs <- commitjob{filename, commit, file}
			}
			total++
		}
		close(commitjobs)
		history = nil
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for job := range commitjobs {
			result, err := processor.process(job)
			if err != nil {
				errors <- err
				return
			}
			if result != nil {
				results <- *result
			}
		}
	}()
	var sha string
	var limit int
	if filter != nil && filter.SHA != "" {
		sha = filter.SHA
	}
	if filter != nil && filter.Limit > 0 {
		limit = filter.Limit
	}
	if err := streamCommits(ctx, gitdir, cachedir, sha, limit, processor, history, commits, errors); err != nil {
		return fmt.Errorf("error streaming commits from git dir from %v. %v", gitdir, err)
	}
	close(commits)
	wg.Wait()
	select {
	case err := <-errors:
		return err
	default:
		break
	}
	return nil
}

// RipDebug is a setting for printing out detail during rip
var RipDebug = os.Getenv("RIPSRC_DEBUG") == "true"
