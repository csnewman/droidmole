package shell

import (
	"fmt"
	"github.com/csnewman/droidmole/agent/protocol"
	"github.com/csnewman/droidmole/agent/server/adb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"log"
	"sync/atomic"
)

func Process(adb adb.Adb, server protocol.AgentController_OpenShellServer) error {
	initMsg, err := server.Recv()
	if err != nil {
		return err
	}

	startMsg := initMsg.GetStart()
	if startMsg == nil {
		return status.Errorf(codes.InvalidArgument, "stream must begin with a start request")
	}

	// Build adb command
	cmd := "shell,v2"
	if startMsg.TermType != nil {
		cmd += ",TERM=" + *startMsg.TermType
	}

	switch startMsg.ShellType {
	case protocol.ShellStartRequest_RAW:
		cmd += ",raw"
	case protocol.ShellStartRequest_PTY:
		cmd += ",pty"
	default:
		return status.Errorf(codes.InvalidArgument, "unknown shell type")
	}

	cmd += ":"
	if startMsg.Command != nil {
		cmd += *startMsg.Command
	}

	// Connect and send command
	emuConn, err := adb.OpenEmulator()
	if err != nil {
		return err
	}

	err = emuConn.SendCommand([]byte(cmd))
	if err != nil {
		return err
	}

	exited := &atomic.Bool{}
	requestChan := make(chan shellRequestChanMsg)
	responseChan := make(chan shellResponseChanMsg)
	go receiveShellRequest(server, requestChan, exited)
	go receiveShellResponse(emuConn, responseChan, exited)

	var processingError error
outer:
	for {
		select {
		case rmsg := <-requestChan:
			if rmsg.err != nil {
				processingError = rmsg.err
				break outer
			}

			inner := rmsg.request.GetMessage()

			switch msg := inner.(type) {
			// Process stdin data
			case *protocol.ShellRequest_Stdin:
				// Send any data
				if msg.Stdin.Data != nil && len(msg.Stdin.Data) > 0 {
					err := emuConn.WriteShellBlob(0, msg.Stdin.Data)
					if err != nil {
						processingError = err
						break outer
					}
				}

				// Close std in if requested
				if msg.Stdin.Close {
					err := emuConn.WriteShellBlob(4, []byte{})
					if err != nil {
						processingError = err
						break outer
					}
				}
			// Process resize request
			case *protocol.ShellRequest_Resize:
				payload := fmt.Sprintf(
					"%dx%d,%dx%d",
					msg.Resize.Rows, msg.Resize.Cols,
					msg.Resize.Width, msg.Resize.Height,
				)

				err := emuConn.WriteShellBlob(5, []byte(payload))
				if err != nil {
					processingError = err
					break outer
				}
			default:
				processingError = status.Errorf(codes.InvalidArgument, "unknown request")
				break outer
			}
		case rmsg := <-responseChan:
			if rmsg.err != nil {
				processingError = rmsg.err
				break outer
			}

			switch rmsg.id {
			// Process stdout & stderr
			case 1, 2:
				channel := protocol.ShellOutputResponse_OUT
				if rmsg.id == 2 {
					channel = protocol.ShellOutputResponse_ERR
				}

				err := server.Send(&protocol.ShellResponse{Message: &protocol.ShellResponse_Output{
					Output: &protocol.ShellOutputResponse{
						Channel: channel,
						Data:    rmsg.data,
					},
				}})
				if err != nil {
					processingError = err
					break outer
				}
			// Process exit notification
			case 3:
				exited.Store(true)
				processingError = server.Send(&protocol.ShellResponse{Message: &protocol.ShellResponse_Exit{
					Exit: &protocol.ShellExitResponse{
						Code: uint32(rmsg.data[0]),
					},
				}})
				break outer
			default:
				processingError = status.Errorf(codes.Internal, "unknown response")
				break outer
			}
		}
	}

	if processingError != nil {
		log.Println("error while processing shell", processingError)
	}

	// Cleanup
	close(requestChan)
	close(responseChan)

	err = emuConn.Close()
	if err != nil {
		log.Println("error while closing adb connection", err)
	}

	return processingError
}

type shellRequestChanMsg struct {
	request *protocol.ShellRequest
	err     error
}

func receiveShellRequest(
	server protocol.AgentController_OpenShellServer,
	requestChan chan shellRequestChanMsg,
	exited *atomic.Bool,
) {
	defer func() {
		// recover from panic caused by writing to a closed channel
		if r := recover(); r != nil {
			if !exited.Load() {
				log.Println("shell request recovered", r)
			}
			return
		}
	}()

	for {
		msg, err := server.Recv()
		requestChan <- shellRequestChanMsg{
			request: msg,
			err:     err,
		}

		if err != nil {
			if err != io.EOF {
				log.Println("shell request error", err)
			}
			return
		}
	}
}

type shellResponseChanMsg struct {
	id   byte
	data []byte
	err  error
}

func receiveShellResponse(
	conn *adb.RawConnection,
	responseChan chan shellResponseChanMsg,
	exited *atomic.Bool,
) {
	defer func() {
		// recover from panic caused by writing to a closed channel
		if r := recover(); r != nil {
			if !exited.Load() {
				log.Println("shell response recovered", r)
			}
			return
		}
	}()

	for {
		id, data, err := conn.ReadShellBlob()
		responseChan <- shellResponseChanMsg{
			id:   id,
			data: data,
			err:  err,
		}

		if err != nil {
			if err != io.EOF {
				log.Println("shell response error", err)
			}
			return
		}
	}
}
