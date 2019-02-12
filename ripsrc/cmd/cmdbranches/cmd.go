package cmdbranches

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/pinpt/ripsrc/ripsrc/pkg/gitrepos"

	"github.com/fatih/color"
	"github.com/pinpt/ripsrc/ripsrc"
	"github.com/pinpt/ripsrc/ripsrc/cmd/cmdutils"
)

type Opts struct {
	// Dir is directory to run ripsrc on.
	// If it contains .git directory inside, this dir will be processed.
	// If the dir name ends with .git and has objects dir inside it will be assumed to be bare repo and processed.
	// If neither of this is true if will process containing dirs following the same algo.
	Dir string

	// Profile set to one of mem, mutex, cpu, block, trace to enable profiling.
	Profile string
}

type Stats struct {
	Repos             int
	SkippedEmptyRepos int
}

type RepoError struct {
	Repo string
	Err  error
}

func (s RepoError) Error() string {
	return fmt.Sprintf("repo: %v err: %v", s.Repo, s.Err)
}

func Run(ctx context.Context, out io.Writer, opts Opts) {
	start := time.Now()

	if opts.Profile != "" {
		runEndHook := cmdutils.EnableProfiling(opts.Profile)
		defer runEndHook()
	}

	{
		onEnd := cmdutils.StartMemLogs()
		defer onEnd()
	}

	stats, repoErrs, err := runOnDirs(ctx, out, opts, opts.Dir, start)
	if err != nil {
		cmdutils.ExitWithErr(err)
	}

	if len(repoErrs) != 0 {
		var errs []error
		for _, e := range repoErrs {
			errs = append(errs, e)
		}
		cmdutils.ExitWithErrs(errs)
	}

	if stats.Repos == 0 {
		cmdutils.ExitWithErr(fmt.Errorf("no git repos found in supplied dir: %v", opts.Dir))
	}
	if stats.SkippedEmptyRepos != 0 {
		fmt.Fprintf(color.Output, "%v", color.YellowString("Warning! Skipped %v empty repos\n", stats.SkippedEmptyRepos))
	}

	fmt.Fprintf(color.Output, "%v", color.GreenString("Finished processing repos %d in %v\n", stats.Repos, time.Since(start)))
}

func runOnDirs(ctx context.Context, wr io.Writer, opts Opts, dir string, start time.Time) (stats Stats, repoErrors []RepoError, rerr error) {

	err := gitrepos.IterDir(dir, 1, func(dir string) error {
		err := runOnRepo(ctx, wr, opts, dir, start)
		stats.Repos += 1
		if err == cmdutils.ErrRevParseFailed {
			stats.SkippedEmptyRepos++
		} else if err != nil {
			re := RepoError{Repo: dir, Err: err}
			repoErrors = append(repoErrors, re)
		}
		return nil
	})
	if err != nil {
		rerr = err
		return
	}
	return
}

var errRevParseFailed = errors.New("git rev-parse HEAD failed")

func runOnRepo(ctx context.Context, wr io.Writer, opts Opts, repoDir string, globalStart time.Time) error {

	return cmdutils.RunOnRepo(ctx, wr, repoDir, func() error {
		res := make(chan ripsrc.Branch)
		done := make(chan bool)

		go func() {
			for branch := range res {
				fmt.Println("[BR]", branch.Name, "commits:", len(branch.Commits))
			}
			done <- true
		}()

		ripOpts := ripsrc.Opts{}
		ripOpts.RepoDir = repoDir
		ripOpts.AllBranches = true
		ripOpts.BranchesUseOrigin = true

		ripper := ripsrc.New(ripOpts)
		err := ripper.Branches(ctx, res)
		<-done

		if err != nil {
			return err
		}

		return nil
	})
}
