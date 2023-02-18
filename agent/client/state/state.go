package state

import (
	"context"
	"github.com/csnewman/droidmole/agent/protocol"
	"github.com/golang/protobuf/ptypes/empty"
)

// EmulatorState represents the state of android emulator.
type EmulatorState protocol.AgentState_EmulatorState

const (
	// EmulatorError signifies the emulator failed to start.
	EmulatorError = EmulatorState(protocol.AgentState_ERROR)

	// EmulatorOff signifies the emulator is off.
	EmulatorOff = EmulatorState(protocol.AgentState_OFF)

	// EmulatorStarting signifies the emulator is booting.
	EmulatorStarting = EmulatorState(protocol.AgentState_STARTING)

	// EmulatorRunning signifies the emulator is running and adb has connected.
	EmulatorRunning = EmulatorState(protocol.AgentState_RUNNING)
)

// Stream represents a stream of agent states.
type Stream struct {
	client protocol.AgentController_StreamStateClient
}

// State represents the current state of the agent process and the virtual machine being controlled.
type State struct {
	EmulatorState EmulatorState
}

// Open starts a new stream.
func Open(ctx context.Context, client protocol.AgentControllerClient) (*Stream, error) {
	stream, err := client.StreamState(ctx, &empty.Empty{})
	if err != nil {
		return nil, err
	}

	return &Stream{
		client: stream,
	}, nil
}

// Recv blocks until a new state is received.
func (s *Stream) Recv() (*State, error) {
	state, err := s.client.Recv()
	if err != nil {
		return nil, err
	}

	return &State{
		EmulatorState: EmulatorState(state.EmulatorState),
	}, nil
}
