package cmd

import (
	"fmt"
	"github.com/Argeric/neurahive-scan-backend/config"
	"github.com/spf13/cobra"
	"os"
)

var (
	flagVersion bool

	rootCmd = &cobra.Command{
		Use:   "neurahive",
		Short: "Web3 storage scan on Neurahive.",
		Run:   start,
	}
)

func init() {
	// print version and exit
	rootCmd.Flags().BoolVarP(
		&flagVersion, "version", "v", false, "If true, print version and exit",
	)
}

func start(cmd *cobra.Command, args []string) {
	if flagVersion {
		config.DumpVersionInfo()
		return
	}
}

// Execute is the command line entrypoint.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
