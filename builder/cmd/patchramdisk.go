package cmd

import (
	"bufio"
	"bytes"
	"errors"
	"github.com/pierrec/lz4/v4"
	"github.com/spf13/cobra"
	"github.com/u-root/u-root/pkg/cpio"
	"io"
	"log"
	"os"
	"path"
	"strings"
)

var patchRamdiskCmd = &cobra.Command{
	Use:   "patch-ramdisk",
	Short: "Replace the kernel modules in a ramdisk image",
	RunE:  executePatchRamdisk,
}

var ramdiskInput string
var ramdiskModules string
var ramdiskOutput string

func init() {
	patchRamdiskCmd.Flags().StringVar(&ramdiskInput, "input", "", "Input File")
	patchRamdiskCmd.Flags().StringVar(&ramdiskModules, "modules", "", "Directory containing replacement modules")
	patchRamdiskCmd.Flags().StringVar(&ramdiskOutput, "output", "", "Destination File")
	patchRamdiskCmd.MarkFlagRequired("input")
	patchRamdiskCmd.MarkFlagRequired("modules")
	patchRamdiskCmd.MarkFlagRequired("output")
}

func executePatchRamdisk(cmd *cobra.Command, args []string) error {
	log.Println("DroidMole Builder")

	// Read compressed
	log.Println("Reading", ramdiskInput)
	compressedRamdisk, err := os.ReadFile(ramdiskInput)
	if err != nil {
		return err
	}

	// Decompress
	log.Println("Decompressing")
	reader := bytes.NewReader(compressedRamdisk)
	lz4Reader := lz4.NewReader(reader)
	ramdisk, err := io.ReadAll(lz4Reader)
	if err != nil {
		return err
	}

	// Split
	log.Println("Processing")
	segements := bytes.Split(ramdisk, []byte("TRAILER!!!\000"))

	var outputBuffer bytes.Buffer

	for i, segment := range segements {
		// Ignore empty padding at end of file
		if isAllZero(segment) {
			break
		}

		log.Println("Processing segment", i)

		// Remove padding at start
		startPos := bytes.Index(segment, []byte("070701"))
		if startPos == -1 {
			return errors.New("invalid ramdisk format")
		}

		segment = segment[startPos:]

		newSegment, err := processSegment(segment)
		if err != nil {
			return err
		}

		outputBuffer.Write(newSegment)

		// Pad
		length := outputBuffer.Len()
		padding := 256 - length%256
		for i := 0; i < padding; i++ {
			outputBuffer.WriteByte(0)
		}
	}

	// Output
	log.Println("Storing new image", ramdiskOutput)
	err = os.WriteFile(ramdiskOutput, outputBuffer.Bytes(), 0777)
	if err != nil {
		return err
	}

	log.Println("Patched")
	return nil
}

func processSegment(segment []byte) ([]byte, error) {
	// Parse input archive
	reader := cpio.Newc.Reader(bytes.NewReader(segment))
	inputArchive, err := cpio.ArchiveFromReader(reader)
	if err != nil {
		return nil, err
	}

	// Prepare new output
	var outputBuffer bytes.Buffer
	outputWriter := bufio.NewWriter(&outputBuffer)
	outputArchive := cpio.Newc.Writer(outputWriter)
	lastIno := uint64(0)

	// Process archive
	hasModules := false
	oldModules := make(map[string]bool)

	for _, name := range inputArchive.Order {
		rec := inputArchive.Files[name]

		// Check if the archive contains modules
		if strings.HasPrefix(name, "lib/modules/") && strings.HasSuffix(name, ".ko") {
			hasModules = true
			oldModules[strings.TrimPrefix(name, "lib/modules/")] = true
			continue
		}

		// Find the last used ino
		if rec.Ino > lastIno {
			lastIno = rec.Ino
		}

		// Pass through file
		err = outputArchive.WriteRecord(rec)
		if err != nil {
			return nil, err
		}
	}

	// If this ramdisk contained modules, replace them
	if hasModules {
		for name, _ := range oldModules {
			log.Println(" - Replacing", name)

			// Read module
			// Fine to read into memory as modules are small
			newFile, err := os.Open(path.Join(ramdiskModules, name))
			if err != nil {
				log.Fatal(err)
			}

			data, err := io.ReadAll(newFile)

			newFile.Close()

			if err != nil {
				return nil, err
			}

			// Allocate new ino
			lastIno += 1

			// Write new module
			err = outputArchive.WriteRecord(cpio.Record{
				ReaderAt: bytes.NewReader(data),
				Info: cpio.Info{
					Ino:      lastIno,
					Mode:     0100644,
					UID:      0,
					GID:      0,
					NLink:    1,
					MTime:    0,
					FileSize: uint64(len(data)),
					Dev:      0,
					Major:    0,
					Minor:    0,
					Rmajor:   0,
					Rminor:   0,
					Name:     "lib/modules/" + name,
				},
				RecPos:  0,
				RecLen:  0,
				FilePos: 0,
			})
			if err != nil {
				return nil, err
			}
		}
	}

	// Complete archive
	err = cpio.WriteTrailer(outputArchive)
	if err != nil {
		return nil, err
	}

	err = outputWriter.Flush()
	if err != nil {
		return nil, err
	}

	return outputBuffer.Bytes(), nil
}

func isAllZero(s []byte) bool {
	for _, v := range s {
		if v != 0 {
			return false
		}
	}
	return true
}
