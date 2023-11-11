// SPDX-License-Identifier: MIT

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.30.0
// 	protoc        v4.23.4
// source: ipc-msg.proto

package ipc_msg

import (
	ipc_desc "github.com/checkpoint-restore/go-criu/v7/crit/images/ipc-desc"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type IpcMsg struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Mtype *uint64 `protobuf:"varint,1,req,name=mtype" json:"mtype,omitempty"`
	Msize *uint32 `protobuf:"varint,2,req,name=msize" json:"msize,omitempty"`
}

func (x *IpcMsg) Reset() {
	*x = IpcMsg{}
	if protoimpl.UnsafeEnabled {
		mi := &file_ipc_msg_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *IpcMsg) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*IpcMsg) ProtoMessage() {}

func (x *IpcMsg) ProtoReflect() protoreflect.Message {
	mi := &file_ipc_msg_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use IpcMsg.ProtoReflect.Descriptor instead.
func (*IpcMsg) Descriptor() ([]byte, []int) {
	return file_ipc_msg_proto_rawDescGZIP(), []int{0}
}

func (x *IpcMsg) GetMtype() uint64 {
	if x != nil && x.Mtype != nil {
		return *x.Mtype
	}
	return 0
}

func (x *IpcMsg) GetMsize() uint32 {
	if x != nil && x.Msize != nil {
		return *x.Msize
	}
	return 0
}

type IpcMsgEntry struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Desc   *ipc_desc.IpcDescEntry `protobuf:"bytes,1,req,name=desc" json:"desc,omitempty"`
	Qbytes *uint32                `protobuf:"varint,2,req,name=qbytes" json:"qbytes,omitempty"`
	Qnum   *uint32                `protobuf:"varint,3,req,name=qnum" json:"qnum,omitempty"`
}

func (x *IpcMsgEntry) Reset() {
	*x = IpcMsgEntry{}
	if protoimpl.UnsafeEnabled {
		mi := &file_ipc_msg_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *IpcMsgEntry) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*IpcMsgEntry) ProtoMessage() {}

func (x *IpcMsgEntry) ProtoReflect() protoreflect.Message {
	mi := &file_ipc_msg_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use IpcMsgEntry.ProtoReflect.Descriptor instead.
func (*IpcMsgEntry) Descriptor() ([]byte, []int) {
	return file_ipc_msg_proto_rawDescGZIP(), []int{1}
}

func (x *IpcMsgEntry) GetDesc() *ipc_desc.IpcDescEntry {
	if x != nil {
		return x.Desc
	}
	return nil
}

func (x *IpcMsgEntry) GetQbytes() uint32 {
	if x != nil && x.Qbytes != nil {
		return *x.Qbytes
	}
	return 0
}

func (x *IpcMsgEntry) GetQnum() uint32 {
	if x != nil && x.Qnum != nil {
		return *x.Qnum
	}
	return 0
}

var File_ipc_msg_proto protoreflect.FileDescriptor

var file_ipc_msg_proto_rawDesc = []byte{
	0x0a, 0x0d, 0x69, 0x70, 0x63, 0x2d, 0x6d, 0x73, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a,
	0x0e, 0x69, 0x70, 0x63, 0x2d, 0x64, 0x65, 0x73, 0x63, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22,
	0x35, 0x0a, 0x07, 0x69, 0x70, 0x63, 0x5f, 0x6d, 0x73, 0x67, 0x12, 0x14, 0x0a, 0x05, 0x6d, 0x74,
	0x79, 0x70, 0x65, 0x18, 0x01, 0x20, 0x02, 0x28, 0x04, 0x52, 0x05, 0x6d, 0x74, 0x79, 0x70, 0x65,
	0x12, 0x14, 0x0a, 0x05, 0x6d, 0x73, 0x69, 0x7a, 0x65, 0x18, 0x02, 0x20, 0x02, 0x28, 0x0d, 0x52,
	0x05, 0x6d, 0x73, 0x69, 0x7a, 0x65, 0x22, 0x60, 0x0a, 0x0d, 0x69, 0x70, 0x63, 0x5f, 0x6d, 0x73,
	0x67, 0x5f, 0x65, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x23, 0x0a, 0x04, 0x64, 0x65, 0x73, 0x63, 0x18,
	0x01, 0x20, 0x02, 0x28, 0x0b, 0x32, 0x0f, 0x2e, 0x69, 0x70, 0x63, 0x5f, 0x64, 0x65, 0x73, 0x63,
	0x5f, 0x65, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x04, 0x64, 0x65, 0x73, 0x63, 0x12, 0x16, 0x0a, 0x06,
	0x71, 0x62, 0x79, 0x74, 0x65, 0x73, 0x18, 0x02, 0x20, 0x02, 0x28, 0x0d, 0x52, 0x06, 0x71, 0x62,
	0x79, 0x74, 0x65, 0x73, 0x12, 0x12, 0x0a, 0x04, 0x71, 0x6e, 0x75, 0x6d, 0x18, 0x03, 0x20, 0x02,
	0x28, 0x0d, 0x52, 0x04, 0x71, 0x6e, 0x75, 0x6d,
}

var (
	file_ipc_msg_proto_rawDescOnce sync.Once
	file_ipc_msg_proto_rawDescData = file_ipc_msg_proto_rawDesc
)

func file_ipc_msg_proto_rawDescGZIP() []byte {
	file_ipc_msg_proto_rawDescOnce.Do(func() {
		file_ipc_msg_proto_rawDescData = protoimpl.X.CompressGZIP(file_ipc_msg_proto_rawDescData)
	})
	return file_ipc_msg_proto_rawDescData
}

var file_ipc_msg_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_ipc_msg_proto_goTypes = []interface{}{
	(*IpcMsg)(nil),                // 0: ipc_msg
	(*IpcMsgEntry)(nil),           // 1: ipc_msg_entry
	(*ipc_desc.IpcDescEntry)(nil), // 2: ipc_desc_entry
}
var file_ipc_msg_proto_depIdxs = []int32{
	2, // 0: ipc_msg_entry.desc:type_name -> ipc_desc_entry
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_ipc_msg_proto_init() }
func file_ipc_msg_proto_init() {
	if File_ipc_msg_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_ipc_msg_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*IpcMsg); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_ipc_msg_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*IpcMsgEntry); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_ipc_msg_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_ipc_msg_proto_goTypes,
		DependencyIndexes: file_ipc_msg_proto_depIdxs,
		MessageInfos:      file_ipc_msg_proto_msgTypes,
	}.Build()
	File_ipc_msg_proto = out.File
	file_ipc_msg_proto_rawDesc = nil
	file_ipc_msg_proto_goTypes = nil
	file_ipc_msg_proto_depIdxs = nil
}
