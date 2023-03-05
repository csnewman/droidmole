package adb

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
)

// RawConnection based on https://android.googlesource.com/platform/packages/modules/adb/+/HEAD/OVERVIEW.TXT
type RawConnection interface {
	Close() error
	WriteRaw(packet []byte) error
	WriteMessage(msg []byte) error
	ReadRaw(blob []byte) error
	ReadStatus() (string, error)
	ReadHexPrefixedBlob() ([]byte, error)
	ReadShellBlob() (byte, []byte, error)
	ReadLine() (*string, error)
	WriteShellBlob(id byte, blob []byte) error
	SendCommand(cmd []byte) error
}

type RawConnectionFactory interface {
	NewRawConnection() (RawConnection, error)
}

type networkRawConn struct {
	conn net.Conn
}

type networkRawConnFactory struct{}

func NewRawConnectionFactory() RawConnectionFactory {
	return &networkRawConnFactory{}
}

func (_ networkRawConnFactory) NewRawConnection() (RawConnection, error) {
	conn, err := net.Dial("tcp", ":5037")
	if err != nil {
		return nil, err
	}

	return &networkRawConn{conn: conn}, nil
}

func (c *networkRawConn) Close() error {
	err := c.conn.Close()
	// Ignore network closed error
	if err != nil && errors.Is(err, net.ErrClosed) {
		return nil
	}
	return err
}

func (c *networkRawConn) WriteRaw(packet []byte) error {
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

func (c *networkRawConn) WriteMessage(msg []byte) error {
	// Determine max message length
	packet := []byte(fmt.Sprintf("%04x%s", len(msg), msg))
	err := c.WriteRaw(packet)
	if err != nil {
		c.Close()
	}

	return err
}

func (c *networkRawConn) ReadRaw(blob []byte) error {
	_, err := io.ReadFull(c.conn, blob)
	if err != nil {
		c.Close()
		return err
	}

	return nil
}

func (c *networkRawConn) ReadStatus() (string, error) {
	resp := make([]byte, 4)
	_, err := io.ReadFull(c.conn, resp)
	if err != nil {
		c.Close()
		return "", err
	}

	return string(resp), nil
}

func (c *networkRawConn) ReadHexPrefixedBlob() ([]byte, error) {
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

func (c *networkRawConn) ReadShellBlob() (byte, []byte, error) {
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

func (c *networkRawConn) ReadLine() (*string, error) {
	scanner := bufio.NewScanner(c.conn)
	if scanner.Scan() {
		text := scanner.Text()
		return &text, nil
	}

	c.Close()
	return nil, scanner.Err()
}

func (c *networkRawConn) WriteShellBlob(id byte, blob []byte) error {
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

func (c *networkRawConn) SendCommand(cmd []byte) error {
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
