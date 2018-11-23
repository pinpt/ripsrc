package ripsrc

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/sergi/go-diff/diffmatchpatch"

	"github.com/pinpt/ripsrc/ripsrc/patch"

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

func formatBuf(line string) string {
	toks := strings.Split(line, "\n")
	newtoks := []string{}
	for i, tok := range toks {
		newtoks = append(newtoks, fmt.Sprintf("%02d|%s", 1+i, tok))
	}
	return strings.Join(newtoks, "\n")
}

type commitjob struct {
	commit Commit
	file   patch.File
}

// Rip will rip through the directory provided looking for git directories
// and will stream blame details for each commit back to results
// the results channel will automatically be called once all the commits are
// streamed. this function will block until all results are streamed
func Rip(ctx context.Context, fdir string, results chan<- BlameResult, filter *Filter, validate bool) error {
	dir, _ := filepath.Abs(fdir)
	gitdirs, err := findGitDir(dir)
	if err != nil {
		return fmt.Errorf("error finding git dir from %v. %v", dir, err)
	}
	errors := make(chan error, 100)
	var size int
	if !validate {
		size = 1000
	}
	processor := NewBlameProcessor(filter)
	commits := make(chan Commit, size)
	commitjobs := make(chan commitjob, size)
	var wg sync.WaitGroup
	wg.Add(1)
	// start the goroutine for processing before we start processing
	go func() {
		defer wg.Done()
		files := make(map[string]*patch.File)
		for commit := range commits {
			for _, file := range commit.diff.files {
				current := files[file.Filename]
				cf := commit.Files[file.Filename]
				if cf.Renamed {
					current = files[cf.RenamedFrom]
				}
				newfile := file.Apply(current, commit)
				if validate {
					newbuf := newfile.String()
					var oldbufb bytes.Buffer
					c := exec.Command("git", "show", commit.SHA+":"+file.Filename)
					c.Stdout = &oldbufb
					c.Stderr = os.Stderr
					c.Dir = dir
					c.Run()
					newbuf = strings.TrimSpace(newbuf)
					oldbuf := strings.TrimSpace(oldbufb.String())
					if newbuf != oldbuf {
						fmt.Println("invalid commit diff", commit.SHA, file.Filename)
						for _, df := range commit.diff.files {
							if df.Filename == file.Filename {
								fmt.Println("diff which was applied:" + df.String())
								break
							}
						}
						fmt.Println(strings.Repeat("-", 120))
						if current != nil {
							fmt.Println("applied to >>" + current.Stringify(true) + "<<")
						} else {
							fmt.Println("applied to >><<")
						}
						fmt.Println(strings.Repeat("-", 120))
						dmp := diffmatchpatch.New()
						diffs := dmp.DiffMain(oldbuf, newbuf, true)
						fmt.Println("DIFFERENCE:")
						fmt.Println(dmp.DiffPrettyText(diffs))
						fmt.Println(strings.Repeat("-", 120))
						fmt.Println("EXPECTED >>" + oldbuf + "<<")
						fmt.Println(strings.Repeat("-", 120))
						fmt.Println("WAS >>" + newbuf + "<<")
						fmt.Println("PREVIOUS COMMITS FOR FILE:")
						fmt.Println(strings.Repeat("-", 120))
						p := commit.parent
						for p != nil {
							tf := p.Files[file.Filename]
							if tf != nil {
								fmt.Println(p.CommitSHA(), p.Date, p.Author(), p.Message)
								for _, diff := range p.diff.files {
									if diff.Filename == file.Filename {
										fmt.Println(diff)
										fmt.Println(strings.Repeat("-", 120))
										break
									}
								}
							}
							p = p.parent
						}
						os.Exit(1)
					}
				}
				files[file.Filename] = newfile
				commitjobs <- commitjob{commit, *newfile}
			}
		}
		close(commitjobs)
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
			return fmt.Errorf("error streaming commits from git dir from %v. %v", gitdir, err)
		}
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
