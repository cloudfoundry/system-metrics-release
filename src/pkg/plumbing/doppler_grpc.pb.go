// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.19.4
// source: system-metrics-release/src/pkg/plumbing/doppler.proto

package plumbing

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// DopplerClient is the client API for Doppler service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type DopplerClient interface {
	Subscribe(ctx context.Context, in *SubscriptionRequest, opts ...grpc.CallOption) (Doppler_SubscribeClient, error)
	BatchSubscribe(ctx context.Context, in *SubscriptionRequest, opts ...grpc.CallOption) (Doppler_BatchSubscribeClient, error)
	ContainerMetrics(ctx context.Context, in *ContainerMetricsRequest, opts ...grpc.CallOption) (*ContainerMetricsResponse, error)
	RecentLogs(ctx context.Context, in *RecentLogsRequest, opts ...grpc.CallOption) (*RecentLogsResponse, error)
}

type dopplerClient struct {
	cc grpc.ClientConnInterface
}

func NewDopplerClient(cc grpc.ClientConnInterface) DopplerClient {
	return &dopplerClient{cc}
}

func (c *dopplerClient) Subscribe(ctx context.Context, in *SubscriptionRequest, opts ...grpc.CallOption) (Doppler_SubscribeClient, error) {
	stream, err := c.cc.NewStream(ctx, &Doppler_ServiceDesc.Streams[0], "/plumbing.Doppler/Subscribe", opts...)
	if err != nil {
		return nil, err
	}
	x := &dopplerSubscribeClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Doppler_SubscribeClient interface {
	Recv() (*Response, error)
	grpc.ClientStream
}

type dopplerSubscribeClient struct {
	grpc.ClientStream
}

func (x *dopplerSubscribeClient) Recv() (*Response, error) {
	m := new(Response)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *dopplerClient) BatchSubscribe(ctx context.Context, in *SubscriptionRequest, opts ...grpc.CallOption) (Doppler_BatchSubscribeClient, error) {
	stream, err := c.cc.NewStream(ctx, &Doppler_ServiceDesc.Streams[1], "/plumbing.Doppler/BatchSubscribe", opts...)
	if err != nil {
		return nil, err
	}
	x := &dopplerBatchSubscribeClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Doppler_BatchSubscribeClient interface {
	Recv() (*BatchResponse, error)
	grpc.ClientStream
}

type dopplerBatchSubscribeClient struct {
	grpc.ClientStream
}

func (x *dopplerBatchSubscribeClient) Recv() (*BatchResponse, error) {
	m := new(BatchResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *dopplerClient) ContainerMetrics(ctx context.Context, in *ContainerMetricsRequest, opts ...grpc.CallOption) (*ContainerMetricsResponse, error) {
	out := new(ContainerMetricsResponse)
	err := c.cc.Invoke(ctx, "/plumbing.Doppler/ContainerMetrics", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *dopplerClient) RecentLogs(ctx context.Context, in *RecentLogsRequest, opts ...grpc.CallOption) (*RecentLogsResponse, error) {
	out := new(RecentLogsResponse)
	err := c.cc.Invoke(ctx, "/plumbing.Doppler/RecentLogs", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// DopplerServer is the server API for Doppler service.
// All implementations must embed UnimplementedDopplerServer
// for forward compatibility
type DopplerServer interface {
	Subscribe(*SubscriptionRequest, Doppler_SubscribeServer) error
	BatchSubscribe(*SubscriptionRequest, Doppler_BatchSubscribeServer) error
	ContainerMetrics(context.Context, *ContainerMetricsRequest) (*ContainerMetricsResponse, error)
	RecentLogs(context.Context, *RecentLogsRequest) (*RecentLogsResponse, error)
	mustEmbedUnimplementedDopplerServer()
}

// UnimplementedDopplerServer must be embedded to have forward compatible implementations.
type UnimplementedDopplerServer struct {
}

func (UnimplementedDopplerServer) Subscribe(*SubscriptionRequest, Doppler_SubscribeServer) error {
	return status.Errorf(codes.Unimplemented, "method Subscribe not implemented")
}
func (UnimplementedDopplerServer) BatchSubscribe(*SubscriptionRequest, Doppler_BatchSubscribeServer) error {
	return status.Errorf(codes.Unimplemented, "method BatchSubscribe not implemented")
}
func (UnimplementedDopplerServer) ContainerMetrics(context.Context, *ContainerMetricsRequest) (*ContainerMetricsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ContainerMetrics not implemented")
}
func (UnimplementedDopplerServer) RecentLogs(context.Context, *RecentLogsRequest) (*RecentLogsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RecentLogs not implemented")
}
func (UnimplementedDopplerServer) mustEmbedUnimplementedDopplerServer() {}

// UnsafeDopplerServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to DopplerServer will
// result in compilation errors.
type UnsafeDopplerServer interface {
	mustEmbedUnimplementedDopplerServer()
}

func RegisterDopplerServer(s grpc.ServiceRegistrar, srv DopplerServer) {
	s.RegisterService(&Doppler_ServiceDesc, srv)
}

func _Doppler_Subscribe_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(SubscriptionRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(DopplerServer).Subscribe(m, &dopplerSubscribeServer{stream})
}

type Doppler_SubscribeServer interface {
	Send(*Response) error
	grpc.ServerStream
}

type dopplerSubscribeServer struct {
	grpc.ServerStream
}

func (x *dopplerSubscribeServer) Send(m *Response) error {
	return x.ServerStream.SendMsg(m)
}

func _Doppler_BatchSubscribe_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(SubscriptionRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(DopplerServer).BatchSubscribe(m, &dopplerBatchSubscribeServer{stream})
}

type Doppler_BatchSubscribeServer interface {
	Send(*BatchResponse) error
	grpc.ServerStream
}

type dopplerBatchSubscribeServer struct {
	grpc.ServerStream
}

func (x *dopplerBatchSubscribeServer) Send(m *BatchResponse) error {
	return x.ServerStream.SendMsg(m)
}

func _Doppler_ContainerMetrics_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ContainerMetricsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DopplerServer).ContainerMetrics(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/plumbing.Doppler/ContainerMetrics",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DopplerServer).ContainerMetrics(ctx, req.(*ContainerMetricsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Doppler_RecentLogs_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RecentLogsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DopplerServer).RecentLogs(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/plumbing.Doppler/RecentLogs",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DopplerServer).RecentLogs(ctx, req.(*RecentLogsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Doppler_ServiceDesc is the grpc.ServiceDesc for Doppler service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Doppler_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "plumbing.Doppler",
	HandlerType: (*DopplerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "ContainerMetrics",
			Handler:    _Doppler_ContainerMetrics_Handler,
		},
		{
			MethodName: "RecentLogs",
			Handler:    _Doppler_RecentLogs_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Subscribe",
			Handler:       _Doppler_Subscribe_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "BatchSubscribe",
			Handler:       _Doppler_BatchSubscribe_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "system-metrics-release/src/pkg/plumbing/doppler.proto",
}

// DopplerIngestorClient is the client API for DopplerIngestor service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type DopplerIngestorClient interface {
	Pusher(ctx context.Context, opts ...grpc.CallOption) (DopplerIngestor_PusherClient, error)
}

type dopplerIngestorClient struct {
	cc grpc.ClientConnInterface
}

func NewDopplerIngestorClient(cc grpc.ClientConnInterface) DopplerIngestorClient {
	return &dopplerIngestorClient{cc}
}

func (c *dopplerIngestorClient) Pusher(ctx context.Context, opts ...grpc.CallOption) (DopplerIngestor_PusherClient, error) {
	stream, err := c.cc.NewStream(ctx, &DopplerIngestor_ServiceDesc.Streams[0], "/plumbing.DopplerIngestor/Pusher", opts...)
	if err != nil {
		return nil, err
	}
	x := &dopplerIngestorPusherClient{stream}
	return x, nil
}

type DopplerIngestor_PusherClient interface {
	Send(*EnvelopeData) error
	CloseAndRecv() (*PushResponse, error)
	grpc.ClientStream
}

type dopplerIngestorPusherClient struct {
	grpc.ClientStream
}

func (x *dopplerIngestorPusherClient) Send(m *EnvelopeData) error {
	return x.ClientStream.SendMsg(m)
}

func (x *dopplerIngestorPusherClient) CloseAndRecv() (*PushResponse, error) {
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	m := new(PushResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// DopplerIngestorServer is the server API for DopplerIngestor service.
// All implementations must embed UnimplementedDopplerIngestorServer
// for forward compatibility
type DopplerIngestorServer interface {
	Pusher(DopplerIngestor_PusherServer) error
	mustEmbedUnimplementedDopplerIngestorServer()
}

// UnimplementedDopplerIngestorServer must be embedded to have forward compatible implementations.
type UnimplementedDopplerIngestorServer struct {
}

func (UnimplementedDopplerIngestorServer) Pusher(DopplerIngestor_PusherServer) error {
	return status.Errorf(codes.Unimplemented, "method Pusher not implemented")
}
func (UnimplementedDopplerIngestorServer) mustEmbedUnimplementedDopplerIngestorServer() {}

// UnsafeDopplerIngestorServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to DopplerIngestorServer will
// result in compilation errors.
type UnsafeDopplerIngestorServer interface {
	mustEmbedUnimplementedDopplerIngestorServer()
}

func RegisterDopplerIngestorServer(s grpc.ServiceRegistrar, srv DopplerIngestorServer) {
	s.RegisterService(&DopplerIngestor_ServiceDesc, srv)
}

func _DopplerIngestor_Pusher_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(DopplerIngestorServer).Pusher(&dopplerIngestorPusherServer{stream})
}

type DopplerIngestor_PusherServer interface {
	SendAndClose(*PushResponse) error
	Recv() (*EnvelopeData, error)
	grpc.ServerStream
}

type dopplerIngestorPusherServer struct {
	grpc.ServerStream
}

func (x *dopplerIngestorPusherServer) SendAndClose(m *PushResponse) error {
	return x.ServerStream.SendMsg(m)
}

func (x *dopplerIngestorPusherServer) Recv() (*EnvelopeData, error) {
	m := new(EnvelopeData)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// DopplerIngestor_ServiceDesc is the grpc.ServiceDesc for DopplerIngestor service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var DopplerIngestor_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "plumbing.DopplerIngestor",
	HandlerType: (*DopplerIngestorServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Pusher",
			Handler:       _DopplerIngestor_Pusher_Handler,
			ClientStreams: true,
		},
	},
	Metadata: "system-metrics-release/src/pkg/plumbing/doppler.proto",
}
