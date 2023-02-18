package shell

import (
	"fmt"
	"github.com/csnewman/droidmole/agent/protocol"
	"github.com/csnewman/droidmole/agent/server/adb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
)

func Process(server protocol.AgentController_OpenShellServer) error {
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
	case protocol.StartShellRequest_RAW:
		cmd += ",raw"
	case protocol.StartShellRequest_PTY:
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

	requestChan := make(chan shellRequestChanMsg)
	responseChan := make(chan shellResponseChanMsg)
	go receiveShellRequest(server, requestChan)
	go receiveShellResponse(emuConn, responseChan)

	var processingError error
outer:
	for {
		log.Println("loop")
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
				log.Println("std in request")
				// Send any data
				if msg.Stdin.Data != nil && len(msg.Stdin.Data) > 0 {
					log.Println("writing blob")
					err := emuConn.WriteShellBlob(0, msg.Stdin.Data)
					if err != nil {
						processingError = err
						break outer
					}
				}

				// Close std in if requested
				if msg.Stdin.Close {
					log.Println("stdin closing")
					err := emuConn.WriteShellBlob(4, []byte{})
					if err != nil {
						processingError = err
						break outer
					}
				}
			// Process resize request
			case *protocol.ShellRequest_Resize:
				log.Println("resize request")
				payload := fmt.Sprintf(
					"%dx%d,%dx%d",
					msg.Resize.Rows, msg.Resize.Cols,
					msg.Resize.Width, msg.Resize.Height,
				)

				log.Println("writing blob")
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
			log.Println("response")
			if rmsg.err != nil {
				processingError = rmsg.err
				break outer
			}

			switch rmsg.id {
			// Process stdout & stderr
			case 1, 2:
				log.Println("stdin/out response")
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
				log.Println("exit response")
				err := server.Send(&protocol.ShellResponse{Message: &protocol.ShellResponse_Exit{
					Exit: &protocol.ShellExitResponse{
						Code: uint32(rmsg.data[0]),
					},
				}})
				if err != nil {
					processingError = err
					break outer
				}
			default:
				processingError = status.Errorf(codes.Internal, "unknown response")
				break outer
			}
		}
	}

	if processingError != nil {
		log.Println(processingError)
	}

	// Cleanup
	close(requestChan)
	close(responseChan)

	return processingError
}

type shellRequestChanMsg struct {
	request *protocol.ShellRequest
	err     error
}

func receiveShellRequest(server protocol.AgentController_OpenShellServer, requestChan chan shellRequestChanMsg) {
	defer func() {
		// recover from panic caused by writing to a closed channel
		if r := recover(); r != nil {
			log.Println("shell request recovered", r)
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
			log.Println("shell request error", err)
			return
		}
	}
}

type shellResponseChanMsg struct {
	id   byte
	data []byte
	err  error
}

func receiveShellResponse(conn *adb.RawConnection, responseChan chan shellResponseChanMsg) {
	defer func() {
		// recover from panic caused by writing to a closed channel
		if r := recover(); r != nil {
			log.Println("shell response recovered", r)
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
			log.Println("shell response error", err)
			return
		}
	}
}
