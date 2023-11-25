package download

import (
	"github.com/csnewman/droidmole/builder/repository"
	"github.com/spf13/cobra"
	"log"
)

const defaultRepo = "https://dl.google.com/android/repository/sys-img/android/sys-img2-1.xml"
const googleRepo = "https://dl.google.com/android/repository/sys-img/google_apis/sys-img2-1.xml"
const playstoreRepo = "https://dl.google.com/android/repository/sys-img/google_apis_playstore/sys-img2-1.xml"

var sysimgCmd = &cobra.Command{
	Use:   "sysimg",
	Short: "Download a Android sysimg",
	Run:   executeSysimg,
}

var sysimgType string
var sysimgApi string
var sysimgAbi string
var sysimgChannel string
var sysimgOutput string

func init() {
	sysimgCmd.Flags().StringVar(&sysimgType, "type", "", "Type (default, google, playstore)")
	sysimgCmd.Flags().StringVar(&sysimgApi, "api", "", "API Level (e.g. 33)")
	sysimgCmd.Flags().StringVar(&sysimgAbi, "abi", "x86_64", "ABI (e.g. x86_64)")
	sysimgCmd.Flags().StringVar(&sysimgChannel, "channel", "channel-0", "Release Channel (e.g. channel-0)")
	sysimgCmd.Flags().StringVar(&sysimgOutput, "output", "", "Destination File")
	sysimgCmd.MarkFlagRequired("type")
	sysimgCmd.MarkFlagRequired("api")
	sysimgCmd.MarkFlagRequired("output")
}

func executeSysimg(cmd *cobra.Command, args []string) {
	log.Println("DroidMole Builder")

	var manifestUrl string
	if sysimgType == "default" {
		manifestUrl = defaultRepo
	} else if sysimgType == "google" {
		manifestUrl = googleRepo
	} else if sysimgType == "playstore" {
		manifestUrl = playstoreRepo
	} else {
		log.Fatal("Unknown repo", sysimgType)
	}

	manifest, err := repository.GetManifest(manifestUrl)
	if err != nil {
		log.Fatal(err)
	}

	currentMajor := -1
	var rpkg repository.RemotePackage
	for _, remotePackage := range manifest.RemotePackages {
		if remotePackage.ChannelRef.Ref != sysimgChannel {
			continue
		}

		if remotePackage.TypeDetails.Abi != sysimgAbi {
			continue
		}

		if remotePackage.TypeDetails.ApiLevel != sysimgApi {
			continue
		}

		if remotePackage.Revision.Major < currentMajor {
			continue
		}

		currentMajor = remotePackage.Revision.Major
		rpkg = remotePackage
	}

	if currentMajor == -1 {
		log.Fatal("Failed to find sysimg")
	}

	log.Println("Selected image", rpkg.Path)
	log.Println("  Display:", rpkg.DisplayName)
	log.Println(" Revision:", rpkg.Revision.Major)
	log.Println(" ApiLevel:", rpkg.TypeDetails.ApiLevel)
	log.Println("      Abi:", rpkg.TypeDetails.Abi)
	log.Println("  Channel:", rpkg.ChannelRef.Ref)

	for _, archive := range rpkg.Archives.Archive {
		if archive.HostOs != "" && archive.HostOs != "linux" {
			continue
		}

		url := "https://dl.google.com/android/repository/sys-img/" + rpkg.TypeDetails.Tag.Id + "/" + archive.Complete.Url

		if err := repository.DownloadArchive(url, sysimgOutput, archive.Complete.Checksum); err != nil {
			log.Fatal(err)
		}

		log.Println("Complete")

		return
	}

	log.Fatal("Failed to find download")
}
