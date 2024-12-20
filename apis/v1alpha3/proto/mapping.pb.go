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
// 	protoc-gen-go v1.36.0
// 	protoc        (unknown)
// source: proto/mapping.proto

package proto

import (
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

type GroupMapping struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Types that are valid to be assigned to Source:
	//
	//	*GroupMapping_GoogleGroups
	Source isGroupMapping_Source `protobuf_oneof:"source"`
	// Types that are valid to be assigned to Target:
	//
	//	*GroupMapping_Github
	//	*GroupMapping_Gitlab
	Target        isGroupMapping_Target `protobuf_oneof:"target"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GroupMapping) Reset() {
	*x = GroupMapping{}
	mi := &file_proto_mapping_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GroupMapping) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GroupMapping) ProtoMessage() {}

func (x *GroupMapping) ProtoReflect() protoreflect.Message {
	mi := &file_proto_mapping_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GroupMapping.ProtoReflect.Descriptor instead.
func (*GroupMapping) Descriptor() ([]byte, []int) {
	return file_proto_mapping_proto_rawDescGZIP(), []int{0}
}

func (x *GroupMapping) GetSource() isGroupMapping_Source {
	if x != nil {
		return x.Source
	}
	return nil
}

func (x *GroupMapping) GetGoogleGroups() *GoogleGroups {
	if x != nil {
		if x, ok := x.Source.(*GroupMapping_GoogleGroups); ok {
			return x.GoogleGroups
		}
	}
	return nil
}

func (x *GroupMapping) GetTarget() isGroupMapping_Target {
	if x != nil {
		return x.Target
	}
	return nil
}

func (x *GroupMapping) GetGithub() *GitHub {
	if x != nil {
		if x, ok := x.Target.(*GroupMapping_Github); ok {
			return x.Github
		}
	}
	return nil
}

func (x *GroupMapping) GetGitlab() *GitLab {
	if x != nil {
		if x, ok := x.Target.(*GroupMapping_Gitlab); ok {
			return x.Gitlab
		}
	}
	return nil
}

type isGroupMapping_Source interface {
	isGroupMapping_Source()
}

type GroupMapping_GoogleGroups struct {
	GoogleGroups *GoogleGroups `protobuf:"bytes,1,opt,name=google_groups,json=googleGroups,proto3,oneof"`
}

func (*GroupMapping_GoogleGroups) isGroupMapping_Source() {}

type isGroupMapping_Target interface {
	isGroupMapping_Target()
}

type GroupMapping_Github struct {
	Github *GitHub `protobuf:"bytes,2,opt,name=github,proto3,oneof"`
}

type GroupMapping_Gitlab struct {
	Gitlab *GitLab `protobuf:"bytes,3,opt,name=gitlab,proto3,oneof"`
}

func (*GroupMapping_Github) isGroupMapping_Target() {}

func (*GroupMapping_Gitlab) isGroupMapping_Target() {}

type GroupMappings struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Mappings      []*GroupMapping        `protobuf:"bytes,1,rep,name=mappings,proto3" json:"mappings,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GroupMappings) Reset() {
	*x = GroupMappings{}
	mi := &file_proto_mapping_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GroupMappings) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GroupMappings) ProtoMessage() {}

func (x *GroupMappings) ProtoReflect() protoreflect.Message {
	mi := &file_proto_mapping_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GroupMappings.ProtoReflect.Descriptor instead.
func (*GroupMappings) Descriptor() ([]byte, []int) {
	return file_proto_mapping_proto_rawDescGZIP(), []int{1}
}

func (x *GroupMappings) GetMappings() []*GroupMapping {
	if x != nil {
		return x.Mappings
	}
	return nil
}

type UserMapping struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Source        string                 `protobuf:"bytes,1,opt,name=source,proto3" json:"source,omitempty"`
	Target        string                 `protobuf:"bytes,2,opt,name=target,proto3" json:"target,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *UserMapping) Reset() {
	*x = UserMapping{}
	mi := &file_proto_mapping_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UserMapping) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserMapping) ProtoMessage() {}

func (x *UserMapping) ProtoReflect() protoreflect.Message {
	mi := &file_proto_mapping_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserMapping.ProtoReflect.Descriptor instead.
func (*UserMapping) Descriptor() ([]byte, []int) {
	return file_proto_mapping_proto_rawDescGZIP(), []int{2}
}

func (x *UserMapping) GetSource() string {
	if x != nil {
		return x.Source
	}
	return ""
}

func (x *UserMapping) GetTarget() string {
	if x != nil {
		return x.Target
	}
	return ""
}

type UserMappings struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Mappings      []*UserMapping         `protobuf:"bytes,1,rep,name=mappings,proto3" json:"mappings,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *UserMappings) Reset() {
	*x = UserMappings{}
	mi := &file_proto_mapping_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UserMappings) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserMappings) ProtoMessage() {}

func (x *UserMappings) ProtoReflect() protoreflect.Message {
	mi := &file_proto_mapping_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserMappings.ProtoReflect.Descriptor instead.
func (*UserMappings) Descriptor() ([]byte, []int) {
	return file_proto_mapping_proto_rawDescGZIP(), []int{3}
}

func (x *UserMappings) GetMappings() []*UserMapping {
	if x != nil {
		return x.Mappings
	}
	return nil
}

type TeamLinkMappings struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	GroupMappings *GroupMappings         `protobuf:"bytes,1,opt,name=group_mappings,json=groupMappings,proto3" json:"group_mappings,omitempty"`
	UserMappings  *UserMappings          `protobuf:"bytes,2,opt,name=user_mappings,json=userMappings,proto3" json:"user_mappings,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *TeamLinkMappings) Reset() {
	*x = TeamLinkMappings{}
	mi := &file_proto_mapping_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *TeamLinkMappings) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TeamLinkMappings) ProtoMessage() {}

func (x *TeamLinkMappings) ProtoReflect() protoreflect.Message {
	mi := &file_proto_mapping_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TeamLinkMappings.ProtoReflect.Descriptor instead.
func (*TeamLinkMappings) Descriptor() ([]byte, []int) {
	return file_proto_mapping_proto_rawDescGZIP(), []int{4}
}

func (x *TeamLinkMappings) GetGroupMappings() *GroupMappings {
	if x != nil {
		return x.GroupMappings
	}
	return nil
}

func (x *TeamLinkMappings) GetUserMappings() *UserMappings {
	if x != nil {
		return x.UserMappings
	}
	return nil
}

var File_proto_mapping_proto protoreflect.FileDescriptor

var file_proto_mapping_proto_rawDesc = []byte{
	0x0a, 0x13, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x6d, 0x61, 0x70, 0x70, 0x69, 0x6e, 0x67, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x09, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x61, 0x70, 0x69,
	0x1a, 0x11, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x67, 0x72, 0x6f, 0x75, 0x70, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x22, 0xbc, 0x01, 0x0a, 0x0c, 0x47, 0x72, 0x6f, 0x75, 0x70, 0x4d, 0x61, 0x70,
	0x70, 0x69, 0x6e, 0x67, 0x12, 0x3e, 0x0a, 0x0d, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x5f, 0x67,
	0x72, 0x6f, 0x75, 0x70, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x17, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x47, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x47, 0x72,
	0x6f, 0x75, 0x70, 0x73, 0x48, 0x00, 0x52, 0x0c, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x47, 0x72,
	0x6f, 0x75, 0x70, 0x73, 0x12, 0x2b, 0x0a, 0x06, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x11, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x61, 0x70, 0x69,
	0x2e, 0x47, 0x69, 0x74, 0x48, 0x75, 0x62, 0x48, 0x01, 0x52, 0x06, 0x67, 0x69, 0x74, 0x68, 0x75,
	0x62, 0x12, 0x2b, 0x0a, 0x06, 0x67, 0x69, 0x74, 0x6c, 0x61, 0x62, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x11, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x47, 0x69,
	0x74, 0x4c, 0x61, 0x62, 0x48, 0x01, 0x52, 0x06, 0x67, 0x69, 0x74, 0x6c, 0x61, 0x62, 0x42, 0x08,
	0x0a, 0x06, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x42, 0x08, 0x0a, 0x06, 0x74, 0x61, 0x72, 0x67,
	0x65, 0x74, 0x22, 0x44, 0x0a, 0x0d, 0x47, 0x72, 0x6f, 0x75, 0x70, 0x4d, 0x61, 0x70, 0x70, 0x69,
	0x6e, 0x67, 0x73, 0x12, 0x33, 0x0a, 0x08, 0x6d, 0x61, 0x70, 0x70, 0x69, 0x6e, 0x67, 0x73, 0x18,
	0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x17, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x61, 0x70,
	0x69, 0x2e, 0x47, 0x72, 0x6f, 0x75, 0x70, 0x4d, 0x61, 0x70, 0x70, 0x69, 0x6e, 0x67, 0x52, 0x08,
	0x6d, 0x61, 0x70, 0x70, 0x69, 0x6e, 0x67, 0x73, 0x22, 0x3d, 0x0a, 0x0b, 0x55, 0x73, 0x65, 0x72,
	0x4d, 0x61, 0x70, 0x70, 0x69, 0x6e, 0x67, 0x12, 0x16, 0x0a, 0x06, 0x73, 0x6f, 0x75, 0x72, 0x63,
	0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x12,
	0x16, 0x0a, 0x06, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x06, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x22, 0x42, 0x0a, 0x0c, 0x55, 0x73, 0x65, 0x72, 0x4d,
	0x61, 0x70, 0x70, 0x69, 0x6e, 0x67, 0x73, 0x12, 0x32, 0x0a, 0x08, 0x6d, 0x61, 0x70, 0x70, 0x69,
	0x6e, 0x67, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x16, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x55, 0x73, 0x65, 0x72, 0x4d, 0x61, 0x70, 0x70, 0x69, 0x6e,
	0x67, 0x52, 0x08, 0x6d, 0x61, 0x70, 0x70, 0x69, 0x6e, 0x67, 0x73, 0x22, 0x91, 0x01, 0x0a, 0x10,
	0x54, 0x65, 0x61, 0x6d, 0x4c, 0x69, 0x6e, 0x6b, 0x4d, 0x61, 0x70, 0x70, 0x69, 0x6e, 0x67, 0x73,
	0x12, 0x3f, 0x0a, 0x0e, 0x67, 0x72, 0x6f, 0x75, 0x70, 0x5f, 0x6d, 0x61, 0x70, 0x70, 0x69, 0x6e,
	0x67, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x18, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x2e, 0x61, 0x70, 0x69, 0x2e, 0x47, 0x72, 0x6f, 0x75, 0x70, 0x4d, 0x61, 0x70, 0x70, 0x69, 0x6e,
	0x67, 0x73, 0x52, 0x0d, 0x67, 0x72, 0x6f, 0x75, 0x70, 0x4d, 0x61, 0x70, 0x70, 0x69, 0x6e, 0x67,
	0x73, 0x12, 0x3c, 0x0a, 0x0d, 0x75, 0x73, 0x65, 0x72, 0x5f, 0x6d, 0x61, 0x70, 0x70, 0x69, 0x6e,
	0x67, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x17, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x2e, 0x61, 0x70, 0x69, 0x2e, 0x55, 0x73, 0x65, 0x72, 0x4d, 0x61, 0x70, 0x70, 0x69, 0x6e, 0x67,
	0x73, 0x52, 0x0c, 0x75, 0x73, 0x65, 0x72, 0x4d, 0x61, 0x70, 0x70, 0x69, 0x6e, 0x67, 0x73, 0x42,
	0x93, 0x01, 0x0a, 0x0d, 0x63, 0x6f, 0x6d, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x61, 0x70,
	0x69, 0x42, 0x0c, 0x4d, 0x61, 0x70, 0x70, 0x69, 0x6e, 0x67, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50,
	0x01, 0x5a, 0x2f, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x61, 0x62,
	0x63, 0x78, 0x79, 0x7a, 0x2f, 0x74, 0x65, 0x61, 0x6d, 0x2d, 0x6c, 0x69, 0x6e, 0x6b, 0x2f, 0x61,
	0x70, 0x69, 0x73, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x33, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0xa2, 0x02, 0x03, 0x50, 0x41, 0x58, 0xaa, 0x02, 0x09, 0x50, 0x72, 0x6f, 0x74, 0x6f,
	0x2e, 0x41, 0x70, 0x69, 0xca, 0x02, 0x09, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x5c, 0x41, 0x70, 0x69,
	0xe2, 0x02, 0x15, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x5c, 0x41, 0x70, 0x69, 0x5c, 0x47, 0x50, 0x42,
	0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0xea, 0x02, 0x0a, 0x50, 0x72, 0x6f, 0x74, 0x6f,
	0x3a, 0x3a, 0x41, 0x70, 0x69, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_proto_mapping_proto_rawDescOnce sync.Once
	file_proto_mapping_proto_rawDescData = file_proto_mapping_proto_rawDesc
)

func file_proto_mapping_proto_rawDescGZIP() []byte {
	file_proto_mapping_proto_rawDescOnce.Do(func() {
		file_proto_mapping_proto_rawDescData = protoimpl.X.CompressGZIP(file_proto_mapping_proto_rawDescData)
	})
	return file_proto_mapping_proto_rawDescData
}

var file_proto_mapping_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_proto_mapping_proto_goTypes = []any{
	(*GroupMapping)(nil),     // 0: proto.api.GroupMapping
	(*GroupMappings)(nil),    // 1: proto.api.GroupMappings
	(*UserMapping)(nil),      // 2: proto.api.UserMapping
	(*UserMappings)(nil),     // 3: proto.api.UserMappings
	(*TeamLinkMappings)(nil), // 4: proto.api.TeamLinkMappings
	(*GoogleGroups)(nil),     // 5: proto.api.GoogleGroups
	(*GitHub)(nil),           // 6: proto.api.GitHub
	(*GitLab)(nil),           // 7: proto.api.GitLab
}
var file_proto_mapping_proto_depIdxs = []int32{
	5, // 0: proto.api.GroupMapping.google_groups:type_name -> proto.api.GoogleGroups
	6, // 1: proto.api.GroupMapping.github:type_name -> proto.api.GitHub
	7, // 2: proto.api.GroupMapping.gitlab:type_name -> proto.api.GitLab
	0, // 3: proto.api.GroupMappings.mappings:type_name -> proto.api.GroupMapping
	2, // 4: proto.api.UserMappings.mappings:type_name -> proto.api.UserMapping
	1, // 5: proto.api.TeamLinkMappings.group_mappings:type_name -> proto.api.GroupMappings
	3, // 6: proto.api.TeamLinkMappings.user_mappings:type_name -> proto.api.UserMappings
	7, // [7:7] is the sub-list for method output_type
	7, // [7:7] is the sub-list for method input_type
	7, // [7:7] is the sub-list for extension type_name
	7, // [7:7] is the sub-list for extension extendee
	0, // [0:7] is the sub-list for field type_name
}

func init() { file_proto_mapping_proto_init() }
func file_proto_mapping_proto_init() {
	if File_proto_mapping_proto != nil {
		return
	}
	file_proto_group_proto_init()
	file_proto_mapping_proto_msgTypes[0].OneofWrappers = []any{
		(*GroupMapping_GoogleGroups)(nil),
		(*GroupMapping_Github)(nil),
		(*GroupMapping_Gitlab)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_proto_mapping_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_proto_mapping_proto_goTypes,
		DependencyIndexes: file_proto_mapping_proto_depIdxs,
		MessageInfos:      file_proto_mapping_proto_msgTypes,
	}.Build()
	File_proto_mapping_proto = out.File
	file_proto_mapping_proto_rawDesc = nil
	file_proto_mapping_proto_goTypes = nil
	file_proto_mapping_proto_depIdxs = nil
}
