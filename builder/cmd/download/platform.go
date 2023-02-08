package download

import (
	"github.com/csnewman/droidmole/builder/repository"
	"github.com/spf13/cobra"
	"log"
)

var platformCmd = &cobra.Command{
	Use:   "platform-tools",
	Short: "Download platform tools",
	Run:   executePlatform,
}

var pltChannel string
var pltOutput string
var pltHostOs string

func init() {
	platformCmd.Flags().StringVar(&emuHostOs, "host-os", "linux", "OS (linux, windows, macosx)")
	platformCmd.Flags().StringVar(&emuChannel, "channel", "channel-0", "Release Channel (e.g. channel-0)")
	platformCmd.Flags().StringVar(&emuOutput, "output", "", "Destination File")
	platformCmd.MarkFlagRequired("output")
}

func executePlatform(cmd *cobra.Command, args []string) {
	log.Println("DroidMole Builder")

	manifest, err := repository.GetManifest(toolRepo)
	if err != nil {
		log.Fatal(err)
	}

	for _, rpkg := range manifest.RemotePackages {
		if rpkg.ChannelRef.Ref != emuChannel {
			continue
		}

		if rpkg.Path != "platform-tools" {
			continue
		}

		for _, archive := range rpkg.Archives.Archive {
			if archive.HostOs != emuHostOs {
				continue
			}

			log.Println("Selected", rpkg.Path)
			log.Println("  Display:", rpkg.DisplayName)
			log.Println(" Revision:", rpkg.Revision.Major)
			log.Println("  Channel:", rpkg.ChannelRef.Ref)
			log.Println("   HostOS:", archive.HostOs)

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
