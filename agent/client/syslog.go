package client

import (
	"context"
	"github.com/csnewman/droidmole/agent/protocol"
	"github.com/golang/protobuf/ptypes/empty"
)

// SysLogStream represents a stream of syslog entries.
type SysLogStream struct {
	client protocol.AgentController_StreamSysLogClient
}

// SysLogEntry represents a syslog entry.
type SysLogEntry struct {
	// Line represents the raw line.
	Line string
}

// StreamSysLog streams the system log (kernel messages).
// Previous messages are not returned. This stream can and should be started before the emulator is started to ensure no
// messages are missed. The stream will is persistent between emulator restarts.
func (c *Client) StreamSysLog(ctx context.Context) (*SysLogStream, error) {
	stream, err := c.client.StreamSysLog(ctx, &empty.Empty{})
	if err != nil {
		return nil, err
	}

	return &SysLogStream{
		client: stream,
	}, nil
}

// Recv blocks until a new entry is received.
func (s *SysLogStream) Recv() (*SysLogEntry, error) {
	entry, err := s.client.Recv()
	if err != nil {
		return nil, err
	}

	return &SysLogEntry{
		Line: entry.Line,
	}, nil
}
