package repository

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"github.com/schollz/progressbar/v3"
	"io"
	"log"
	"net/http"
	"os"
)

func DownloadArchive(url string, path string, hash string) error {
	log.Println("Downloading", url)

	if _, err := os.Stat(path); err == nil {
		f, err := os.Open(path)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		h := sha1.New()
		if _, err := io.Copy(h, f); err != nil {
			return err
		}

		computed := fmt.Sprintf("%x", h.Sum(nil))

		if computed == hash {
			log.Println("Skipping: File exists and hash matches")
			return nil
		}

		log.Println("File exist with incorrect hash. Re-downloading")
	} else if errors.Is(err, os.ErrNotExist) {
		// Fallthrough
	} else {
		return err
	}

	out, err := os.Create(path + ".tmp")
	if err != nil {
		return err
	}

	resp, err := http.Get(url)
	if err != nil {
		out.Close()
		return err
	}
	defer resp.Body.Close()

	h := sha1.New()

	bar := progressbar.DefaultBytes(resp.ContentLength)
	io.Copy(io.MultiWriter(out, bar, h), resp.Body)

	// Ensure closed before rename
	out.Close()

	computed := fmt.Sprintf("%x", h.Sum(nil))

	if computed != hash {
		log.Println("Hash mismatch")
		return errors.New("download hash mismatch")
	}

	return os.Rename(path+".tmp", path)
}
