package ripsrc

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
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
	// the SHA to start streaming from, if not provided will start from the beginning
	SHA string
	// the number of commits to limit, if 0 will include them all
	Limit int
}

// Rip will rip through the directory provided looking for git directories
// and will stream blame details for each commit back to results
// the results channel will automatically be called once all the commits are
// streamed. this function will block until all results are streamed
func Rip(ctx context.Context, dir string, results chan<- BlameResult, errors chan<- error, filter *Filter) {
	pool := NewBlameWorkerPool(ctx, errors, filter)
	pool.Start()
	commits := make(chan Commit, 1000)
	gitdirs, err := findGitDir(dir)
	if err != nil {
		errors <- fmt.Errorf("error finding git dir from %v. %v", dir, err)
		return
	}
	after := make(chan bool, 1)
	// start the goroutine for processing before we start processing
	go func() {
		var count int
		orderedShas := make([]string, 0)
		var currentShaIndex int
		backlog := make(map[string][]*BlameResult)
		var mu sync.Mutex
		// feed each commit into our worker pool for blame processing
		for commit := range commits {
			mu.Lock()
			orderedShas = append(orderedShas, commit.SHA)
			mu.Unlock()
			var wg sync.WaitGroup
			wg.Add(1)
			// submit will send the commit job for async processing ... however, we need to stream them
			// back to the results channel in order that they were originally committed so we're going to
			// have to reorder the results and cache the pending ones that finish before the right order
			pool.Submit(commit, func(err error, result *BlameResult, total int) {
				var last bool
				if err != nil {
					errors <- err
					last = true
				} else {
					if result == nil {
						wg.Done()
						return
					}
					mu.Lock()
					defer mu.Unlock()
					arr := backlog[result.Commit.SHA]
					if arr == nil {
						arr = make([]*BlameResult, 0)
					}
					arr = append(arr, result)
					backlog[result.Commit.SHA] = arr
					last = total == len(arr)
					currentSha := orderedShas[currentShaIndex]
					// fmt.Println("last", last, "count", len(arr), "total", total, "currentShaIndex", currentShaIndex, "filename", result.Filename, "sha", result.Commit.SHA, "current", currentSha)
					// if the current sha matches the one we're looking for and it's the last result
					// we can go ahead and flush (send) and move the index forward to the next sha we're looking for
					if currentSha == result.Commit.SHA && last {
						// sort so it's predictable for the order of the filename
						sort.Slice(arr, func(i, j int) bool {
							return arr[i].Filename < arr[j].Filename
						})
						// fmt.Println(">>>", currentSha)
						for _, r := range arr {
							results <- *r
							count++
						}
						// delete the save memory
						delete(backlog, result.Commit.SHA)
						arr = nil
						// advance to the next sha
						currentShaIndex++
						if len(backlog) != 0 {
							panic("backlog should be empty")
						}
					}
				}
				result = nil
				if last {
					// we're done with this commit once we get to the end
					// we do this just to make sure all commits are processed and written
					// to the results channel before we finish and return
					wg.Done()
				}
			})
			wg.Wait()
		}
		after <- true
	}()
	// setup a goroutine to start processing commits
	for _, gitdir := range gitdirs {
		var sha string
		var limit int
		if filter != nil && filter.SHA != "" {
			sha = filter.SHA
		}
		if filter != nil && filter.Limit > 0 {
			limit = filter.Limit
		}
		if err := streamCommits(ctx, gitdir, sha, limit, commits, errors); err != nil {
			errors <- fmt.Errorf("error streaming commits from git dir from %v. %v", gitdir, err)
			return
		}
	}
	close(commits)
	<-after
	close(results)
	pool.Close()
}
