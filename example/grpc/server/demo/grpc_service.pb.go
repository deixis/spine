// Code generated by protoc-gen-go. DO NOT EDIT.
// source: example/grpc/server/demo/grpc_service.proto

package demo

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
	return fileDescriptor_37276dcd28d66420, []int{0}
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
	return fileDescriptor_37276dcd28d66420, []int{1}
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
	proto.RegisterType((*Request)(nil), "demo.Request")
	proto.RegisterType((*Response)(nil), "demo.Response")
}

func init() {
	proto.RegisterFile("example/grpc/server/demo/grpc_service.proto", fileDescriptor_37276dcd28d66420)
}

var fileDescriptor_37276dcd28d66420 = []byte{
	// 147 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xd2, 0x4e, 0xad, 0x48, 0xcc,
	0x2d, 0xc8, 0x49, 0xd5, 0x4f, 0x2f, 0x2a, 0x48, 0xd6, 0x2f, 0x4e, 0x2d, 0x2a, 0x4b, 0x2d, 0xd2,
	0x4f, 0x49, 0xcd, 0xcd, 0x07, 0x0b, 0xc4, 0x83, 0x04, 0x32, 0x93, 0x53, 0xf5, 0x0a, 0x8a, 0xf2,
	0x4b, 0xf2, 0x85, 0x58, 0x40, 0x12, 0x4a, 0xd2, 0x5c, 0xec, 0x41, 0xa9, 0x85, 0xa5, 0xa9, 0xc5,
	0x25, 0x42, 0x02, 0x5c, 0xcc, 0xb9, 0xc5, 0xe9, 0x12, 0x8c, 0x0a, 0x8c, 0x1a, 0x9c, 0x41, 0x20,
	0xa6, 0x92, 0x0c, 0x17, 0x47, 0x50, 0x6a, 0x71, 0x41, 0x7e, 0x5e, 0x71, 0x2a, 0xa6, 0xac, 0x91,
	0x01, 0x17, 0x8b, 0x4b, 0x6a, 0x6e, 0xbe, 0x90, 0x06, 0x17, 0xab, 0x47, 0x6a, 0x4e, 0x4e, 0xbe,
	0x10, 0xaf, 0x1e, 0xc8, 0x48, 0x3d, 0xa8, 0x79, 0x52, 0x7c, 0x30, 0x2e, 0xc4, 0x04, 0x25, 0x86,
	0x24, 0x36, 0xb0, 0xcd, 0xc6, 0x80, 0x00, 0x00, 0x00, 0xff, 0xff, 0xc7, 0x61, 0xb6, 0x01, 0xa8,
	0x00, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConnInterface

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion6

// DemoClient is the client API for Demo service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type DemoClient interface {
	Hello(ctx context.Context, in *Request, opts ...grpc.CallOption) (*Response, error)
}

type demoClient struct {
	cc grpc.ClientConnInterface
}

func NewDemoClient(cc grpc.ClientConnInterface) DemoClient {
	return &demoClient{cc}
}

func (c *demoClient) Hello(ctx context.Context, in *Request, opts ...grpc.CallOption) (*Response, error) {
	out := new(Response)
	err := c.cc.Invoke(ctx, "/demo.Demo/Hello", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// DemoServer is the server API for Demo service.
type DemoServer interface {
	Hello(context.Context, *Request) (*Response, error)
}

// UnimplementedDemoServer can be embedded to have forward compatible implementations.
type UnimplementedDemoServer struct {
}

func (*UnimplementedDemoServer) Hello(ctx context.Context, req *Request) (*Response, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Hello not implemented")
}

func RegisterDemoServer(s *grpc.Server, srv DemoServer) {
	s.RegisterService(&_Demo_serviceDesc, srv)
}

func _Demo_Hello_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Request)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DemoServer).Hello(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/demo.Demo/Hello",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DemoServer).Hello(ctx, req.(*Request))
	}
	return interceptor(ctx, in, info, handler)
}

var _Demo_serviceDesc = grpc.ServiceDesc{
	ServiceName: "demo.Demo",
	HandlerType: (*DemoServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Hello",
			Handler:    _Demo_Hello_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "example/grpc/server/demo/grpc_service.proto",
}
