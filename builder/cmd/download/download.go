package download

import "github.com/spf13/cobra"

var Cmd = &cobra.Command{
	Use:   "download",
	Short: "Download components",
}

func init() {
	Cmd.AddCommand(sysimgCmd)
	Cmd.AddCommand(emulatorCmd)
	Cmd.AddCommand(platformCmd)
}
