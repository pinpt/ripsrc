package ripcmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/pinpt/ripsrc/ripsrc"
)

type Opts struct {
	// Dir is directory to run ripsrc on.
	// If it contains .git directory inside, this dir will be processed.
	// If the dir name ends with .git and has objects dir inside it will be assumed to be bare repo and processed.
	// If neither of this is true if will process containing dirs following the same algo.
	Dir string
}

type Stats struct {
	Repos      int
	EmptyRepos int
	Entries    int
}

func Run(ctx context.Context, out io.Writer, opts Opts) {
	start := time.Now()
	stats, err := runOnDirs(ctx, out, opts, opts.Dir, 1)
	if err != nil {
		fmt.Println("failed processing with err", err)
		os.Exit(1)
	}
	if stats.Repos == 0 {
		fmt.Println("failed processing, no git repos found in supplied dir:", opts.Dir)
		os.Exit(1)
	}
	if stats.EmptyRepos != 0 {
		fmt.Fprintf(color.Output, "%v", color.RedString("Warning! Skipped %v empty repos\n", stats.EmptyRepos))
	}
	fmt.Printf("finished processing repos %d entries %d in %v\n", stats.Repos, stats.Entries, time.Since(start))
}

func runOnDirs(ctx context.Context, wr io.Writer, opts Opts, dir string, recurseLevels int) (stats Stats, _ error) {
	stat, err := os.Stat(dir)
	if err != nil {
		return stats, fmt.Errorf("can't stat passed dir, err: %v", err)
	}
	if !stat.IsDir() {
		return stats, fmt.Errorf("passed dir is a file, expecting a dir")
	}
	// check if contains .git
	containsDotGit, err := dirContainsDir(dir, ".git")
	if err != nil {
		return stats, err
	}
	if containsDotGit {
		entries, err := runOnRepo(ctx, wr, dir)
		if err != nil {
			return stats, err
		}
		stats.Repos = 1
		stats.Entries = entries
		if entries == 0 {
			stats.EmptyRepos++
		}
		return stats, nil
	}

	loc, err := filepath.Abs(dir)
	if err != nil {
		return stats, fmt.Errorf("can't convert passed dir to absolute path, err: %v", err)
	}

	if filepath.Ext(loc) == ".git" {
		containsObjects, err := dirContainsDir(dir, "objects")
		if err != nil {
			return stats, err
		}
		if containsObjects {
			entries, err := runOnRepo(ctx, wr, dir)
			if err != nil {
				return stats, err
			}
			stats.Repos = 1
			stats.Entries = entries
			if entries == 0 {
				stats.EmptyRepos++
			}
			return stats, nil
		}
	}

	if recurseLevels == 0 {
		return stats, nil
	}

	subs, err := ioutil.ReadDir(dir)
	if err != nil {
		return stats, fmt.Errorf("can't read passed dir, err: %v", err)
	}

	for _, sub := range subs {
		if !sub.IsDir() {
			continue
		}
		subEntries, err := runOnDirs(ctx, wr, opts, filepath.Join(dir, sub.Name()), recurseLevels-1)
		stats.Repos += subEntries.Repos
		stats.Entries += subEntries.Entries
		stats.EmptyRepos += subEntries.EmptyRepos
		if err != nil {
			return stats, err
		}
	}
	return stats, nil
}

func dirContainsDir(dir string, sub string) (bool, error) {
	stat, err := os.Stat(filepath.Join(dir, sub))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		} else {
			return false, fmt.Errorf("can't check if dir contains %v, dir: %v err: %v", sub, dir, err)
		}
	}
	if !stat.IsDir() {
		return false, nil
	}
	return true, nil
}

func runOnRepo(ctx context.Context, wr io.Writer, repoDir string) (entries int, _ error) {
	start := time.Now()
	fmt.Fprintf(color.Output, "starting processing %v\n", color.GreenString(repoDir))
	if !hasHeadCommit(ctx, repoDir) {
		fmt.Fprintf(color.Output, "repo is empty (or other problem running git rev-parse HEAD) %v\n", color.GreenString(repoDir))
		return 0, nil
	}
	ripper := ripsrc.New()

	res := make(chan ripsrc.BlameResult)
	done := make(chan bool)
	go func() {
		for blame := range res {
			entries++
			var license string
			if blame.License != nil {
				license = fmt.Sprintf("%v (%.0f%%)", color.RedString(blame.License.Name), 100*blame.License.Confidence)
			}
			fmt.Fprintf(color.Output, "[%s] %s language=%s,license=%v,loc=%v,sloc=%v,comments=%v,blanks=%v,complexity=%v,skipped=%v,status=%s,author=%s\n", color.CyanString(blame.Commit.SHA[0:8]), color.GreenString(blame.Filename), color.MagentaString(blame.Language), license, blame.Loc, color.YellowString("%v", blame.Sloc), blame.Comments, blame.Comments, blame.Complexity, blame.Skipped, blame.Commit.Files[blame.Filename].Status, blame.Commit.Author())
		}
		done <- true
	}()

	err := ripper.Rip(ctx, repoDir, res)
	<-done

	if err != nil {
		return entries, err
	}

	fmt.Fprintln(wr)
	ripper.GitProcessTimings.OutputStats(wr)
	fmt.Fprintln(wr)
	ripper.CodeInfoTimings.OutputStats(wr)
	fmt.Fprintln(wr)

	fmt.Fprintf(color.Output, "finished repo processing for %v in %v. %d entries processed\n", color.HiGreenString(repoDir), time.Since(start), entries)

	return entries, nil
}

func hasHeadCommit(ctx context.Context, repoDir string) bool {
	out := bytes.NewBuffer(nil)
	c := exec.Command("git", "rev-parse", "HEAD")
	c.Dir = repoDir
	c.Stdout = out
	c.Run()
	res := strings.TrimSpace(out.String())
	if len(res) != 40 {
		return false
	}
	return true
}
