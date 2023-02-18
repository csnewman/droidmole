package syslog

import (
	"context"
	"github.com/csnewman/droidmole/agent/protocol"
	"github.com/golang/protobuf/ptypes/empty"
)

// Stream represents a stream of syslog entries.
type Stream struct {
	client protocol.AgentController_StreamSysLogClient
}

// Entry represents a syslog entry.
type Entry struct {
	// Line represents the raw line.
	Line string
}

// Open starts a new stream.
func Open(ctx context.Context, client protocol.AgentControllerClient) (*Stream, error) {
	stream, err := client.StreamSysLog(ctx, &empty.Empty{})
	if err != nil {
		return nil, err
	}

	return &Stream{
		client: stream,
	}, nil
}

// Recv blocks until a new entry is received.
func (s *Stream) Recv() (*Entry, error) {
	entry, err := s.client.Recv()
	if err != nil {
		return nil, err
	}

	return &Entry{
		Line: entry.Line,
	}, nil
}
