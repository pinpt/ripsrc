package cmd

import (
	"fmt"
	"os"

	"github.com/pinpt/ripsrc/ripsrc/gitblame2"

	"github.com/pinpt/ripsrc/ripsrc/history3/process"
	"github.com/spf13/cobra"
)

var validateIncBlameCmd = &cobra.Command{
	Use:  "validate_inc_blame <repodirs...>",
	Args: cobra.RangeArgs(1, 999),
	Run: func(cmd *cobra.Command, args []string) {
		for _, repoDir := range args {
			fmt.Println("running on repo", repoDir)

			pr := process.New(process.Opts{RepoDir: repoDir})

			res := make(chan process.Result)
			done := make(chan bool)
			go func() {
				for r := range res {
					fmt.Println("Checking commit:", r.Commit)
					for p, bl1 := range r.Files {
						fmt.Println("Checking file:", r.Commit, p)
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
							os.Exit(1)
						}
						// validate
						if len(bl1.Lines) != len(bl2.Lines) {
							fmt.Println("Error: lines len mismatch")
							rerr()
						}
						for i := range bl1.Lines {
							l1 := bl1.Lines[i]
							l2 := bl2.Lines[i]
							if l1.Commit != l2.CommitHash {
								fmt.Println("invalid commit")
								rerr()
							}
							if string(l1.Line) != l2.Content {
								fmt.Println("invalid content")
								rerr()
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
	rootCmd.AddCommand(cmd)
}
