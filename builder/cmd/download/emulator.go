package download

import (
	"github.com/csnewman/droidmole/builder/repository"
	"github.com/spf13/cobra"
	"log"
)

const toolRepo = "https://dl.google.com/android/repository/repository2-3.xml"

var emulatorCmd = &cobra.Command{
	Use:   "emulator",
	Short: "Download the Android emulator",
	Run:   executeEmulator,
}

var (
	emuChannel  string
	emuOutput   string
	emuHostOs   string
	emuHostArch string
)

func init() {
	emulatorCmd.Flags().StringVar(&emuHostOs, "host-os", "linux", "OS (linux, windows, macosx)")
	emulatorCmd.Flags().StringVar(&emuHostArch, "host-arch", "x86", "OS Arch (x86, aarch64)")
	emulatorCmd.Flags().StringVar(&emuChannel, "channel", "channel-0", "Release Channel (e.g. channel-0)")
	emulatorCmd.Flags().StringVar(&emuOutput, "output", "", "Destination File")
	emulatorCmd.MarkFlagRequired("output")
}

func executeEmulator(cmd *cobra.Command, args []string) {
	log.Println("DroidMole Builder")

	manifest, err := repository.GetManifest(toolRepo)
	if err != nil {
		log.Fatal(err)
	}

	for _, rpkg := range manifest.RemotePackages {
		if rpkg.ChannelRef.Ref != emuChannel {
			continue
		}

		if rpkg.Path != "emulator" {
			continue
		}

		for _, archive := range rpkg.Archives.Archive {
			if archive.HostOs != emuHostOs {
				continue
			}

			if archive.HostArch != emuHostArch {
				continue
			}

			log.Println("Selected", rpkg.Path)
			log.Println("  Display:", rpkg.DisplayName)
			log.Println(" Revision:", rpkg.Revision.Major)
			log.Println("  Channel:", rpkg.ChannelRef.Ref)
			log.Println("   HostOS:", archive.HostOs)
			log.Println(" HostArch:", archive.HostArch)

			url := "https://dl.google.com/android/repository/" + archive.Complete.Url

			if err := repository.DownloadArchive(url, emuOutput, archive.Complete.Checksum); err != nil {
				log.Fatal(err)
			}

			log.Println("Complete")

			return
		}
	}

	log.Fatal("Failed to find download")
}
