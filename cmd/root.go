package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/pinpt/ripsrc/ripsrc/cmd/cmdbranches"
	"github.com/pinpt/ripsrc/ripsrc/cmd/cmdcode"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use: "ripsrc",
}

var codeCmd = &cobra.Command{
	Use:   "code <dir>",
	Short: "Extracts code information from repos in a directory",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		opts := cmdcode.Opts{}
		opts.Dir = args[0]
		opts.CommitFromIncl, _ = cmd.Flags().GetString("sha")
		opts.Profile, _ = cmd.Flags().GetString("profile")
		cmdcode.Run(ctx, os.Stdout, opts)
	},
}

var branchesCmd = &cobra.Command{
	Use:   "branches <dir>",
	Short: "Extracts information about branches from repos in a directory",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		opts := cmdbranches.Opts{}
		opts.Dir = args[0]
		opts.Profile, _ = cmd.Flags().GetString("profile")
		cmdbranches.Run(ctx, os.Stdout, opts)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {

	RegisterIncBlame()

	codeCmd.Flags().String("sha", "", "start streaming from sha")
	codeCmd.Flags().String("profile", "", "one of mem, mutex, cpu, block, trace or empty to disable")
	rootCmd.AddCommand(codeCmd)

	branchesCmd.Flags().String("profile", "", "one of mem, mutex, cpu, block, trace or empty to disable")
	rootCmd.AddCommand(branchesCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
