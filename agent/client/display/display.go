package display

import (
	"context"
	"github.com/csnewman/droidmole/agent/protocol"
)

// Format represents the format to encode frames with.
type Format protocol.StreamDisplayRequest_FrameFormat

const (
	// RGB888 encodes 3 bytes per pixel.
	RGB888 = Format(protocol.StreamDisplayRequest_RGB888)

	// VP8 codec. Uses intermediate frames.
	VP8 = Format(protocol.StreamDisplayRequest_VP8)
)

// A Request represents the configuration the display should be streamed.
type Request struct {
	// Format specifies the frame encoding format.
	Format Format

	// MaxFPS specifies the maximum number of frames to encode per second.
	// Extra frames will be dropped, with the most recent frame encoded every 1/max_fps seconds.
	// Set to 0 to disable limit.
	MaxFPS uint32

	// KeyframeInterval specifies how often in milliseconds to encode a keyframe.
	// Set to 0 to only send when required. Not all formats use intermediate frames.
	KeyframeInterval uint32
}

// Stream represents a stream of display frames.
type Stream struct {
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

// Open starts a new stream.
func Open(ctx context.Context, client protocol.AgentControllerClient, request Request) (*Stream, error) {
	stream, err := client.StreamDisplay(ctx, &protocol.StreamDisplayRequest{
		Format:           protocol.StreamDisplayRequest_FrameFormat(request.Format),
		MaxFps:           request.MaxFPS,
		KeyframeInterval: request.KeyframeInterval,
	})
	if err != nil {
		return nil, err
	}

	return &Stream{
		client: stream,
	}, nil
}

// Recv blocks until a new frame is generated.
func (s *Stream) Recv() (*Frame, error) {
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
