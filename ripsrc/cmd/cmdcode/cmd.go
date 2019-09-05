package cmdcode

import (
	"context"
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

	// CommitFromIncl starts from specific commit (inclusive). May also include some previous commits.
	CommitFromIncl string

	// Profile set to one of mem, mutex, cpu, block, trace to enable profiling.
	Profile string
}

type Stats struct {
	Repos             int
	SkippedEmptyRepos int
	Entries           int
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

	fmt.Fprintf(color.Output, "%v", color.GreenString("Finished processing repos %d entries %d in %v\n", stats.Repos, stats.Entries, time.Since(start)))
}

func runOnDirs(ctx context.Context, wr io.Writer, opts Opts, dir string, start time.Time) (stats Stats, repoErrors []RepoError, rerr error) {

	err := gitrepos.IterDir(dir, 1, func(dir string) error {
		entries, err := runOnRepo(ctx, wr, opts, dir, start)
		stats.Repos += 1
		stats.Entries += entries
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

func runOnRepo(ctx context.Context, wr io.Writer, opts Opts, repoDir string, globalStart time.Time) (entries int, _ error) {

	err := cmdutils.RunOnRepo(ctx, wr, repoDir, func() error {
		res := make(chan ripsrc.CommitCode)
		done := make(chan bool)

		go func() {

			for commit := range res {
				fmt.Println(commit.SHA, commit.Date)
				for blame := range commit.Files {
					entries++
					var license string
					if blame.License != nil {
						license = fmt.Sprintf("%v (%.0f%%)", color.RedString(blame.License.Name), 100*blame.License.Confidence)
					}
					timeSinceStartMin := int(time.Since(globalStart).Minutes())
					fmt.Fprintf(color.Output, "[%s][%s][%sm] %s language=%s,license=%v,loc=%v,sloc=%v,comments=%v,blanks=%v,complexity=%v,skipped=%v,status=%s,author=%s\n", color.YellowString("%v", repoDir), color.CyanString(blame.Commit.SHA[0:8]), color.YellowString("%v", timeSinceStartMin), color.GreenString(blame.Filename), color.MagentaString(blame.Language), license, blame.Loc, color.YellowString("%v", blame.Sloc), blame.Comments, blame.Comments, blame.Complexity, blame.Skipped, blame.Commit.Files[blame.Filename].Status, blame.Commit.Author())

				}
			}
			done <- true
		}()

		ripOpts := ripsrc.Opts{}
		ripOpts.RepoDir = repoDir
		ripOpts.CommitFromIncl = opts.CommitFromIncl
		ripOpts.NoStrictResume = true

		ripper := ripsrc.New(ripOpts)
		err := ripper.CodeByCommit(ctx, res)
		<-done

		if err != nil {
			return err
		}

		fmt.Fprintln(wr)
		ripper.GitProcessTimings.OutputStats(wr)
		fmt.Fprintln(wr)
		ripper.CodeInfoTimings.OutputStats(wr)
		fmt.Fprintln(wr)

		fmt.Fprintf(wr, "%d entries processed\n", entries)

		return nil
	})
	return entries, err
}
