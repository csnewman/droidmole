package adb

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func StartServer() error {
	// Ensure android directory exists
	homedir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Join(homedir, ".android"), 0644)
	if err != nil {
		return err
	}

	// Regenerate adb key
	keyPath := filepath.Join(homedir, ".android/adbkey")

	// Generate ley
	log.Println("Generating ADB key")
	cmd := exec.Command("/android/platform-tools/adb", "keygen", keyPath)
	cmd.Stdout = log.Writer()
	cmd.Stderr = log.Writer()
	err = cmd.Run()
	if err != nil {
		return err
	}

	err = os.Chmod(keyPath, 0600)
	if err != nil {
		return err
	}

	// Start server
	log.Println("Starting ADB server")
	cmd = exec.Command("/android/platform-tools/adb", "start-server")
	cmd.Stdout = log.Writer()
	cmd.Stderr = log.Writer()
	err = cmd.Run()
	if err != nil {
		return err
	}

	log.Println("Starting socat")
	cmd = exec.Command("socat", "-d", "tcp-listen:8037,reuseaddr,fork", "tcp:127.0.0.1:5037")
	cmd.Stdout = log.Writer()
	cmd.Stderr = log.Writer()
	err = cmd.Start()
	if err != nil {
		return err
	}

	return nil
}
