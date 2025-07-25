// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v3.21.12
// source: core/Discover.proto

package core

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Endpoint struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Address       []byte                 `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
	Port          int32                  `protobuf:"varint,2,opt,name=port,proto3" json:"port,omitempty"`
	NodeId        []byte                 `protobuf:"bytes,3,opt,name=nodeId,proto3" json:"nodeId,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Endpoint) Reset() {
	*x = Endpoint{}
	mi := &file_core_Discover_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Endpoint) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Endpoint) ProtoMessage() {}

func (x *Endpoint) ProtoReflect() protoreflect.Message {
	mi := &file_core_Discover_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Endpoint.ProtoReflect.Descriptor instead.
func (*Endpoint) Descriptor() ([]byte, []int) {
	return file_core_Discover_proto_rawDescGZIP(), []int{0}
}

func (x *Endpoint) GetAddress() []byte {
	if x != nil {
		return x.Address
	}
	return nil
}

func (x *Endpoint) GetPort() int32 {
	if x != nil {
		return x.Port
	}
	return 0
}

func (x *Endpoint) GetNodeId() []byte {
	if x != nil {
		return x.NodeId
	}
	return nil
}

type PingMessage struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	From          *Endpoint              `protobuf:"bytes,1,opt,name=from,proto3" json:"from,omitempty"`
	To            *Endpoint              `protobuf:"bytes,2,opt,name=to,proto3" json:"to,omitempty"`
	Version       int32                  `protobuf:"varint,3,opt,name=version,proto3" json:"version,omitempty"`
	Timestamp     int64                  `protobuf:"varint,4,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *PingMessage) Reset() {
	*x = PingMessage{}
	mi := &file_core_Discover_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *PingMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PingMessage) ProtoMessage() {}

func (x *PingMessage) ProtoReflect() protoreflect.Message {
	mi := &file_core_Discover_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PingMessage.ProtoReflect.Descriptor instead.
func (*PingMessage) Descriptor() ([]byte, []int) {
	return file_core_Discover_proto_rawDescGZIP(), []int{1}
}

func (x *PingMessage) GetFrom() *Endpoint {
	if x != nil {
		return x.From
	}
	return nil
}

func (x *PingMessage) GetTo() *Endpoint {
	if x != nil {
		return x.To
	}
	return nil
}

func (x *PingMessage) GetVersion() int32 {
	if x != nil {
		return x.Version
	}
	return 0
}

func (x *PingMessage) GetTimestamp() int64 {
	if x != nil {
		return x.Timestamp
	}
	return 0
}

type PongMessage struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	From          *Endpoint              `protobuf:"bytes,1,opt,name=from,proto3" json:"from,omitempty"`
	Echo          int32                  `protobuf:"varint,2,opt,name=echo,proto3" json:"echo,omitempty"`
	Timestamp     int64                  `protobuf:"varint,3,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *PongMessage) Reset() {
	*x = PongMessage{}
	mi := &file_core_Discover_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *PongMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PongMessage) ProtoMessage() {}

func (x *PongMessage) ProtoReflect() protoreflect.Message {
	mi := &file_core_Discover_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PongMessage.ProtoReflect.Descriptor instead.
func (*PongMessage) Descriptor() ([]byte, []int) {
	return file_core_Discover_proto_rawDescGZIP(), []int{2}
}

func (x *PongMessage) GetFrom() *Endpoint {
	if x != nil {
		return x.From
	}
	return nil
}

func (x *PongMessage) GetEcho() int32 {
	if x != nil {
		return x.Echo
	}
	return 0
}

func (x *PongMessage) GetTimestamp() int64 {
	if x != nil {
		return x.Timestamp
	}
	return 0
}

type FindNeighbours struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	From          *Endpoint              `protobuf:"bytes,1,opt,name=from,proto3" json:"from,omitempty"`
	TargetId      []byte                 `protobuf:"bytes,2,opt,name=targetId,proto3" json:"targetId,omitempty"`
	Timestamp     int64                  `protobuf:"varint,3,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *FindNeighbours) Reset() {
	*x = FindNeighbours{}
	mi := &file_core_Discover_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *FindNeighbours) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FindNeighbours) ProtoMessage() {}

func (x *FindNeighbours) ProtoReflect() protoreflect.Message {
	mi := &file_core_Discover_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FindNeighbours.ProtoReflect.Descriptor instead.
func (*FindNeighbours) Descriptor() ([]byte, []int) {
	return file_core_Discover_proto_rawDescGZIP(), []int{3}
}

func (x *FindNeighbours) GetFrom() *Endpoint {
	if x != nil {
		return x.From
	}
	return nil
}

func (x *FindNeighbours) GetTargetId() []byte {
	if x != nil {
		return x.TargetId
	}
	return nil
}

func (x *FindNeighbours) GetTimestamp() int64 {
	if x != nil {
		return x.Timestamp
	}
	return 0
}

type Neighbours struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	From          *Endpoint              `protobuf:"bytes,1,opt,name=from,proto3" json:"from,omitempty"`
	Neighbours    []*Endpoint            `protobuf:"bytes,2,rep,name=neighbours,proto3" json:"neighbours,omitempty"`
	Timestamp     int64                  `protobuf:"varint,3,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Neighbours) Reset() {
	*x = Neighbours{}
	mi := &file_core_Discover_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Neighbours) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Neighbours) ProtoMessage() {}

func (x *Neighbours) ProtoReflect() protoreflect.Message {
	mi := &file_core_Discover_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Neighbours.ProtoReflect.Descriptor instead.
func (*Neighbours) Descriptor() ([]byte, []int) {
	return file_core_Discover_proto_rawDescGZIP(), []int{4}
}

func (x *Neighbours) GetFrom() *Endpoint {
	if x != nil {
		return x.From
	}
	return nil
}

func (x *Neighbours) GetNeighbours() []*Endpoint {
	if x != nil {
		return x.Neighbours
	}
	return nil
}

func (x *Neighbours) GetTimestamp() int64 {
	if x != nil {
		return x.Timestamp
	}
	return 0
}

type BackupMessage struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Flag          bool                   `protobuf:"varint,1,opt,name=flag,proto3" json:"flag,omitempty"`
	Priority      int32                  `protobuf:"varint,2,opt,name=priority,proto3" json:"priority,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *BackupMessage) Reset() {
	*x = BackupMessage{}
	mi := &file_core_Discover_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *BackupMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BackupMessage) ProtoMessage() {}

func (x *BackupMessage) ProtoReflect() protoreflect.Message {
	mi := &file_core_Discover_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BackupMessage.ProtoReflect.Descriptor instead.
func (*BackupMessage) Descriptor() ([]byte, []int) {
	return file_core_Discover_proto_rawDescGZIP(), []int{5}
}

func (x *BackupMessage) GetFlag() bool {
	if x != nil {
		return x.Flag
	}
	return false
}

func (x *BackupMessage) GetPriority() int32 {
	if x != nil {
		return x.Priority
	}
	return 0
}

var File_core_Discover_proto protoreflect.FileDescriptor

const file_core_Discover_proto_rawDesc = "" +
	"\n" +
	"\x13core/Discover.proto\x12\bprotocol\"P\n" +
	"\bEndpoint\x12\x18\n" +
	"\aaddress\x18\x01 \x01(\fR\aaddress\x12\x12\n" +
	"\x04port\x18\x02 \x01(\x05R\x04port\x12\x16\n" +
	"\x06nodeId\x18\x03 \x01(\fR\x06nodeId\"\x91\x01\n" +
	"\vPingMessage\x12&\n" +
	"\x04from\x18\x01 \x01(\v2\x12.protocol.EndpointR\x04from\x12\"\n" +
	"\x02to\x18\x02 \x01(\v2\x12.protocol.EndpointR\x02to\x12\x18\n" +
	"\aversion\x18\x03 \x01(\x05R\aversion\x12\x1c\n" +
	"\ttimestamp\x18\x04 \x01(\x03R\ttimestamp\"g\n" +
	"\vPongMessage\x12&\n" +
	"\x04from\x18\x01 \x01(\v2\x12.protocol.EndpointR\x04from\x12\x12\n" +
	"\x04echo\x18\x02 \x01(\x05R\x04echo\x12\x1c\n" +
	"\ttimestamp\x18\x03 \x01(\x03R\ttimestamp\"r\n" +
	"\x0eFindNeighbours\x12&\n" +
	"\x04from\x18\x01 \x01(\v2\x12.protocol.EndpointR\x04from\x12\x1a\n" +
	"\btargetId\x18\x02 \x01(\fR\btargetId\x12\x1c\n" +
	"\ttimestamp\x18\x03 \x01(\x03R\ttimestamp\"\x86\x01\n" +
	"\n" +
	"Neighbours\x12&\n" +
	"\x04from\x18\x01 \x01(\v2\x12.protocol.EndpointR\x04from\x122\n" +
	"\n" +
	"neighbours\x18\x02 \x03(\v2\x12.protocol.EndpointR\n" +
	"neighbours\x12\x1c\n" +
	"\ttimestamp\x18\x03 \x01(\x03R\ttimestamp\"?\n" +
	"\rBackupMessage\x12\x12\n" +
	"\x04flag\x18\x01 \x01(\bR\x04flag\x12\x1a\n" +
	"\bpriority\x18\x02 \x01(\x05R\bpriorityBF\n" +
	"\x0forg.tron.protosB\bDiscoverZ)github.com/tronprotocol/grpc-gateway/coreb\x06proto3"

var (
	file_core_Discover_proto_rawDescOnce sync.Once
	file_core_Discover_proto_rawDescData []byte
)

func file_core_Discover_proto_rawDescGZIP() []byte {
	file_core_Discover_proto_rawDescOnce.Do(func() {
		file_core_Discover_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_core_Discover_proto_rawDesc), len(file_core_Discover_proto_rawDesc)))
	})
	return file_core_Discover_proto_rawDescData
}

var file_core_Discover_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_core_Discover_proto_goTypes = []any{
	(*Endpoint)(nil),       // 0: protocol.Endpoint
	(*PingMessage)(nil),    // 1: protocol.PingMessage
	(*PongMessage)(nil),    // 2: protocol.PongMessage
	(*FindNeighbours)(nil), // 3: protocol.FindNeighbours
	(*Neighbours)(nil),     // 4: protocol.Neighbours
	(*BackupMessage)(nil),  // 5: protocol.BackupMessage
}
var file_core_Discover_proto_depIdxs = []int32{
	0, // 0: protocol.PingMessage.from:type_name -> protocol.Endpoint
	0, // 1: protocol.PingMessage.to:type_name -> protocol.Endpoint
	0, // 2: protocol.PongMessage.from:type_name -> protocol.Endpoint
	0, // 3: protocol.FindNeighbours.from:type_name -> protocol.Endpoint
	0, // 4: protocol.Neighbours.from:type_name -> protocol.Endpoint
	0, // 5: protocol.Neighbours.neighbours:type_name -> protocol.Endpoint
	6, // [6:6] is the sub-list for method output_type
	6, // [6:6] is the sub-list for method input_type
	6, // [6:6] is the sub-list for extension type_name
	6, // [6:6] is the sub-list for extension extendee
	0, // [0:6] is the sub-list for field type_name
}

func init() { file_core_Discover_proto_init() }
func file_core_Discover_proto_init() {
	if File_core_Discover_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_core_Discover_proto_rawDesc), len(file_core_Discover_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_core_Discover_proto_goTypes,
		DependencyIndexes: file_core_Discover_proto_depIdxs,
		MessageInfos:      file_core_Discover_proto_msgTypes,
	}.Build()
	File_core_Discover_proto = out.File
	file_core_Discover_proto_goTypes = nil
	file_core_Discover_proto_depIdxs = nil
}
