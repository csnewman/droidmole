package client

import (
	"context"
	"github.com/csnewman/droidmole/agent/protocol"
)

// FrameFormat represents the format to encode frames with.
type FrameFormat protocol.StreamDisplayRequest_FrameFormat

const (
	// RGB888 encodes 3 bytes per pixel.
	RGB888 = FrameFormat(protocol.StreamDisplayRequest_RGB888)

	// VP8 codec. Uses intermediate frames.
	VP8 = FrameFormat(protocol.StreamDisplayRequest_VP8)
)

// A DisplayRequest represents the configuration the display should be streamed.
type DisplayRequest struct {
	// Format specifies the frame encoding format.
	Format FrameFormat

	// MaxFPS specifies the maximum number of frames to encode per second.
	// Extra frames will be dropped, with the most recent frame encoded every 1/max_fps seconds.
	// Set to 0 to disable limit.
	MaxFPS uint32

	// KeyframeInterval specifies how often in milliseconds to encode a keyframe.
	// Set to 0 to only send when required. Not all formats use intermediate frames.
	KeyframeInterval uint32
}

// DisplayStream represents a stream of display frames.
type DisplayStream struct {
	client protocol.AgentController_StreamDisplayClient
}

// Frame represents a single display frame.
type Frame struct {
	// Keyframe specifies whether this is a key frame. For some formats, this will always be true.
	Keyframe bool
	// Width specifies the width of the frame.
	Width uint32
	// Height specifies the height of the frame.
	Height uint32
	// Data contains the raw frame data.
	Data []byte
}

// StreamDisplay streams the display in the requested format.
// An initial value will be immediately produced with the current display content. This stream can and should be started
// before the emulator is started to ensure no frames are missed. The stream will is persistent between emulator
// restarts.
func (c *Client) StreamDisplay(ctx context.Context, request DisplayRequest) (*DisplayStream, error) {
	stream, err := c.client.StreamDisplay(ctx, &protocol.StreamDisplayRequest{
		Format:           protocol.StreamDisplayRequest_FrameFormat(request.Format),
		MaxFps:           request.MaxFPS,
		KeyframeInterval: request.KeyframeInterval,
	})
	if err != nil {
		return nil, err
	}

	return &DisplayStream{
		client: stream,
	}, nil
}

// Recv blocks until a new frame is generated.
func (s *DisplayStream) Recv() (*Frame, error) {
	frame, err := s.client.Recv()
	if err != nil {
		return nil, err
	}

	return &Frame{
		Keyframe: frame.Keyframe,
		Width:    frame.Width,
		Height:   frame.Height,
		Data:     frame.Data,
	}, nil
}
