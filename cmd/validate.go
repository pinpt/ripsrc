package cmd

import (
	"fmt"
	"strings"

	"github.com/pinpt/ripsrc/ripsrc/gitblame2"

	"github.com/pinpt/ripsrc/ripsrc/history3/process"
	"github.com/spf13/cobra"
)

/*
var manuallyChecked = map[string]bool{
	// https://github.com/vuejs/vue repo ~3k commits ~1k pull reqs
	// block is repeated and both attributions are ok
	"test/unit/features/transition/transition.spec.js":          true,
	"test/unit/features/component/component-keep-alive.spec.js": true,

	// 50257a58810a83b27316a793adf8d59a81ef3cf0
	"benchmarks/ssr/common.js": true,

	// bc2918f0e596d0e133a25606cbb66075402ce6c3
	"packages/vue-template-compiler/build.js": true,

	// copy of file is crete with one line diff
	// incremental blame condiders it a new file
	// regular git blame considers it copy + change in another
	// both files exist
	"test/ssr/fixtures/async-bar.js": true,
}
*/

var manuallyChecked = map[string]bool{}

var validateIncBlameCmd = &cobra.Command{
	Use:  "validate_inc_blame <repodirs...>",
	Args: cobra.RangeArgs(1, 999),
	Run: func(cmd *cobra.Command, args []string) {
		for _, repoDir := range args {
			fmt.Println("running on repo", repoDir)

			opts := process.Opts{}
			opts.RepoDir = repoDir
			opts.CommitFromIncl, _ = cmd.Flags().GetString("commit-from-incl")

			pr := process.New(opts)

			res := make(chan process.Result)
			done := make(chan bool)
			go func() {
				for r := range res {
					fmt.Println("Checking commit:", r.Commit)

				LOOPFILES:
					for p, bl1 := range r.Files {
						fmt.Println("Checking file:", r.Commit, p)
						if manuallyChecked[p] {
							fmt.Println("Skipping checking file (manully checked to be ok to differ):", p)
							continue
						}
						if len(bl1.Lines) == 0 {
							// removed file, no blame
							continue
						}
						bl2, err := gitblame2.Run(repoDir, r.Commit, p)
						if err != nil {
							panic(err)
						}
						rerr := func() {
							fmt.Println("Failed checks.")
							fmt.Println("Content of incremental blame")
							fmt.Println(bl1)
							fmt.Println("Content of git blame")
							fmt.Println(bl2)
							fmt.Println("ERROR!")
						}
						// validate
						if len(bl1.Lines) != len(bl2.Lines) {
							fmt.Println("Error: lines len mismatch")
							rerr()
							continue
						}
						for i := range bl1.Lines {
							l1 := bl1.Lines[i]
							l2 := bl2.Lines[i]
							// TODO: looks like currently we convert newlines to linux style, even if it was different in source
							c1 := strings.TrimSpace(string(l1.Line))
							c2 := strings.TrimSpace(l2.Content)
							if l1.Commit != l2.CommitHash {
								// repeating blocks are in sometime in different order when both ordering is correct
								mostlySameContent := false
								if c1 == c2 {
									if len(c1) <= 10 {
										mostlySameContent = true
									}
									if c1 == "return" {
										mostlySameContent = true
									}
								}
								// the new blame sometimes assigned ownership to lines in a different order, happens for repeating lines, such as empty lines, lines, containing braces, etc
								// ignore diff for lines with 1 char only
								if !mostlySameContent {
									fmt.Println("invalid commit, line ", i)
									continue LOOPFILES
								}
							}
							if c1 != c2 {
								fmt.Println("invalid content, line ", i)
								continue LOOPFILES
							}
						}

					}
				}
				done <- true
			}()
			err := pr.Run(res)
			if err != nil {
				panic(err)
			}
			<-done
			fmt.Println("SUCCESS on", repoDir)
		}
	},
}

func RegisterIncBlame() {
	cmd := validateIncBlameCmd
	cmd.Flags().String("commit-from-incl", "", "start from specific commit (inclusive)")
	rootCmd.AddCommand(cmd)
}
