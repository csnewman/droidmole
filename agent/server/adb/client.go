package adb

import (
	"github.com/csnewman/droidmole/agent/util"
	"strings"
)

func SendCommand(cmd []byte) (*RawConnection, error) {
	conn, err := NewRawConnection()
	if err != nil {
		return nil, err
	}

	err = conn.SendCommand(cmd)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return conn, nil
}

func ExecuteCommand(cmd []byte, hasBody bool) ([]byte, error) {
	conn, err := SendCommand(cmd)
	if err != nil {
		return nil, err
	}

	defer conn.Close()

	// Read body
	if !hasBody {
		return nil, nil
	}

	return conn.ReadHexPrefixedBlob()
}

func WaitForEmulator() error {
	conn, err := SendCommand([]byte("host:track-devices"))
	if err != nil {
		return err
	}

	defer conn.Close()

	// TODO: Add timeout
	for {
		msg, err := conn.ReadHexPrefixedBlob()
		if err != nil {
			return err
		}

		lines := util.SplitLines(string(msg))
		if len(lines) == 0 {
			continue
		}

		for _, line := range lines {
			parts := strings.Fields(line)

			if parts[1] == "device" {
				return nil
			}

		}
	}
}

func OpenEmulator() (*RawConnection, error) {
	return SendCommand([]byte("host:transport-local"))
}
