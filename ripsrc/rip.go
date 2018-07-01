package ripsrc

import (
	"fmt"
	"path/filepath"
	"regexp"
	"runtime"
	"sync"

	"github.com/karrick/godirwalk"
)

// find either export gitdata or .git - that way it works for both dev and prod
func findGitDir(dir string) ([]string, error) {
	dupe := make(map[string]bool)
	fileList := []string{}
	err := godirwalk.Walk(dir, &godirwalk.Options{
		Callback: func(osPathname string, de *godirwalk.Dirent) error {
			if de.IsDir() {
				basedir := de.Name()
				if basedir == "gitdata" && !dupe[osPathname] {
					fileList = append(fileList, osPathname)
					dupe[osPathname] = true
				} else if basedir == ".git" {
					fp := filepath.Dir(osPathname)
					if !dupe[fp] {
						dupe[fp] = true
						fileList = append(fileList, fp)
					}
				}
			}
			return nil
		},
	})
	return fileList, err
}

// Filter provides a blacklist (exclude) and/or whitelist (include) filter
type Filter struct {
	Blacklist *regexp.Regexp
	Whitelist *regexp.Regexp
}

// Rip will rip through all directories provided looking for git directories
// and will stream blame details for each commit back to results
// the results channel will automatically be called once all the commits are
// streamed. this function will block until all results are streamed
func Rip(dirs []string, results chan<- BlameResult, errors chan<- error, filter *Filter) {
	pool := NewBlameWorkerPool(runtime.NumCPU(), results, errors, filter)
	pool.Start()
	commits := make(chan *Commit, 1000)
	var wg sync.WaitGroup
	sem := NewSemaphore(runtime.NumCPU())
	// cycle through each directory in the command line and stream commits from them
	for _, dir := range dirs {
		gitdirs, err := findGitDir(dir)
		if err != nil {
			errors <- fmt.Errorf("error finding git dir from %v. %v", dir, err)
			return
		}
		for _, gitdir := range gitdirs {
			wg.Add(1)
			// we use a semaphore so that we don't overrun the open files limit
			sem.Acquire()
			if err := StreamCommits(gitdir, commits, &wg, errors); err != nil {
				sem.Release()
				errors <- fmt.Errorf("error streaming commits from git dir from %v. %v", gitdir, err)
				return
			}
			sem.Release()
		}
	}
	// setup a goroutine to start processing commits
	var count int
	after := make(chan bool, 1)
	go func() {
		// feed each commit into our worker pool for blame processing
		for commit := range commits {
			pool.Submit(commit)
			count++
		}
		after <- true
	}()
	// now wait for our processing to complete
	wg.Wait()
	close(commits)
	<-after
	pool.Close()
}
