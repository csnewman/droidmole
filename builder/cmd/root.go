package cmd

import (
	"fmt"
	"github.com/csnewman/droidmole/builder/cmd/download"
	"github.com/spf13/cobra"
	"os"
)

var rootCmd = &cobra.Command{
	Use:   "builder",
	Short: "Utilities for building droidmole",
}

func init() {
	rootCmd.CompletionOptions.HiddenDefaultCmd = true
	rootCmd.AddCommand(download.Cmd)
	rootCmd.AddCommand(patchRamdiskCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
