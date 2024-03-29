package client

import (
	"context"
	"github.com/csnewman/droidmole/agent/protocol"
)

type InputEvent interface {
	ToRequest() protocol.InputRequest
}

// SendInput forward an input event to the emulator.
func (c *Client) SendInput(ctx context.Context, event InputEvent) error {
	request := event.ToRequest()
	_, err := c.client.SendInput(ctx, &request)
	return err
}

type TouchEvent struct {
	// A unique id to represent a pointer. Ids can be reused. Ids are shared amongst all connections.
	Identifier uint32

	// Coords
	X uint32
	Y uint32

	// Pointer device. A pressure of 0 must be sent to signal the event of the touch.
	Pressure   uint32
	TouchMajor int32
	TouchMinor int32
}

func (e TouchEvent) ToRequest() protocol.InputRequest {
	return protocol.InputRequest{
		Event: &protocol.InputRequest_Touch{
			Touch: &protocol.TouchEvent{
				Identifier: e.Identifier,
				X:          e.X,
				Y:          e.Y,
				Pressure:   e.Pressure,
				TouchMajor: e.TouchMajor,
				TouchMinor: e.TouchMinor,
			},
		},
	}
}
