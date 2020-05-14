// Code generated by protoc-gen-go. DO NOT EDIT.
// source: net/grpc/grpc_service_test.proto

package grpc_test

import (
	context "context"
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type Request struct {
	Msg                  string   `protobuf:"bytes,1,opt,name=msg,proto3" json:"msg,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Request) Reset()         { *m = Request{} }
func (m *Request) String() string { return proto.CompactTextString(m) }
func (*Request) ProtoMessage()    {}
func (*Request) Descriptor() ([]byte, []int) {
	return fileDescriptor_2aa370c2ba25d01c, []int{0}
}

func (m *Request) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Request.Unmarshal(m, b)
}
func (m *Request) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Request.Marshal(b, m, deterministic)
}
func (m *Request) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Request.Merge(m, src)
}
func (m *Request) XXX_Size() int {
	return xxx_messageInfo_Request.Size(m)
}
func (m *Request) XXX_DiscardUnknown() {
	xxx_messageInfo_Request.DiscardUnknown(m)
}

var xxx_messageInfo_Request proto.InternalMessageInfo

func (m *Request) GetMsg() string {
	if m != nil {
		return m.Msg
	}
	return ""
}

type Response struct {
	Msg                  string   `protobuf:"bytes,1,opt,name=msg,proto3" json:"msg,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Response) Reset()         { *m = Response{} }
func (m *Response) String() string { return proto.CompactTextString(m) }
func (*Response) ProtoMessage()    {}
func (*Response) Descriptor() ([]byte, []int) {
	return fileDescriptor_2aa370c2ba25d01c, []int{1}
}

func (m *Response) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Response.Unmarshal(m, b)
}
func (m *Response) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Response.Marshal(b, m, deterministic)
}
func (m *Response) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Response.Merge(m, src)
}
func (m *Response) XXX_Size() int {
	return xxx_messageInfo_Response.Size(m)
}
func (m *Response) XXX_DiscardUnknown() {
	xxx_messageInfo_Response.DiscardUnknown(m)
}

var xxx_messageInfo_Response proto.InternalMessageInfo

func (m *Response) GetMsg() string {
	if m != nil {
		return m.Msg
	}
	return ""
}

func init() {
	proto.RegisterType((*Request)(nil), "grpc_test.Request")
	proto.RegisterType((*Response)(nil), "grpc_test.Response")
}

func init() {
	proto.RegisterFile("net/grpc/grpc_service_test.proto", fileDescriptor_2aa370c2ba25d01c)
}

var fileDescriptor_2aa370c2ba25d01c = []byte{
	// 154 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x52, 0xc8, 0x4b, 0x2d, 0xd1,
	0x4f, 0x2f, 0x2a, 0x48, 0x06, 0x13, 0xf1, 0xc5, 0xa9, 0x45, 0x65, 0x99, 0xc9, 0xa9, 0xf1, 0x25,
	0xa9, 0xc5, 0x25, 0x7a, 0x05, 0x45, 0xf9, 0x25, 0xf9, 0x42, 0x9c, 0x60, 0x09, 0x90, 0x80, 0x92,
	0x34, 0x17, 0x7b, 0x50, 0x6a, 0x61, 0x69, 0x6a, 0x71, 0x89, 0x90, 0x00, 0x17, 0x73, 0x6e, 0x71,
	0xba, 0x04, 0xa3, 0x02, 0xa3, 0x06, 0x67, 0x10, 0x88, 0xa9, 0x24, 0xc3, 0xc5, 0x11, 0x94, 0x5a,
	0x5c, 0x90, 0x9f, 0x57, 0x9c, 0x8a, 0x29, 0x6b, 0x54, 0xc6, 0xc5, 0x12, 0x02, 0xd2, 0x67, 0xc4,
	0xc5, 0xea, 0x91, 0x9a, 0x93, 0x93, 0x2f, 0x24, 0xa4, 0x07, 0x37, 0x57, 0x0f, 0x6a, 0xa8, 0x94,
	0x30, 0x8a, 0x18, 0xc4, 0x2c, 0x25, 0x06, 0x21, 0x2b, 0x2e, 0x4e, 0xb0, 0x1e, 0xb7, 0x9c, 0xfc,
	0x72, 0x12, 0xf4, 0x69, 0x30, 0x1a, 0x30, 0x26, 0xb1, 0x81, 0x3d, 0x61, 0x0c, 0x08, 0x00, 0x00,
	0xff, 0xff, 0x75, 0x16, 0x53, 0xce, 0xe8, 0x00, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConnInterface

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion6

// TestClient is the client API for Test service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type TestClient interface {
	Hello(ctx context.Context, in *Request, opts ...grpc.CallOption) (*Response, error)
	HelloFlow(ctx context.Context, opts ...grpc.CallOption) (Test_HelloFlowClient, error)
}

type testClient struct {
	cc grpc.ClientConnInterface
}

func NewTestClient(cc grpc.ClientConnInterface) TestClient {
	return &testClient{cc}
}

func (c *testClient) Hello(ctx context.Context, in *Request, opts ...grpc.CallOption) (*Response, error) {
	out := new(Response)
	err := c.cc.Invoke(ctx, "/grpc_test.Test/Hello", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *testClient) HelloFlow(ctx context.Context, opts ...grpc.CallOption) (Test_HelloFlowClient, error) {
	stream, err := c.cc.NewStream(ctx, &_Test_serviceDesc.Streams[0], "/grpc_test.Test/HelloFlow", opts...)
	if err != nil {
		return nil, err
	}
	x := &testHelloFlowClient{stream}
	return x, nil
}

type Test_HelloFlowClient interface {
	Send(*Request) error
	Recv() (*Response, error)
	grpc.ClientStream
}

type testHelloFlowClient struct {
	grpc.ClientStream
}

func (x *testHelloFlowClient) Send(m *Request) error {
	return x.ClientStream.SendMsg(m)
}

func (x *testHelloFlowClient) Recv() (*Response, error) {
	m := new(Response)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// TestServer is the server API for Test service.
type TestServer interface {
	Hello(context.Context, *Request) (*Response, error)
	HelloFlow(Test_HelloFlowServer) error
}

// UnimplementedTestServer can be embedded to have forward compatible implementations.
type UnimplementedTestServer struct {
}

func (*UnimplementedTestServer) Hello(ctx context.Context, req *Request) (*Response, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Hello not implemented")
}
func (*UnimplementedTestServer) HelloFlow(srv Test_HelloFlowServer) error {
	return status.Errorf(codes.Unimplemented, "method HelloFlow not implemented")
}

func RegisterTestServer(s *grpc.Server, srv TestServer) {
	s.RegisterService(&_Test_serviceDesc, srv)
}

func _Test_Hello_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Request)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TestServer).Hello(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/grpc_test.Test/Hello",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TestServer).Hello(ctx, req.(*Request))
	}
	return interceptor(ctx, in, info, handler)
}

func _Test_HelloFlow_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(TestServer).HelloFlow(&testHelloFlowServer{stream})
}

type Test_HelloFlowServer interface {
	Send(*Response) error
	Recv() (*Request, error)
	grpc.ServerStream
}

type testHelloFlowServer struct {
	grpc.ServerStream
}

func (x *testHelloFlowServer) Send(m *Response) error {
	return x.ServerStream.SendMsg(m)
}

func (x *testHelloFlowServer) Recv() (*Request, error) {
	m := new(Request)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

var _Test_serviceDesc = grpc.ServiceDesc{
	ServiceName: "grpc_test.Test",
	HandlerType: (*TestServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Hello",
			Handler:    _Test_Hello_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "HelloFlow",
			Handler:       _Test_HelloFlow_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "net/grpc/grpc_service_test.proto",
}
