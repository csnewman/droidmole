// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.12.4
// source: agent.proto

package protocol

import (
	context "context"
	empty "github.com/golang/protobuf/ptypes/empty"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// AgentControllerClient is the client API for AgentController service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type AgentControllerClient interface {
	// Streams the state of the agent process.
	// An initial value will be immediately produced with the current agent state. Subsequent values may indicate a change
	// in the agent state, however this is not guaranteed and the same state can be delivered multiple times.
	StreamState(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (AgentController_StreamStateClient, error)
	// Requests the emulator starts. An error will be returned if the emulator is already running.
	StartEmulator(ctx context.Context, in *StartEmulatorRequest, opts ...grpc.CallOption) (*empty.Empty, error)
	// Streams the display in the requested format.
	// An initial value will be immediately produced with the current display content. This stream can and should be
	// started before the emulator is started to ensure no frames are missed. The stream will is persistent between
	// emulator restarts.
	StreamDisplay(ctx context.Context, in *StreamDisplayRequest, opts ...grpc.CallOption) (AgentController_StreamDisplayClient, error)
	// Streams the system log (kernel messages).
	// Previous messages are not returned. This stream can and should be started before the emulator is started to ensure
	// no messages are missed. The stream will is persistent between emulator restarts.
	StreamSysLog(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (AgentController_StreamSysLogClient, error)
	SendInput(ctx context.Context, in *TouchEvent, opts ...grpc.CallOption) (*empty.Empty, error)
	// Opens an ADB shell to the emulator.
	// Requires that the emulator has reached the "running" state, otherwise an error will be returned.
	// The request stream must start with a single ShellStartRequest message.
	OpenShell(ctx context.Context, opts ...grpc.CallOption) (AgentController_OpenShellClient, error)
}

type agentControllerClient struct {
	cc grpc.ClientConnInterface
}

func NewAgentControllerClient(cc grpc.ClientConnInterface) AgentControllerClient {
	return &agentControllerClient{cc}
}

func (c *agentControllerClient) StreamState(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (AgentController_StreamStateClient, error) {
	stream, err := c.cc.NewStream(ctx, &AgentController_ServiceDesc.Streams[0], "/AgentController/streamState", opts...)
	if err != nil {
		return nil, err
	}
	x := &agentControllerStreamStateClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type AgentController_StreamStateClient interface {
	Recv() (*AgentState, error)
	grpc.ClientStream
}

type agentControllerStreamStateClient struct {
	grpc.ClientStream
}

func (x *agentControllerStreamStateClient) Recv() (*AgentState, error) {
	m := new(AgentState)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *agentControllerClient) StartEmulator(ctx context.Context, in *StartEmulatorRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	out := new(empty.Empty)
	err := c.cc.Invoke(ctx, "/AgentController/startEmulator", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *agentControllerClient) StreamDisplay(ctx context.Context, in *StreamDisplayRequest, opts ...grpc.CallOption) (AgentController_StreamDisplayClient, error) {
	stream, err := c.cc.NewStream(ctx, &AgentController_ServiceDesc.Streams[1], "/AgentController/streamDisplay", opts...)
	if err != nil {
		return nil, err
	}
	x := &agentControllerStreamDisplayClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type AgentController_StreamDisplayClient interface {
	Recv() (*DisplayFrame, error)
	grpc.ClientStream
}

type agentControllerStreamDisplayClient struct {
	grpc.ClientStream
}

func (x *agentControllerStreamDisplayClient) Recv() (*DisplayFrame, error) {
	m := new(DisplayFrame)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *agentControllerClient) StreamSysLog(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (AgentController_StreamSysLogClient, error) {
	stream, err := c.cc.NewStream(ctx, &AgentController_ServiceDesc.Streams[2], "/AgentController/streamSysLog", opts...)
	if err != nil {
		return nil, err
	}
	x := &agentControllerStreamSysLogClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type AgentController_StreamSysLogClient interface {
	Recv() (*SysLogEntry, error)
	grpc.ClientStream
}

type agentControllerStreamSysLogClient struct {
	grpc.ClientStream
}

func (x *agentControllerStreamSysLogClient) Recv() (*SysLogEntry, error) {
	m := new(SysLogEntry)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *agentControllerClient) SendInput(ctx context.Context, in *TouchEvent, opts ...grpc.CallOption) (*empty.Empty, error) {
	out := new(empty.Empty)
	err := c.cc.Invoke(ctx, "/AgentController/sendInput", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *agentControllerClient) OpenShell(ctx context.Context, opts ...grpc.CallOption) (AgentController_OpenShellClient, error) {
	stream, err := c.cc.NewStream(ctx, &AgentController_ServiceDesc.Streams[3], "/AgentController/openShell", opts...)
	if err != nil {
		return nil, err
	}
	x := &agentControllerOpenShellClient{stream}
	return x, nil
}

type AgentController_OpenShellClient interface {
	Send(*ShellRequest) error
	Recv() (*ShellResponse, error)
	grpc.ClientStream
}

type agentControllerOpenShellClient struct {
	grpc.ClientStream
}

func (x *agentControllerOpenShellClient) Send(m *ShellRequest) error {
	return x.ClientStream.SendMsg(m)
}

func (x *agentControllerOpenShellClient) Recv() (*ShellResponse, error) {
	m := new(ShellResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// AgentControllerServer is the server API for AgentController service.
// All implementations must embed UnimplementedAgentControllerServer
// for forward compatibility
type AgentControllerServer interface {
	// Streams the state of the agent process.
	// An initial value will be immediately produced with the current agent state. Subsequent values may indicate a change
	// in the agent state, however this is not guaranteed and the same state can be delivered multiple times.
	StreamState(*empty.Empty, AgentController_StreamStateServer) error
	// Requests the emulator starts. An error will be returned if the emulator is already running.
	StartEmulator(context.Context, *StartEmulatorRequest) (*empty.Empty, error)
	// Streams the display in the requested format.
	// An initial value will be immediately produced with the current display content. This stream can and should be
	// started before the emulator is started to ensure no frames are missed. The stream will is persistent between
	// emulator restarts.
	StreamDisplay(*StreamDisplayRequest, AgentController_StreamDisplayServer) error
	// Streams the system log (kernel messages).
	// Previous messages are not returned. This stream can and should be started before the emulator is started to ensure
	// no messages are missed. The stream will is persistent between emulator restarts.
	StreamSysLog(*empty.Empty, AgentController_StreamSysLogServer) error
	SendInput(context.Context, *TouchEvent) (*empty.Empty, error)
	// Opens an ADB shell to the emulator.
	// Requires that the emulator has reached the "running" state, otherwise an error will be returned.
	// The request stream must start with a single ShellStartRequest message.
	OpenShell(AgentController_OpenShellServer) error
	mustEmbedUnimplementedAgentControllerServer()
}

// UnimplementedAgentControllerServer must be embedded to have forward compatible implementations.
type UnimplementedAgentControllerServer struct {
}

func (UnimplementedAgentControllerServer) StreamState(*empty.Empty, AgentController_StreamStateServer) error {
	return status.Errorf(codes.Unimplemented, "method StreamState not implemented")
}
func (UnimplementedAgentControllerServer) StartEmulator(context.Context, *StartEmulatorRequest) (*empty.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method StartEmulator not implemented")
}
func (UnimplementedAgentControllerServer) StreamDisplay(*StreamDisplayRequest, AgentController_StreamDisplayServer) error {
	return status.Errorf(codes.Unimplemented, "method StreamDisplay not implemented")
}
func (UnimplementedAgentControllerServer) StreamSysLog(*empty.Empty, AgentController_StreamSysLogServer) error {
	return status.Errorf(codes.Unimplemented, "method StreamSysLog not implemented")
}
func (UnimplementedAgentControllerServer) SendInput(context.Context, *TouchEvent) (*empty.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SendInput not implemented")
}
func (UnimplementedAgentControllerServer) OpenShell(AgentController_OpenShellServer) error {
	return status.Errorf(codes.Unimplemented, "method OpenShell not implemented")
}
func (UnimplementedAgentControllerServer) mustEmbedUnimplementedAgentControllerServer() {}

// UnsafeAgentControllerServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to AgentControllerServer will
// result in compilation errors.
type UnsafeAgentControllerServer interface {
	mustEmbedUnimplementedAgentControllerServer()
}

func RegisterAgentControllerServer(s grpc.ServiceRegistrar, srv AgentControllerServer) {
	s.RegisterService(&AgentController_ServiceDesc, srv)
}

func _AgentController_StreamState_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(empty.Empty)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(AgentControllerServer).StreamState(m, &agentControllerStreamStateServer{stream})
}

type AgentController_StreamStateServer interface {
	Send(*AgentState) error
	grpc.ServerStream
}

type agentControllerStreamStateServer struct {
	grpc.ServerStream
}

func (x *agentControllerStreamStateServer) Send(m *AgentState) error {
	return x.ServerStream.SendMsg(m)
}

func _AgentController_StartEmulator_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StartEmulatorRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AgentControllerServer).StartEmulator(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/AgentController/startEmulator",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AgentControllerServer).StartEmulator(ctx, req.(*StartEmulatorRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _AgentController_StreamDisplay_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(StreamDisplayRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(AgentControllerServer).StreamDisplay(m, &agentControllerStreamDisplayServer{stream})
}

type AgentController_StreamDisplayServer interface {
	Send(*DisplayFrame) error
	grpc.ServerStream
}

type agentControllerStreamDisplayServer struct {
	grpc.ServerStream
}

func (x *agentControllerStreamDisplayServer) Send(m *DisplayFrame) error {
	return x.ServerStream.SendMsg(m)
}

func _AgentController_StreamSysLog_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(empty.Empty)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(AgentControllerServer).StreamSysLog(m, &agentControllerStreamSysLogServer{stream})
}

type AgentController_StreamSysLogServer interface {
	Send(*SysLogEntry) error
	grpc.ServerStream
}

type agentControllerStreamSysLogServer struct {
	grpc.ServerStream
}

func (x *agentControllerStreamSysLogServer) Send(m *SysLogEntry) error {
	return x.ServerStream.SendMsg(m)
}

func _AgentController_SendInput_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(TouchEvent)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AgentControllerServer).SendInput(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/AgentController/sendInput",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AgentControllerServer).SendInput(ctx, req.(*TouchEvent))
	}
	return interceptor(ctx, in, info, handler)
}

func _AgentController_OpenShell_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(AgentControllerServer).OpenShell(&agentControllerOpenShellServer{stream})
}

type AgentController_OpenShellServer interface {
	Send(*ShellResponse) error
	Recv() (*ShellRequest, error)
	grpc.ServerStream
}

type agentControllerOpenShellServer struct {
	grpc.ServerStream
}

func (x *agentControllerOpenShellServer) Send(m *ShellResponse) error {
	return x.ServerStream.SendMsg(m)
}

func (x *agentControllerOpenShellServer) Recv() (*ShellRequest, error) {
	m := new(ShellRequest)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// AgentController_ServiceDesc is the grpc.ServiceDesc for AgentController service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var AgentController_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "AgentController",
	HandlerType: (*AgentControllerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "startEmulator",
			Handler:    _AgentController_StartEmulator_Handler,
		},
		{
			MethodName: "sendInput",
			Handler:    _AgentController_SendInput_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "streamState",
			Handler:       _AgentController_StreamState_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "streamDisplay",
			Handler:       _AgentController_StreamDisplay_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "streamSysLog",
			Handler:       _AgentController_StreamSysLog_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "openShell",
			Handler:       _AgentController_OpenShell_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "agent.proto",
}
