package cmd

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/fatih/color"
	"github.com/pinpt/ripsrc/ripsrc"
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
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
