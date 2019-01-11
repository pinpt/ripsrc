package cmd

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/fatih/color"
	"github.com/pinpt/ripsrc/ripsrc"
	"github.com/pkg/profile"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:  "ripsrc <dir>",
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("starting ripsrc")
		started := time.Now()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		// potentially enable profiling
		p, _ := cmd.Flags().GetString("profile")
		var dir string
		if p != "" {
			dir, _ = ioutil.TempDir("", "profile")
			defer func() {
				fn := filepath.Join(dir, p+".pprof")
				abs, _ := filepath.Abs(os.Args[0])
				fmt.Printf("to view profile, run `go tool pprof --pdf %s %s`\n", abs, fn)
			}()
			switch p {
			case "cpu":
				{
					defer profile.Start(profile.CPUProfile, profile.ProfilePath(dir), profile.Quiet).Stop()
				}
			case "mem":
				{
					defer profile.Start(profile.MemProfile, profile.ProfilePath(dir), profile.Quiet).Stop()
				}
			case "trace":
				{
					defer profile.Start(profile.TraceProfile, profile.ProfilePath(dir), profile.Quiet).Stop()
				}
			case "block":
				{
					defer profile.Start(profile.BlockProfile, profile.ProfilePath(dir), profile.Quiet).Stop()
				}
			case "mutex":
				{
					defer profile.Start(profile.MutexProfile, profile.ProfilePath(dir), profile.Quiet).Stop()
				}
			default:
				{
					panic("unexpected profile: " + p)
				}
			}
		}
		/*
			var filter *ripsrc.Filter
			include, _ := cmd.Flags().GetString("include")
			exclude, _ := cmd.Flags().GetString("exclude")
			sha, _ := cmd.Flags().GetString("sha")
			if include != "" || exclude != "" || sha != "" {
				filter = &ripsrc.Filter{}
				if include != "" {
					filter.Whitelist = regexp.MustCompile(include)
				}
				if exclude != "" {
					filter.Blacklist = regexp.MustCompile(exclude)
				}
				if sha != "" {
					filter.SHA = sha
				}
			}*/

		count := 0
		totalRepos := 0
		createRepoProcessor := func(repo string, printname bool) (chan bool, chan ripsrc.BlameResult) {
			resultsDone := make(chan bool, 1)
			results := make(chan ripsrc.BlameResult, 10)
			go func() {
				var repostr string
				if printname {
					repostr = color.HiYellowString("%s ", repo)
				}
				for blame := range results {
					count++
					var license string
					if blame.License != nil {
						license = fmt.Sprintf("%v (%.0f%%)", color.RedString(blame.License.Name), 100*blame.License.Confidence)
					}
					fmt.Fprintf(color.Output, "%s[%s] %s language=%s,license=%v,loc=%v,sloc=%v,comments=%v,blanks=%v,complexity=%v,skipped=%v,status=%s,author=%s\n", repostr, color.CyanString(blame.Commit.SHA[0:8]), color.GreenString(blame.Filename), color.MagentaString(blame.Language), license, blame.Loc, color.YellowString("%v", blame.Sloc), blame.Comments, blame.Comments, blame.Complexity, blame.Skipped, blame.Commit.Files[blame.Filename].Status, blame.Commit.Author())
				}
				resultsDone <- true
			}()
			return resultsDone, results
		}

		bares, _ := cmd.Flags().GetBool("bares")
		if bares {
			entries, err := ioutil.ReadDir(args[0])
			if err != nil {
				panic(err)
			}
			for _, entry := range entries {
				entryName := entry.Name()
				if !entry.IsDir() || filepath.Ext(entryName) != ".git" {
					continue
				}
				resultsDone, results := createRepoProcessor(entry.Name(), true)
				start := count
				ripper := ripsrc.New()
				localstart := time.Now()
				totalRepos++
				if err := ripper.Rip(ctx, filepath.Join(args[0], entry.Name()), results); err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
				<-resultsDone
				fmt.Fprintf(color.Output, "finished repo processing for %v in %v. %d entries processed\n", color.HiGreenString(entryName), time.Since(localstart), count-start)
			}

			fmt.Printf("finished processing %d entries from %d directories (%v repos) in %v\n", count, len(args), totalRepos, time.Since(started))

			return
		}

		if f, err := os.Stat(filepath.Join(args[0], ".git")); err == nil && f.IsDir() {
			resultsDone, results := createRepoProcessor("", false)
			ripper := ripsrc.New()
			totalRepos++
			if err := ripper.Rip(ctx, args[0], results); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			<-resultsDone
			outputStats(ripper, color.Output)
		} else {
			files, err := ioutil.ReadDir(args[0])
			if err != nil {
				panic(err)
			}
			for _, dir := range files {
				fd, _ := filepath.Abs(filepath.Join(args[0], dir.Name(), ".git"))
				if _, err := os.Stat(fd); err == nil {
					name := filepath.Base(filepath.Dir(filepath.Dir(fd))) + "/" + filepath.Base(filepath.Dir(fd))
					resultsDone, results := createRepoProcessor(name, true)
					fmt.Fprintf(color.Output, "starting repo processing for %v\n", color.HiGreenString(name))
					start := count
					ripper := ripsrc.New()
					localstart := time.Now()
					totalRepos++
					if err := ripper.Rip(ctx, filepath.Dir(fd), results); err != nil {
						fmt.Println(err)
						os.Exit(1)
					}
					<-resultsDone
					fmt.Fprintf(color.Output, "finished repo processing for %v in %v. %d entries processed\n", color.HiGreenString(name), time.Since(localstart), count-start)

				}
			}
		}

		fmt.Printf("finished processing %d entries from %d directories (%v repos) in %v\n", count, len(args), totalRepos, time.Since(started))
	},
}

func outputStats(ripper *ripsrc.Ripper, wr io.Writer) {
	ripper.GitProcessTimings.OutputStats(wr)
	fmt.Fprintln(wr)
	ripper.CodeInfoTimings.OutputStats(wr)
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	RegisterIncBlame()

	rootCmd.Flags().String("include", "", "include filter as a regular expression")
	rootCmd.Flags().String("exclude", "", "exclude filter as a regular expression")
	rootCmd.Flags().String("sha", "", "start streaming from sha")
	rootCmd.Flags().String("profile", "", "one of mem, mutex, cpu, block, trace or empty to disable")
	rootCmd.Flags().Bool("bares", false, "run dir containing bare repositories")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
