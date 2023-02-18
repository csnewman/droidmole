package client

import (
	"context"
	"github.com/csnewman/droidmole/agent/client/display"
	"github.com/csnewman/droidmole/agent/client/state"
	"github.com/csnewman/droidmole/agent/client/syslog"
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

// StreamState streams the state of the agent process.
// An initial value will be immediately produced with the current agent state. Subsequent values may indicate a change
// in the agent state, however this is not guaranteed and the same state can be delivered multiple times.
func (c *Client) StreamState(ctx context.Context) (*state.Stream, error) {
	return state.Open(ctx, c.client)
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
}

// StartEmulator requests the emulator starts. An error will be returned if the emulator is already running.
func (c *Client) StartEmulator(ctx context.Context, request StartEmulatorRequest) error {
	_, err := c.client.StartEmulator(ctx, &protocol.StartEmulatorRequest{
		RamSize:    request.RamSize,
		CoreCount:  request.CoreCount,
		LcdDensity: request.LcdDensity,
		LcdWidth:   request.LcdWidth,
		LcdHeight:  request.LcdHeight,
	})
	return err
}

// StreamDisplay streams the display in the requested format.
// An initial value will be immediately produced with the current display content. This stream can and should be started
// before the emulator is started to ensure no frames are missed. The stream will is persistent between emulator
// restarts.
func (c *Client) StreamDisplay(ctx context.Context, request display.Request) (*display.Stream, error) {
	return display.Open(ctx, c.client, request)
}

// StreamSysLog streams the system log (kernel messages).
// Previous messages are not returned. This stream can and should be started before the emulator is started to ensure no
// messages are missed. The stream will is persistent between emulator restarts.
func (c *Client) StreamSysLog(ctx context.Context) (*syslog.Stream, error) {
	return syslog.Open(ctx, c.client)
}

func (c *Client) Close() error {
	// TODO: Close all streams
	return c.conn.Close()
}
