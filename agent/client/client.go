package client

import (
	"context"
	"github.com/csnewman/droidmole/agent/client/shell"
	"github.com/csnewman/droidmole/agent/client/sync"
	"github.com/csnewman/droidmole/agent/protocol"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client represents a connection to a agent.
type Client struct {
	conn   *grpc.ClientConn
	client protocol.AgentControllerClient
}

// Connect opens a new connection to the given address.
func Connect(addr string) (*Client, error) {
	var opts []grpc.DialOption
	// TODO: Implement secure connections
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	opts = append(opts, grpc.WithBlock())

	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		return nil, err
	}

	client := protocol.NewAgentControllerClient(conn)

	return &Client{
		conn:   conn,
		client: client,
	}, nil
}

// StartEmulatorRequest represents a request to boot the emulator with the given configuration.
//
// Example Settings:
// Ram: 2048 Cores: 1
// Display 720x1280 320dpi
type StartEmulatorRequest struct {
	// RamSize signifies memory in MBs.
	RamSize uint32

	// CoreCount signifies the number of cores.
	CoreCount uint32

	// LcdDensity signifies the DPI of the main display.
	LcdDensity uint32

	// LcdWidth signifies the width of the main display.
	LcdWidth uint32

	// LcdHeight signifies the height of the main display.
	LcdHeight uint32

	// RootADB requests the ADB connection is rooted. This does not guarantee the connection is rooted.
	RootADB bool
}

// StartEmulator requests the emulator starts. An error will be returned if the emulator is already running.
func (c *Client) StartEmulator(ctx context.Context, request StartEmulatorRequest) error {
	_, err := c.client.StartEmulator(ctx, &protocol.StartEmulatorRequest{
		RamSize:    request.RamSize,
		CoreCount:  request.CoreCount,
		LcdDensity: request.LcdDensity,
		LcdWidth:   request.LcdWidth,
		LcdHeight:  request.LcdHeight,
		RootAdb:    request.RootADB,
	})
	return err
}

// StopEmulator requests the emulator exists.
func (c *Client) StopEmulator(ctx context.Context, forceExit bool) error {
	_, err := c.client.StopEmulator(ctx, &protocol.StopEmulatorRequest{
		ForceExit: forceExit,
	})
	return err
}

// OpenShell opens an ADB shell to the emulator.
// Requires that the emulator has reached the "running" state, otherwise an error will be returned.
func (c *Client) OpenShell(ctx context.Context, request shell.Request) (*shell.Shell, error) {
	return shell.Open(ctx, c.client, request)
}

// ListDirectory list all files in a directory.
// Requires that the emulator has reached the "running" state, otherwise an error will be returned.
func (c *Client) ListDirectory(ctx context.Context, path string) ([]sync.DirectoryEntry, error) {
	return sync.ListDirectory(ctx, c.client, path)
}

// StatFile stats a given path, optionally following links.
// Requires that the emulator has reached the "running" state, otherwise an error will be returned.
func (c *Client) StatFile(ctx context.Context, path string, followLinks bool) (*sync.FileStat, error) {
	return sync.StatFile(ctx, c.client, path, followLinks)
}

// PullFile starts a file download transfer for the given path.
// Requires that the emulator has reached the "running" state, otherwise an error will be returned.
func (c *Client) PullFile(ctx context.Context, path string) (*sync.PullStream, error) {
	return sync.PullFile(ctx, c.client, path)
}

// PushFile starts a file upload transfer for the given path.
// Requires that the emulator has reached the "running" state, otherwise an error will be returned.
func (c *Client) PushFile(ctx context.Context, path string, mode uint32) (*sync.PushStream, error) {
	return sync.PushFile(ctx, c.client, path, mode)
}

func (c *Client) Close() error {
	// TODO: Close all streams
	return c.conn.Close()
}
