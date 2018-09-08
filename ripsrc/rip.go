package ripsrc

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"runtime"
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
}

// Rip will rip through the directory provided looking for git directories
// and will stream blame details for each commit back to results
// the results channel will automatically be called once all the commits are
// streamed. this function will block until all results are streamed
func Rip(ctx context.Context, dir string, results chan<- BlameResult, errors chan<- error, filter *Filter) {
	pool := NewBlameWorkerPool(ctx, runtime.NumCPU(), errors, filter)
	pool.Start()
	commits := make(chan *Commit, 1000)
	gitdirs, err := findGitDir(dir)
	if err != nil {
		errors <- fmt.Errorf("error finding git dir from %v. %v", dir, err)
		return
	}
	for _, gitdir := range gitdirs {
		var sha string
		if filter != nil && filter.SHA != "" {
			sha = filter.SHA
		}
		if err := streamCommits(ctx, gitdir, sha, commits, errors); err != nil {
			errors <- fmt.Errorf("error streaming commits from git dir from %v. %v", gitdir, err)
			return
		}
	}
	// setup a goroutine to start processing commits
	var count int
	after := make(chan bool, 1)
	orderedShas := make([]string, 0)
	var currentShaIndex int
	var backlog sync.Map
	go func() {
		var wg sync.WaitGroup
		// feed each commit into our worker pool for blame processing
		for commit := range commits {
			// fmt.Println(">>>>>>", commit.SHA)
			orderedShas = append(orderedShas, commit.SHA)
			wg.Add(1)
			// submit will send the commit job for async processing ... however, we need to stream them
			// back to the results channel in order that they were originally committed so we're going to
			// have to reorder the results and cache the pending ones that finish before the right order
			pool.Submit(commit, func(err error, result *BlameResult, total int) {
				// fmt.Println("total", total, "currentShaIndex", currentShaIndex, "filename", result.Filename, result.Commit.SHA)
				var last bool
				if err != nil {
					errors <- err
					last = true
				} else {
					var arr []*BlameResult
					found, ok := backlog.Load(result.Commit.SHA)
					if ok {
						arr = found.([]*BlameResult)
					} else {
						arr = make([]*BlameResult, 0)
					}
					arr = append(arr, result)
					backlog.Store(result.Commit.SHA, arr)
					last = total == len(arr)
					currentSha := orderedShas[currentShaIndex]
					// fmt.Println("last", last, "currentSha", currentSha, "len", len(arr), "total", total)
					// if the current sha matches the one we're looking for and it's the last result
					// we can go ahead and flush (send) and move the index forward to the next sha we're looking for
					if currentSha == result.Commit.SHA && last {
						// sort so it's predictable for the order of the filename
						sort.Slice(arr, func(i, j int) bool {
							return arr[i].Filename < arr[j].Filename
						})
						for _, r := range arr {
							results <- *r
							count++
						}
						// delete the save memory
						backlog.Delete(result.Commit.SHA)
						// advance to the next sha
						currentShaIndex++
					}
				}
				if last {
					// we're done with this commit once we get to the end
					// we do this just to make sure all commits are processed and written
					// to the results channel before we finish and return
					wg.Done()
				}
			})
		}
		wg.Wait()
		after <- true
	}()
	close(commits)
	<-after
	// now we can get here and we're finished but we still have a number of shas that have been buffered (out of order)
	// but still need to be flushed. in this case, we have all the data, we just need to send from current sha index to the end
	for i := currentShaIndex; i < len(orderedShas); i++ {
		sha := orderedShas[i]
		found, ok := backlog.Load(sha)
		if ok {
			arr := found.([]*BlameResult)
			// sort so it's predictable for the order of the filename
			sort.Slice(arr, func(i, j int) bool {
				return arr[i].Filename < arr[j].Filename
			})
			for _, r := range arr {
				results <- *r
				count++
			}
		} else {
			panic("expected sha not found " + sha)
		}
	}
	close(results)
	pool.Close()
}
