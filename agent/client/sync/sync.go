package sync

import (
	"context"
	"errors"
	"fmt"
	"github.com/csnewman/droidmole/agent/protocol"
)

// FileStat represent a file stats
type FileStat struct {
	Dev   uint64
	INo   uint64
	Mode  uint32
	NLink uint32
	UId   uint32
	GId   uint32
	Size  uint64
	ATime int64
	MTime int64
	CTime int64
}

func convertFileStat(stat protocol.FileStat) FileStat {
	return FileStat{
		Dev:   stat.Dev,
		INo:   stat.Ino,
		Mode:  stat.Mode,
		NLink: stat.Nlink,
		UId:   stat.Uid,
		GId:   stat.Gid,
		Size:  stat.Size,
		ATime: stat.Atime,
		MTime: stat.Mtime,
		CTime: stat.Ctime,
	}
}

// DirectoryEntry represents a single entry in a directory.
type DirectoryEntry struct {
	// Name stores the entry name.
	Name string
	// StatError stores the error code from attempting to run stat on the entry.
	StatError uint32
	// Stat stores the file stat result if successful. StatError should be compared against 0 first.
	Stat *FileStat
}

// ListDirectory list all files in a directory.
func ListDirectory(ctx context.Context, client protocol.AgentControllerClient, path string) ([]DirectoryEntry, error) {
	resp, err := client.ListDirectory(ctx, &protocol.ListDirectoryRequest{Path: path})
	if err != nil {
		return nil, err
	}

	var result []DirectoryEntry

	for _, entry := range resp.Entries {
		var newEntry = DirectoryEntry{
			Name:      entry.Name,
			StatError: 0,
			Stat:      nil,
		}

		switch msg := (entry.Stat).(type) {
		case *protocol.ListDirectoryEntry_StatError:
			newEntry.StatError = msg.StatError
		case *protocol.ListDirectoryEntry_StatValue:
			value := convertFileStat(*msg.StatValue)
			newEntry.Stat = &value
		default:
			return nil, errors.New("unknown response")
		}

		result = append(result, newEntry)
	}

	return result, nil
}

// StatFile stats a given path, optionally following links.
func StatFile(ctx context.Context, client protocol.AgentControllerClient, path string, followLinks bool) (*FileStat, error) {
	resp, err := client.StatFile(ctx, &protocol.StatFileRequest{
		Path:        path,
		FollowLinks: followLinks,
	})
	if err != nil {
		return nil, err
	}

	switch msg := (resp.Stat).(type) {
	case *protocol.StatFileResponse_StatError:
		return nil, fmt.Errorf("stat failed: %d", msg.StatError)
	case *protocol.StatFileResponse_StatValue:
		value := convertFileStat(*msg.StatValue)
		return &value, nil
	default:
		return nil, errors.New("unknown response")
	}
}

// PullStream represents a download stream.
type PullStream struct {
	server   protocol.AgentController_PullFileClient
	complete bool
}

// PullFile starts a file transfer for the given path.
func PullFile(ctx context.Context, client protocol.AgentControllerClient, path string) (*PullStream, error) {
	server, err := client.PullFile(ctx, &protocol.PullFileRequest{
		Path: path,
	})
	if err != nil {
		return nil, err
	}

	return &PullStream{server: server}, nil
}

// Recv blocks until a new fragment is received.
// A (nil, nil) response signifies the stream has completed.
func (s *PullStream) Recv() ([]byte, error) {
	if s.complete {
		return nil, nil
	}

	resp, err := s.server.Recv()
	if err != nil {
		return nil, err
	}

	if resp.Last {
		s.complete = true
	}

	return resp.Data, nil
}

// Complete returns whether the entire file has been returned.
func (s *PullStream) Complete() bool {
	return s.complete
}

// PushStream represents a upload stream.
type PushStream struct {
	server   protocol.AgentController_PushFileClient
	complete bool
}

// PushFile starts a file transfer for the given path.
func PushFile(ctx context.Context, client protocol.AgentControllerClient, path string, mode uint32) (*PushStream, error) {
	server, err := client.PushFile(ctx)
	if err != nil {
		return nil, err
	}

	err = server.Send(&protocol.PushFileRequest{
		Message: &protocol.PushFileRequest_Start{
			Start: &protocol.PushFileStartRequest{
				Path: path,
				Mode: mode,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return &PushStream{server: server}, nil
}

// Send sends a block.
func (s *PushStream) Send(blob []byte) error {
	return s.server.Send(&protocol.PushFileRequest{
		Message: &protocol.PushFileRequest_Data{
			Data: &protocol.PushFileDataRequest{
				Data: blob,
			},
		},
	})
}

// End marks the end of the transfer.
func (s *PushStream) End(mtime uint32) error {
	err := s.server.Send(&protocol.PushFileRequest{
		Message: &protocol.PushFileRequest_End{
			End: &protocol.PushFileEndRequest{
				Mtime: mtime,
			},
		},
	})
	s.server.CloseSend()
	return err
}
