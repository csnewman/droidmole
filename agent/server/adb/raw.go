package adb

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
)

// RawConnection based on https://android.googlesource.com/platform/packages/modules/adb/+/HEAD/OVERVIEW.TXT
type RawConnection struct {
	conn net.Conn
}

func NewRawConnection() (*RawConnection, error) {
	conn, err := net.Dial("tcp", ":5037")
	if err != nil {
		return nil, err
	}

	return &RawConnection{conn: conn}, nil
}

func (c *RawConnection) Close() error {
	err := c.conn.Close()
	// Ignore network closed error
	if err != nil && errors.Is(err, net.ErrClosed) {
		return nil
	}
	return err
}

func (c *RawConnection) WriteRaw(packet []byte) error {
	wrote, err := c.conn.Write(packet)
	if err != nil {
		c.Close()
		return err
	}

	if wrote != len(packet) {
		c.Close()
		// TODO: Retry?
		return errors.New("failed to write full packet")
	}

	return nil
}

func (c *RawConnection) WriteMessage(msg []byte) error {
	// Determine max message length
	packet := []byte(fmt.Sprintf("%04x%s", len(msg), msg))
	err := c.WriteRaw(packet)
	if err != nil {
		c.Close()
	}

	return err
}

func (c *RawConnection) ReadRaw(blob []byte) error {
	_, err := io.ReadFull(c.conn, blob)
	if err != nil {
		c.Close()
		return err
	}

	return nil
}

func (c *RawConnection) ReadStatus() (string, error) {
	resp := make([]byte, 4)
	_, err := io.ReadFull(c.conn, resp)
	if err != nil {
		c.Close()
		return "", err
	}

	return string(resp), nil
}

func (c *RawConnection) ReadHexPrefixedBlob() ([]byte, error) {
	sizeBlob := make([]byte, 4)
	_, err := io.ReadFull(c.conn, sizeBlob)
	if err != nil {
		c.Close()
		return nil, err
	}

	size, err := strconv.ParseInt(string(sizeBlob), 16, 64)
	if err != nil {
		c.Close()
		return nil, err
	}

	blob := make([]byte, size)
	_, err = io.ReadFull(c.conn, blob)
	if err != nil {
		c.Close()
		return nil, err
	}

	return blob, nil
}

func (c *RawConnection) ReadShellBlob() (byte, []byte, error) {
	header := make([]byte, 5)
	_, err := io.ReadFull(c.conn, header)
	if err != nil {
		c.Close()
		return 0, nil, err
	}

	id := header[0]
	size := binary.LittleEndian.Uint32(header[1:])

	blob := make([]byte, size)
	_, err = io.ReadFull(c.conn, blob)
	if err != nil {
		c.Close()
		return 0, nil, err
	}

	return id, blob, nil
}

func (c *RawConnection) WriteShellBlob(id byte, blob []byte) error {
	header := make([]byte, 5)
	header[0] = id
	binary.LittleEndian.PutUint32(header[1:], uint32(len(blob)))

	err := c.WriteRaw(header)
	if err != nil {
		c.Close()
		return err
	}

	err = c.WriteRaw(blob)
	if err != nil {
		c.Close()
		return err
	}

	return nil
}

func (c *RawConnection) SendCommand(cmd []byte) error {
	err := c.WriteMessage(cmd)
	if err != nil {
		c.Close()
		return err
	}

	status, err := c.ReadStatus()
	if err != nil {
		c.Close()
		return err
	}

	if status != "OKAY" {
		// Read error
		errBlob, err := c.ReadHexPrefixedBlob()
		if err != nil {
			c.Close()
			return err
		}

		return errors.New(fmt.Sprint("server error: ", string(errBlob)))
	}

	return nil
}
