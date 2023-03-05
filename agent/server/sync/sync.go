package sync

import (
	"github.com/csnewman/droidmole/agent/protocol"
	"github.com/csnewman/droidmole/agent/server/adb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func convertFileStat(stat adb.FileStat) *protocol.FileStat {
	return &protocol.FileStat{
		Dev:   stat.Dev,
		Ino:   stat.INo,
		Mode:  stat.Mode,
		Nlink: stat.NLink,
		Uid:   stat.UId,
		Gid:   stat.GId,
		Size:  stat.Size,
		Atime: stat.ATime,
		Mtime: stat.MTime,
		Ctime: stat.CTime,
	}
}

func ListDirectory(adb adb.Adb, request protocol.ListDirectoryRequest) (*protocol.ListDirectoryResponse, error) {
	entries, err := adb.ListDirectory(request.Path)
	if err != nil {
		return nil, err
	}

	var result []*protocol.ListDirectoryEntry

	for _, entry := range entries {
		if entry.StatError != 0 {
			result = append(result, &protocol.ListDirectoryEntry{
				Name: entry.Name,
				Stat: &protocol.ListDirectoryEntry_StatError{
					StatError: entry.StatError,
				},
			})
		} else {
			result = append(result, &protocol.ListDirectoryEntry{
				Name: entry.Name,
				Stat: &protocol.ListDirectoryEntry_StatValue{
					StatValue: convertFileStat(entry.Stat),
				},
			})
		}
	}

	return &protocol.ListDirectoryResponse{
		Entries: result,
	}, nil

}

func StatFile(adb adb.Adb, request protocol.StatFileRequest) (*protocol.StatFileResponse, error) {
	statError, stat, err := adb.StatFile(request.Path, request.FollowLinks)
	if err != nil {
		return nil, err
	}

	if statError != 0 {
		return &protocol.StatFileResponse{
			Stat: &protocol.StatFileResponse_StatError{
				StatError: statError,
			},
		}, nil
	} else {
		return &protocol.StatFileResponse{
			Stat: &protocol.StatFileResponse_StatValue{
				StatValue: convertFileStat(*stat),
			},
		}, nil
	}
}

func PullFile(adb adb.Adb, request protocol.PullFileRequest, server protocol.AgentController_PullFileServer) error {
	stream, err := adb.PullFile(request.Path)
	if err != nil {
		return err
	}

	defer stream.Close()

	for {
		data, err := stream.Recv()
		if err != nil {
			return err
		}

		// EOF
		if data == nil {
			break
		}

		err = server.Send(&protocol.PullFileResponse{
			Data: data,
			Last: false,
		})
		if err != nil {
			return err
		}
	}

	return server.Send(&protocol.PullFileResponse{
		Data: nil,
		Last: true,
	})
}

func PushFile(adb adb.Adb, server protocol.AgentController_PushFileServer) error {
	initMsg, err := server.Recv()
	if err != nil {
		return err
	}

	startMsg := initMsg.GetStart()
	if startMsg == nil {
		return status.Errorf(codes.InvalidArgument, "stream must begin with a start request")
	}

	stream, err := adb.PushFile(startMsg.Path, startMsg.Mode)
	if err != nil {
		return err
	}

	defer stream.Close()

	for {
		rmsg, err := server.Recv()
		if err != nil {
			return err
		}

		switch msg := (rmsg.Message).(type) {
		case *protocol.PushFileRequest_Data:
			err = stream.Send(msg.Data.Data)
			if err != nil {
				return err
			}
		case *protocol.PushFileRequest_End:
			err = stream.Done(msg.End.Mtime)
			if err != nil {
				return err
			}

			return nil
		default:
			return status.Errorf(codes.InvalidArgument, "unknown request")
		}
	}
}
