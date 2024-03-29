package adb

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapio"
	"os"
	"os/exec"
	"path/filepath"
)

type Adb interface {
	StartServer() error
	SendCommand(cmd []byte) (RawConnection, error)
	ExecuteCommand(cmd []byte, hasBody bool) ([]byte, error)
	WaitForEmulator() error
	OpenEmulator() (RawConnection, error)
	ListDirectory(path string) ([]ListDirectoryEntry, error)
	StatFile(path string, followLinks bool) (uint32, *FileStat, error)
	PullFile(path string) (*PullFileStream, error)
	PushFile(path string, mode uint32) (*PushFileStream, error)
}

type systemImpl struct {
	log        *zap.SugaredLogger
	rawFactory RawConnectionFactory
}

func New(log *zap.SugaredLogger, rawFactory RawConnectionFactory) Adb {
	return &systemImpl{
		log:        log,
		rawFactory: rawFactory,
	}
}

func (s *systemImpl) StartServer() error {
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

	childLogger := s.log.Named("adb").Desugar()
	logWriter := &zapio.Writer{
		Log: childLogger,
	}

	// Generate key
	s.log.Info("Generating ADB key")
	cmd := exec.Command("/android/platform-tools/adb", "keygen", keyPath)
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter
	err = cmd.Run()
	if err != nil {
		return err
	}

	err = os.Chmod(keyPath, 0600)
	if err != nil {
		return err
	}

	// Start server
	s.log.Info("Starting ADB server")
	cmd = exec.Command("/android/platform-tools/adb", "start-server")
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter
	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil
}
