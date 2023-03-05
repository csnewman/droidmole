package adb

import (
	"errors"
	"github.com/csnewman/droidmole/agent/util/testutil"
	"github.com/matryer/is"
	"io"
	"net"
	"testing"
)

func TestNetworkRawConn_WriteMessage(t *testing.T) {
	is := is.New(t)
	server, client := net.Pipe()
	conn := &networkRawConn{
		conn: client,
	}

	is.NoErr(testutil.RunParallel(
		t,
		func(t *testing.T) {
			is := is.New(t)
			is.NoErr(conn.WriteMessage([]byte("example message")))
			is.NoErr(client.Close())
		},
		func(t *testing.T) {
			is := is.New(t)
			data, err := io.ReadAll(server)
			is.NoErr(err)
			is.Equal(string(data), "000fexample message")
		},
	))
}

func TestNetworkRawConn_WriteShellBlob(t *testing.T) {
	is := is.New(t)
	server, client := net.Pipe()
	conn := &networkRawConn{
		conn: client,
	}

	is.NoErr(testutil.RunParallel(
		t,
		func(t *testing.T) {
			is := is.New(t)
			is.NoErr(conn.WriteShellBlob(123, []byte("example message")))
			is.NoErr(client.Close())
		},
		func(t *testing.T) {
			is := is.New(t)
			data, err := io.ReadAll(server)
			is.NoErr(err)
			is.Equal(data, []byte("\u007b\u000f\u0000\u0000\u0000example message"))
		},
	))
}

func TestNetworkRawConn_ReadHexPrefixedBlob(t *testing.T) {
	is := is.New(t)
	server, client := net.Pipe()
	conn := &networkRawConn{
		conn: client,
	}

	is.NoErr(testutil.RunParallel(
		t,
		func(t *testing.T) {
			is := is.New(t)
			_, err := server.Write([]byte("000fexample message"))
			is.NoErr(err)
			is.NoErr(server.Close())
		},
		func(t *testing.T) {
			is := is.New(t)
			data, err := conn.ReadHexPrefixedBlob()
			is.NoErr(err)
			is.Equal(string(data), "example message")
		},
	))
}

func TestNetworkRawConn_ReadShellBlob(t *testing.T) {
	is := is.New(t)
	server, client := net.Pipe()
	conn := &networkRawConn{
		conn: client,
	}

	is.NoErr(testutil.RunParallel(
		t,
		func(t *testing.T) {
			is := is.New(t)
			_, err := server.Write([]byte("\u007b\u000f\u0000\u0000\u0000example message"))
			is.NoErr(err)
			is.NoErr(server.Close())
		},
		func(t *testing.T) {
			is := is.New(t)
			id, data, err := conn.ReadShellBlob()
			is.NoErr(err)
			is.Equal(id, uint8(123))
			is.Equal(string(data), "example message")
		},
	))
}

func TestNetworkRawConn_ReadLine(t *testing.T) {
	is := is.New(t)
	server, client := net.Pipe()
	conn := &networkRawConn{
		conn: client,
	}

	is.NoErr(testutil.RunParallel(
		t,
		func(t *testing.T) {
			is := is.New(t)
			_, err := server.Write([]byte("Example line\n"))
			is.NoErr(err)
			_, err = server.Write([]byte("Another line\n"))
			is.NoErr(err)
			is.NoErr(server.Close())
		},
		func(t *testing.T) {
			is := is.New(t)
			data, err := conn.ReadLine()
			is.NoErr(err)
			is.True(data != nil)
			is.Equal(*data, "Example line")

			data, err = conn.ReadLine()
			is.NoErr(err)
			is.True(data != nil)
			is.Equal(*data, "Another line")
		},
	))
}

func TestNetworkRawConn_ReadStatus(t *testing.T) {
	is := is.New(t)
	server, client := net.Pipe()
	conn := &networkRawConn{
		conn: client,
	}

	is.NoErr(testutil.RunParallel(
		t,
		func(t *testing.T) {
			is := is.New(t)
			_, err := server.Write([]byte("ABCD"))
			is.NoErr(err)
			_, err = server.Write([]byte("EFGH"))
			is.NoErr(err)
			is.NoErr(server.Close())
		},
		func(t *testing.T) {
			is := is.New(t)
			data, err := conn.ReadStatus()
			is.NoErr(err)
			is.Equal(data, "ABCD")

			data, err = conn.ReadStatus()
			is.NoErr(err)
			is.Equal(data, "EFGH")
		},
	))
}

func TestNetworkRawConn_SendCommand(t *testing.T) {
	is := is.New(t)
	server, client := net.Pipe()

	is.NoErr(testutil.RunParallel(
		t,
		func(t *testing.T) {
			is := is.New(t)
			conn := &networkRawConn{
				conn: client,
			}

			is.NoErr(conn.SendCommand([]byte("example command")))
			is.Equal(conn.SendCommand([]byte("bad command")), errors.New("server error: Example Error"))
			is.NoErr(client.Close())
		},
		func(t *testing.T) {
			is := is.New(t)
			conn := &networkRawConn{
				conn: server,
			}

			// Command 1
			data, err := conn.ReadHexPrefixedBlob()
			is.NoErr(err)
			is.Equal(string(data), "example command")

			is.NoErr(conn.WriteRaw([]byte("OKAY")))

			// Command 2
			data, err = conn.ReadHexPrefixedBlob()
			is.NoErr(err)
			is.Equal(string(data), "bad command")

			is.NoErr(conn.WriteRaw([]byte("FAIL")))
			is.NoErr(conn.WriteMessage([]byte("Example Error")))
		},
	))
}
