package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/pprof"
	"strings"
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
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		errors := make(chan error, 1)
		go func() {
			for err := range errors {
				cancel()
				fmt.Println(err)
				os.Exit(1)
			}
		}()
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
		dumpmem, _ := cmd.Flags().GetBool("dump-mem")
		if dumpmem {
			go func() {
				var s runtime.MemStats
				var c int
				os.MkdirAll("profile", 0755)
				dump := func() {
					c++
					runtime.ReadMemStats(&s)
					fmt.Println(strings.Repeat("*", 80))
					fmt.Println("Alloc       : ", s.Alloc)
					fmt.Println("Total Alloc : ", s.TotalAlloc)
					fmt.Println("Alive       : ", s.Mallocs-s.Frees)
					fmt.Println(strings.Repeat("*", 80))
					f, _ := os.Create(fmt.Sprintf("profile/profile.%d.pb.gz", c))
					defer f.Close()
					runtime.GC()
					pprof.WriteHeapProfile(f)
				}
				for {
					select {
					case <-time.After(5 * time.Second):
						dump()
					case <-ctx.Done():
						dump()
					}
				}
			}()
		}
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
		}
		var count int
		results := make(chan ripsrc.BlameResult, 10)
		resultsDone := make(chan bool, 1)
		go func() {
			for blame := range results {
				count++
				var license string
				if blame.License != nil {
					license = fmt.Sprintf("%v (%.0f%%)", color.RedString(blame.License.Name), 100*blame.License.Confidence)
				}
				fmt.Printf("[%s] %s language=%s,license=%v,loc=%v,sloc=%v,comments=%v,blanks=%v,complexity=%v,skipped=%v\n", color.CyanString(blame.Commit.SHA[0:8]), color.GreenString(blame.Filename), color.MagentaString(blame.Language), license, blame.Loc, color.YellowString("%v", blame.Sloc), blame.Comments, blame.Comments, blame.Complexity, blame.Skipped)
			}
			resultsDone <- true
		}()
		started := time.Now()
		ripsrc.Rip(ctx, args[0], results, errors, filter)
		<-resultsDone
		fmt.Printf("finished processing %d entries from %d directories in %v\n", count, len(args), time.Since(started))
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	rootCmd.Flags().String("include", "", "include filter as a regular expression")
	rootCmd.Flags().String("exclude", "", "exclude filter as a regular expression")
	rootCmd.Flags().String("sha", "", "start streaming from sha")
	rootCmd.Flags().Bool("dump-mem", false, "dump memory stats every 5 sec")
	rootCmd.Flags().String("profile", "", "one of mem, mutex, cpu, block, trace or empty to disable")
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
