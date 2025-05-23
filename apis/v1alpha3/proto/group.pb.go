// Copyright 2024 The Authors (see AUTHORS file)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.4
// 	protoc        (unknown)
// source: proto/group.proto

package proto

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

type GitHub struct {
	state                protoimpl.MessageState `protogen:"open.v1"`
	OrgId                int64                  `protobuf:"varint,1,opt,name=org_id,json=orgId,proto3" json:"org_id,omitempty"`
	TeamId               int64                  `protobuf:"varint,2,opt,name=team_id,json=teamId,proto3" json:"team_id,omitempty"`
	RequireUserEnableSso bool                   `protobuf:"varint,3,opt,name=require_user_enable_sso,json=requireUserEnableSso,proto3" json:"require_user_enable_sso,omitempty"`
	unknownFields        protoimpl.UnknownFields
	sizeCache            protoimpl.SizeCache
}

func (x *GitHub) Reset() {
	*x = GitHub{}
	mi := &file_proto_group_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GitHub) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GitHub) ProtoMessage() {}

func (x *GitHub) ProtoReflect() protoreflect.Message {
	mi := &file_proto_group_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GitHub.ProtoReflect.Descriptor instead.
func (*GitHub) Descriptor() ([]byte, []int) {
	return file_proto_group_proto_rawDescGZIP(), []int{0}
}

func (x *GitHub) GetOrgId() int64 {
	if x != nil {
		return x.OrgId
	}
	return 0
}

func (x *GitHub) GetTeamId() int64 {
	if x != nil {
		return x.TeamId
	}
	return 0
}

func (x *GitHub) GetRequireUserEnableSso() bool {
	if x != nil {
		return x.RequireUserEnableSso
	}
	return false
}

type GitLab struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	GroupId       int64                  `protobuf:"varint,1,opt,name=group_id,json=groupId,proto3" json:"group_id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GitLab) Reset() {
	*x = GitLab{}
	mi := &file_proto_group_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GitLab) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GitLab) ProtoMessage() {}

func (x *GitLab) ProtoReflect() protoreflect.Message {
	mi := &file_proto_group_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GitLab.ProtoReflect.Descriptor instead.
func (*GitLab) Descriptor() ([]byte, []int) {
	return file_proto_group_proto_rawDescGZIP(), []int{1}
}

func (x *GitLab) GetGroupId() int64 {
	if x != nil {
		return x.GroupId
	}
	return 0
}

type GoogleGroups struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	GroupId       string                 `protobuf:"bytes,1,opt,name=group_id,json=groupId,proto3" json:"group_id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GoogleGroups) Reset() {
	*x = GoogleGroups{}
	mi := &file_proto_group_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GoogleGroups) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GoogleGroups) ProtoMessage() {}

func (x *GoogleGroups) ProtoReflect() protoreflect.Message {
	mi := &file_proto_group_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GoogleGroups.ProtoReflect.Descriptor instead.
func (*GoogleGroups) Descriptor() ([]byte, []int) {
	return file_proto_group_proto_rawDescGZIP(), []int{2}
}

func (x *GoogleGroups) GetGroupId() string {
	if x != nil {
		return x.GroupId
	}
	return ""
}

var File_proto_group_proto protoreflect.FileDescriptor

var file_proto_group_proto_rawDesc = string([]byte{
	0x0a, 0x11, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x67, 0x72, 0x6f, 0x75, 0x70, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x12, 0x09, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x61, 0x70, 0x69, 0x22, 0x6f,
	0x0a, 0x06, 0x47, 0x69, 0x74, 0x48, 0x75, 0x62, 0x12, 0x15, 0x0a, 0x06, 0x6f, 0x72, 0x67, 0x5f,
	0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03, 0x52, 0x05, 0x6f, 0x72, 0x67, 0x49, 0x64, 0x12,
	0x17, 0x0a, 0x07, 0x74, 0x65, 0x61, 0x6d, 0x5f, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03,
	0x52, 0x06, 0x74, 0x65, 0x61, 0x6d, 0x49, 0x64, 0x12, 0x35, 0x0a, 0x17, 0x72, 0x65, 0x71, 0x75,
	0x69, 0x72, 0x65, 0x5f, 0x75, 0x73, 0x65, 0x72, 0x5f, 0x65, 0x6e, 0x61, 0x62, 0x6c, 0x65, 0x5f,
	0x73, 0x73, 0x6f, 0x18, 0x03, 0x20, 0x01, 0x28, 0x08, 0x52, 0x14, 0x72, 0x65, 0x71, 0x75, 0x69,
	0x72, 0x65, 0x55, 0x73, 0x65, 0x72, 0x45, 0x6e, 0x61, 0x62, 0x6c, 0x65, 0x53, 0x73, 0x6f, 0x22,
	0x23, 0x0a, 0x06, 0x47, 0x69, 0x74, 0x4c, 0x61, 0x62, 0x12, 0x19, 0x0a, 0x08, 0x67, 0x72, 0x6f,
	0x75, 0x70, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03, 0x52, 0x07, 0x67, 0x72, 0x6f,
	0x75, 0x70, 0x49, 0x64, 0x22, 0x29, 0x0a, 0x0c, 0x47, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x47, 0x72,
	0x6f, 0x75, 0x70, 0x73, 0x12, 0x19, 0x0a, 0x08, 0x67, 0x72, 0x6f, 0x75, 0x70, 0x5f, 0x69, 0x64,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x67, 0x72, 0x6f, 0x75, 0x70, 0x49, 0x64, 0x42,
	0x91, 0x01, 0x0a, 0x0d, 0x63, 0x6f, 0x6d, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x61, 0x70,
	0x69, 0x42, 0x0a, 0x47, 0x72, 0x6f, 0x75, 0x70, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a,
	0x2f, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x61, 0x62, 0x63, 0x78,
	0x79, 0x7a, 0x2f, 0x74, 0x65, 0x61, 0x6d, 0x2d, 0x6c, 0x69, 0x6e, 0x6b, 0x2f, 0x61, 0x70, 0x69,
	0x73, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x33, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0xa2, 0x02, 0x03, 0x50, 0x41, 0x58, 0xaa, 0x02, 0x09, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x41,
	0x70, 0x69, 0xca, 0x02, 0x09, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x5c, 0x41, 0x70, 0x69, 0xe2, 0x02,
	0x15, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x5c, 0x41, 0x70, 0x69, 0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65,
	0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0xea, 0x02, 0x0a, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x3a, 0x3a,
	0x41, 0x70, 0x69, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
})

var (
	file_proto_group_proto_rawDescOnce sync.Once
	file_proto_group_proto_rawDescData []byte
)

func file_proto_group_proto_rawDescGZIP() []byte {
	file_proto_group_proto_rawDescOnce.Do(func() {
		file_proto_group_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_proto_group_proto_rawDesc), len(file_proto_group_proto_rawDesc)))
	})
	return file_proto_group_proto_rawDescData
}

var file_proto_group_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_proto_group_proto_goTypes = []any{
	(*GitHub)(nil),       // 0: proto.api.GitHub
	(*GitLab)(nil),       // 1: proto.api.GitLab
	(*GoogleGroups)(nil), // 2: proto.api.GoogleGroups
}
var file_proto_group_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_proto_group_proto_init() }
func file_proto_group_proto_init() {
	if File_proto_group_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_proto_group_proto_rawDesc), len(file_proto_group_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_proto_group_proto_goTypes,
		DependencyIndexes: file_proto_group_proto_depIdxs,
		MessageInfos:      file_proto_group_proto_msgTypes,
	}.Build()
	File_proto_group_proto = out.File
	file_proto_group_proto_goTypes = nil
	file_proto_group_proto_depIdxs = nil
}
