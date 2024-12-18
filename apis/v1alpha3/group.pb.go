// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.35.2
// 	protoc        (unknown)
// source: group.proto

package v1alpha3

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

type GoogleGroup struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	GroupId *string `protobuf:"bytes,1,req,name=group_id,json=groupId" json:"group_id,omitempty"`
}

func (x *GoogleGroup) Reset() {
	*x = GoogleGroup{}
	mi := &file_group_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GoogleGroup) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GoogleGroup) ProtoMessage() {}

func (x *GoogleGroup) ProtoReflect() protoreflect.Message {
	mi := &file_group_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GoogleGroup.ProtoReflect.Descriptor instead.
func (*GoogleGroup) Descriptor() ([]byte, []int) {
	return file_group_proto_rawDescGZIP(), []int{0}
}

func (x *GoogleGroup) GetGroupId() string {
	if x != nil && x.GroupId != nil {
		return *x.GroupId
	}
	return ""
}

type GitHubTeam struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	OrgId  *int64 `protobuf:"varint,1,req,name=org_id,json=orgId" json:"org_id,omitempty"`
	TeamId *int64 `protobuf:"varint,2,req,name=team_id,json=teamId" json:"team_id,omitempty"`
}

func (x *GitHubTeam) Reset() {
	*x = GitHubTeam{}
	mi := &file_group_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GitHubTeam) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GitHubTeam) ProtoMessage() {}

func (x *GitHubTeam) ProtoReflect() protoreflect.Message {
	mi := &file_group_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GitHubTeam.ProtoReflect.Descriptor instead.
func (*GitHubTeam) Descriptor() ([]byte, []int) {
	return file_group_proto_rawDescGZIP(), []int{1}
}

func (x *GitHubTeam) GetOrgId() int64 {
	if x != nil && x.OrgId != nil {
		return *x.OrgId
	}
	return 0
}

func (x *GitHubTeam) GetTeamId() int64 {
	if x != nil && x.TeamId != nil {
		return *x.TeamId
	}
	return 0
}

type GoogleGroupToGitHubTeamMapping struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	GoogleGroup *GoogleGroup `protobuf:"bytes,1,req,name=google_group,json=googleGroup" json:"google_group,omitempty"`
	GitHubTeam  *GitHubTeam  `protobuf:"bytes,2,req,name=git_hub_team,json=gitHubTeam" json:"git_hub_team,omitempty"`
}

func (x *GoogleGroupToGitHubTeamMapping) Reset() {
	*x = GoogleGroupToGitHubTeamMapping{}
	mi := &file_group_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GoogleGroupToGitHubTeamMapping) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GoogleGroupToGitHubTeamMapping) ProtoMessage() {}

func (x *GoogleGroupToGitHubTeamMapping) ProtoReflect() protoreflect.Message {
	mi := &file_group_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GoogleGroupToGitHubTeamMapping.ProtoReflect.Descriptor instead.
func (*GoogleGroupToGitHubTeamMapping) Descriptor() ([]byte, []int) {
	return file_group_proto_rawDescGZIP(), []int{2}
}

func (x *GoogleGroupToGitHubTeamMapping) GetGoogleGroup() *GoogleGroup {
	if x != nil {
		return x.GoogleGroup
	}
	return nil
}

func (x *GoogleGroupToGitHubTeamMapping) GetGitHubTeam() *GitHubTeam {
	if x != nil {
		return x.GitHubTeam
	}
	return nil
}

type GoogleGroupToGitHubTeamMappings struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Mappings []*GoogleGroupToGitHubTeamMapping `protobuf:"bytes,1,rep,name=mappings" json:"mappings,omitempty"`
}

func (x *GoogleGroupToGitHubTeamMappings) Reset() {
	*x = GoogleGroupToGitHubTeamMappings{}
	mi := &file_group_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GoogleGroupToGitHubTeamMappings) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GoogleGroupToGitHubTeamMappings) ProtoMessage() {}

func (x *GoogleGroupToGitHubTeamMappings) ProtoReflect() protoreflect.Message {
	mi := &file_group_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GoogleGroupToGitHubTeamMappings.ProtoReflect.Descriptor instead.
func (*GoogleGroupToGitHubTeamMappings) Descriptor() ([]byte, []int) {
	return file_group_proto_rawDescGZIP(), []int{3}
}

func (x *GoogleGroupToGitHubTeamMappings) GetMappings() []*GoogleGroupToGitHubTeamMapping {
	if x != nil {
		return x.Mappings
	}
	return nil
}

var File_group_proto protoreflect.FileDescriptor

var file_group_proto_rawDesc = []byte{
	0x0a, 0x0b, 0x67, 0x72, 0x6f, 0x75, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x28, 0x0a,
	0x0b, 0x47, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x47, 0x72, 0x6f, 0x75, 0x70, 0x12, 0x19, 0x0a, 0x08,
	0x67, 0x72, 0x6f, 0x75, 0x70, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x02, 0x28, 0x09, 0x52, 0x07,
	0x67, 0x72, 0x6f, 0x75, 0x70, 0x49, 0x64, 0x22, 0x3c, 0x0a, 0x0a, 0x47, 0x69, 0x74, 0x48, 0x75,
	0x62, 0x54, 0x65, 0x61, 0x6d, 0x12, 0x15, 0x0a, 0x06, 0x6f, 0x72, 0x67, 0x5f, 0x69, 0x64, 0x18,
	0x01, 0x20, 0x02, 0x28, 0x03, 0x52, 0x05, 0x6f, 0x72, 0x67, 0x49, 0x64, 0x12, 0x17, 0x0a, 0x07,
	0x74, 0x65, 0x61, 0x6d, 0x5f, 0x69, 0x64, 0x18, 0x02, 0x20, 0x02, 0x28, 0x03, 0x52, 0x06, 0x74,
	0x65, 0x61, 0x6d, 0x49, 0x64, 0x22, 0x80, 0x01, 0x0a, 0x1e, 0x47, 0x6f, 0x6f, 0x67, 0x6c, 0x65,
	0x47, 0x72, 0x6f, 0x75, 0x70, 0x54, 0x6f, 0x47, 0x69, 0x74, 0x48, 0x75, 0x62, 0x54, 0x65, 0x61,
	0x6d, 0x4d, 0x61, 0x70, 0x70, 0x69, 0x6e, 0x67, 0x12, 0x2f, 0x0a, 0x0c, 0x67, 0x6f, 0x6f, 0x67,
	0x6c, 0x65, 0x5f, 0x67, 0x72, 0x6f, 0x75, 0x70, 0x18, 0x01, 0x20, 0x02, 0x28, 0x0b, 0x32, 0x0c,
	0x2e, 0x47, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x47, 0x72, 0x6f, 0x75, 0x70, 0x52, 0x0b, 0x67, 0x6f,
	0x6f, 0x67, 0x6c, 0x65, 0x47, 0x72, 0x6f, 0x75, 0x70, 0x12, 0x2d, 0x0a, 0x0c, 0x67, 0x69, 0x74,
	0x5f, 0x68, 0x75, 0x62, 0x5f, 0x74, 0x65, 0x61, 0x6d, 0x18, 0x02, 0x20, 0x02, 0x28, 0x0b, 0x32,
	0x0b, 0x2e, 0x47, 0x69, 0x74, 0x48, 0x75, 0x62, 0x54, 0x65, 0x61, 0x6d, 0x52, 0x0a, 0x67, 0x69,
	0x74, 0x48, 0x75, 0x62, 0x54, 0x65, 0x61, 0x6d, 0x22, 0x5e, 0x0a, 0x1f, 0x47, 0x6f, 0x6f, 0x67,
	0x6c, 0x65, 0x47, 0x72, 0x6f, 0x75, 0x70, 0x54, 0x6f, 0x47, 0x69, 0x74, 0x48, 0x75, 0x62, 0x54,
	0x65, 0x61, 0x6d, 0x4d, 0x61, 0x70, 0x70, 0x69, 0x6e, 0x67, 0x73, 0x12, 0x3b, 0x0a, 0x08, 0x6d,
	0x61, 0x70, 0x70, 0x69, 0x6e, 0x67, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x1f, 0x2e,
	0x47, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x47, 0x72, 0x6f, 0x75, 0x70, 0x54, 0x6f, 0x47, 0x69, 0x74,
	0x48, 0x75, 0x62, 0x54, 0x65, 0x61, 0x6d, 0x4d, 0x61, 0x70, 0x70, 0x69, 0x6e, 0x67, 0x52, 0x08,
	0x6d, 0x61, 0x70, 0x70, 0x69, 0x6e, 0x67, 0x73, 0x42, 0x39, 0x42, 0x0a, 0x47, 0x72, 0x6f, 0x75,
	0x70, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x29, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62,
	0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x61, 0x62, 0x63, 0x78, 0x79, 0x7a, 0x2f, 0x74, 0x65, 0x61, 0x6d,
	0x2d, 0x6c, 0x69, 0x6e, 0x6b, 0x2f, 0x61, 0x70, 0x69, 0x73, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70,
	0x68, 0x61, 0x33,
}

var (
	file_group_proto_rawDescOnce sync.Once
	file_group_proto_rawDescData = file_group_proto_rawDesc
)

func file_group_proto_rawDescGZIP() []byte {
	file_group_proto_rawDescOnce.Do(func() {
		file_group_proto_rawDescData = protoimpl.X.CompressGZIP(file_group_proto_rawDescData)
	})
	return file_group_proto_rawDescData
}

var file_group_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_group_proto_goTypes = []any{
	(*GoogleGroup)(nil),                     // 0: GoogleGroup
	(*GitHubTeam)(nil),                      // 1: GitHubTeam
	(*GoogleGroupToGitHubTeamMapping)(nil),  // 2: GoogleGroupToGitHubTeamMapping
	(*GoogleGroupToGitHubTeamMappings)(nil), // 3: GoogleGroupToGitHubTeamMappings
}
var file_group_proto_depIdxs = []int32{
	0, // 0: GoogleGroupToGitHubTeamMapping.google_group:type_name -> GoogleGroup
	1, // 1: GoogleGroupToGitHubTeamMapping.git_hub_team:type_name -> GitHubTeam
	2, // 2: GoogleGroupToGitHubTeamMappings.mappings:type_name -> GoogleGroupToGitHubTeamMapping
	3, // [3:3] is the sub-list for method output_type
	3, // [3:3] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_group_proto_init() }
func file_group_proto_init() {
	if File_group_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_group_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_group_proto_goTypes,
		DependencyIndexes: file_group_proto_depIdxs,
		MessageInfos:      file_group_proto_msgTypes,
	}.Build()
	File_group_proto = out.File
	file_group_proto_rawDesc = nil
	file_group_proto_goTypes = nil
	file_group_proto_depIdxs = nil
}
