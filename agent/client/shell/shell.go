package shell

import (
	"context"
	"errors"
	"github.com/csnewman/droidmole/agent/protocol"
)

// Type represents the shell type.
type Type protocol.ShellStartRequest_ShellType

const (
	// TypeRaw represents a raw shell.
	TypeRaw = Type(protocol.ShellStartRequest_RAW)

	// TypePTY represents a PTY shell.
	TypePTY = Type(protocol.ShellStartRequest_PTY)
)

// OutputChannel represents the output stream.
type OutputChannel protocol.ShellOutputResponse_ShellOutputChannel

const (
	// ChannelOut represents the stdout stream.
	ChannelOut = OutputChannel(protocol.ShellOutputResponse_OUT)

	// ChannelErr represents the stderr stream.
	ChannelErr = OutputChannel(protocol.ShellOutputResponse_ERR)
)

// Request represents a request to spawn a given command.
type Request struct {
	// Type signifies the shell type.
	Type Type

	// Command signifies the command line to execute.
	// Specify no command to spawn an interactive shell.
	Command *string

	// TermType signifies the "TERM=" environment value.
	TermType *string
}

// Output represents data outputted from the shell.
type Output struct {
	// Channel signifies the stream channel.
	Channel OutputChannel

	// Data signifies the stream blob.
	Data []byte
}

// Shell represents a shell over ADB.
type Shell struct {
	client   protocol.AgentController_OpenShellClient
	exited   bool
	exitCode uint8
}

// Open starts a new shell.
func Open(ctx context.Context, client protocol.AgentControllerClient, request Request) (*Shell, error) {
	shellClient, err := client.OpenShell(ctx)
	if err != nil {
		return nil, err
	}

	err = shellClient.Send(&protocol.ShellRequest{Message: &protocol.ShellRequest_Start{
		Start: &protocol.ShellStartRequest{
			ShellType: protocol.ShellStartRequest_ShellType(request.Type),
			Command:   request.Command,
			TermType:  request.TermType,
		},
	}})
	if err != nil {
		return nil, err
	}

	return &Shell{
		client: shellClient,
	}, nil
}

// SendInput feeds the blob into the stdin stream.
func (s *Shell) SendInput(data []byte) error {
	return s.client.Send(&protocol.ShellRequest{Message: &protocol.ShellRequest_Stdin{
		Stdin: &protocol.ShellStdInRequest{
			Data:  data,
			Close: false,
		},
	}})
}

// CloseInput closes the stdin stream.
func (s *Shell) CloseInput() error {
	return s.client.Send(&protocol.ShellRequest{Message: &protocol.ShellRequest_Stdin{
		Stdin: &protocol.ShellStdInRequest{
			Data:  nil,
			Close: true,
		},
	}})
}

// Resize notifies the shell that the screen has changed size.
func (s *Shell) Resize(rows uint32, cols uint32, width uint32, height uint32) error {
	return s.client.Send(&protocol.ShellRequest{Message: &protocol.ShellRequest_Resize{
		Resize: &protocol.ShellResizeRequest{
			Rows:   rows,
			Cols:   cols,
			Width:  width,
			Height: height,
		},
	}})
}

// Recv blocks until a new message is received.
// A (nil, nil) response signifies the shell has exited.
func (s *Shell) Recv() (*Output, error) {
	if s.exited {
		return nil, nil
	}

	rmsg, err := s.client.Recv()
	if err != nil {
		return nil, err
	}

	inner := rmsg.GetMessage()

	switch msg := inner.(type) {
	case *protocol.ShellResponse_Output:
		return &Output{
			Channel: OutputChannel(msg.Output.Channel),
			Data:    msg.Output.Data,
		}, nil
	case *protocol.ShellResponse_Exit:
		s.exitCode = uint8(msg.Exit.Code)
		s.exited = true
		return nil, nil
	default:
		return nil, errors.New("unknown response")
	}
}

// ExitCode returns whether the shell has closed and the exit code if so.
func (s *Shell) ExitCode() (bool, uint8) {
	return s.exited, s.exitCode
}
