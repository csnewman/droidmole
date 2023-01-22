package cmd

import "github.com/spf13/cobra"

var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download components",
}

func init() {
	downloadCmd.AddCommand(sysimgCmd)
	downloadCmd.AddCommand(emulatorCmd)
}
