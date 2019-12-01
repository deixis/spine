// Code generated by protoc-gen-go. DO NOT EDIT.
// source: net/pubsub/adapter/nats/natspb/nats.proto

package natspb

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
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
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type Message struct {
	Payload              []byte   `protobuf:"bytes,1,opt,name=payload,proto3" json:"payload,omitempty"`
	Transit              []byte   `protobuf:"bytes,2,opt,name=transit,proto3" json:"transit,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Message) Reset()         { *m = Message{} }
func (m *Message) String() string { return proto.CompactTextString(m) }
func (*Message) ProtoMessage()    {}
func (*Message) Descriptor() ([]byte, []int) {
	return fileDescriptor_6abd49ebdf34a7b3, []int{0}
}

func (m *Message) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Message.Unmarshal(m, b)
}
func (m *Message) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Message.Marshal(b, m, deterministic)
}
func (m *Message) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Message.Merge(m, src)
}
func (m *Message) XXX_Size() int {
	return xxx_messageInfo_Message.Size(m)
}
func (m *Message) XXX_DiscardUnknown() {
	xxx_messageInfo_Message.DiscardUnknown(m)
}

var xxx_messageInfo_Message proto.InternalMessageInfo

func (m *Message) GetPayload() []byte {
	if m != nil {
		return m.Payload
	}
	return nil
}

func (m *Message) GetTransit() []byte {
	if m != nil {
		return m.Transit
	}
	return nil
}

func init() {
	proto.RegisterType((*Message)(nil), "natspb.Message")
}

func init() {
	proto.RegisterFile("net/pubsub/adapter/nats/natspb/nats.proto", fileDescriptor_6abd49ebdf34a7b3)
}

var fileDescriptor_6abd49ebdf34a7b3 = []byte{
	// 115 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xd2, 0xcc, 0x4b, 0x2d, 0xd1,
	0x2f, 0x28, 0x4d, 0x2a, 0x2e, 0x4d, 0xd2, 0x4f, 0x4c, 0x49, 0x2c, 0x28, 0x49, 0x2d, 0xd2, 0xcf,
	0x4b, 0x2c, 0x29, 0x06, 0x13, 0x05, 0x49, 0x60, 0x4a, 0xaf, 0xa0, 0x28, 0xbf, 0x24, 0x5f, 0x88,
	0x0d, 0x22, 0xa4, 0x64, 0xcb, 0xc5, 0xee, 0x9b, 0x5a, 0x5c, 0x9c, 0x98, 0x9e, 0x2a, 0x24, 0xc1,
	0xc5, 0x5e, 0x90, 0x58, 0x99, 0x93, 0x9f, 0x98, 0x22, 0xc1, 0xa8, 0xc0, 0xa8, 0xc1, 0x13, 0x04,
	0xe3, 0x82, 0x64, 0x4a, 0x8a, 0x12, 0xf3, 0x8a, 0x33, 0x4b, 0x24, 0x98, 0x20, 0x32, 0x50, 0x6e,
	0x12, 0x1b, 0xd8, 0x34, 0x63, 0x40, 0x00, 0x00, 0x00, 0xff, 0xff, 0xb3, 0xd3, 0x30, 0x43, 0x7a,
	0x00, 0x00, 0x00,
}
