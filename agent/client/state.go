package client

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

// StateStream represents a stream of agent states.
type StateStream struct {
	client protocol.AgentController_StreamStateClient
}

// State represents the current state of the agent process and the virtual machine being controlled.
type State struct {
	// EmulatorState stores the state of the emulator.
	EmulatorState EmulatorState

	// EmulatorError stores the error message associated with the error state.
	EmulatorError *string
}

// StreamState streams the state of the agent process.
// An initial value will be immediately produced with the current agent state. Subsequent values may indicate a change
// in the agent state, however this is not guaranteed and the same state can be delivered multiple times.
func (c *Client) StreamState(ctx context.Context) (*StateStream, error) {
	stream, err := c.client.StreamState(ctx, &empty.Empty{})
	if err != nil {
		return nil, err
	}

	return &StateStream{
		client: stream,
	}, nil
}

// Recv blocks until a new state is received.
func (s *StateStream) Recv() (*State, error) {
	state, err := s.client.Recv()
	if err != nil {
		return nil, err
	}

	return &State{
		EmulatorState: EmulatorState(state.EmulatorState),
		EmulatorError: state.EmulatorError,
	}, nil
}
