package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/pinpt/ripsrc/ripsrc/ripcmd"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:  "ripsrc <dir>",
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		opts := ripcmd.Opts{}
		opts.Dir = args[0]
		opts.CommitFromIncl, _ = cmd.Flags().GetString("sha")
		ripcmd.Run(ctx, os.Stdout, opts)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	RegisterIncBlame()

	//rootCmd.Flags().String("include", "", "include filter as a regular expression")
	//rootCmd.Flags().String("exclude", "", "exclude filter as a regular expression")
	rootCmd.Flags().String("sha", "", "start streaming from sha")
	//rootCmd.Flags().String("profile", "", "one of mem, mutex, cpu, block, trace or empty to disable")
	//rootCmd.Flags().Bool("bares", false, "run dir containing bare repositories")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
