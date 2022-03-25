// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0-devel
// 	protoc        v3.13.0
// source: api/proto/v1/mixes.proto

package v1

import (
	timestamp "github.com/golang/protobuf/ptypes/timestamp"
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

type Mix struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id          string               `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Category    string               `protobuf:"bytes,2,opt,name=category,proto3" json:"category,omitempty"`
	PostIds     []string             `protobuf:"bytes,3,rep,name=post_ids,json=postIds,proto3" json:"post_ids,omitempty"`
	Background  *Background          `protobuf:"bytes,4,opt,name=background,proto3" json:"background,omitempty"`
	RequestedAt *timestamp.Timestamp `protobuf:"bytes,5,opt,name=requested_at,json=requestedAt,proto3" json:"requested_at,omitempty"`
	Title       string               `protobuf:"bytes,6,opt,name=title,proto3" json:"title,omitempty"`
}

func (x *Mix) Reset() {
	*x = Mix{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_proto_v1_mixes_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Mix) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Mix) ProtoMessage() {}

func (x *Mix) ProtoReflect() protoreflect.Message {
	mi := &file_api_proto_v1_mixes_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Mix.ProtoReflect.Descriptor instead.
func (*Mix) Descriptor() ([]byte, []int) {
	return file_api_proto_v1_mixes_proto_rawDescGZIP(), []int{0}
}

func (x *Mix) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *Mix) GetCategory() string {
	if x != nil {
		return x.Category
	}
	return ""
}

func (x *Mix) GetPostIds() []string {
	if x != nil {
		return x.PostIds
	}
	return nil
}

func (x *Mix) GetBackground() *Background {
	if x != nil {
		return x.Background
	}
	return nil
}

func (x *Mix) GetRequestedAt() *timestamp.Timestamp {
	if x != nil {
		return x.RequestedAt
	}
	return nil
}

func (x *Mix) GetTitle() string {
	if x != nil {
		return x.Title
	}
	return ""
}

type Background struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Fallback        string `protobuf:"bytes,1,opt,name=fallback,proto3" json:"fallback,omitempty"`
	BackgroundImage string `protobuf:"bytes,2,opt,name=background_image,json=backgroundImage,proto3" json:"background_image,omitempty"`
}

func (x *Background) Reset() {
	*x = Background{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_proto_v1_mixes_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Background) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Background) ProtoMessage() {}

func (x *Background) ProtoReflect() protoreflect.Message {
	mi := &file_api_proto_v1_mixes_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Background.ProtoReflect.Descriptor instead.
func (*Background) Descriptor() ([]byte, []int) {
	return file_api_proto_v1_mixes_proto_rawDescGZIP(), []int{1}
}

func (x *Background) GetFallback() string {
	if x != nil {
		return x.Fallback
	}
	return ""
}

func (x *Background) GetBackgroundImage() string {
	if x != nil {
		return x.BackgroundImage
	}
	return ""
}

type GetMixesRes struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Mixes []*Mix `protobuf:"bytes,1,rep,name=mixes,proto3" json:"mixes,omitempty"`
}

func (x *GetMixesRes) Reset() {
	*x = GetMixesRes{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_proto_v1_mixes_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetMixesRes) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetMixesRes) ProtoMessage() {}

func (x *GetMixesRes) ProtoReflect() protoreflect.Message {
	mi := &file_api_proto_v1_mixes_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetMixesRes.ProtoReflect.Descriptor instead.
func (*GetMixesRes) Descriptor() ([]byte, []int) {
	return file_api_proto_v1_mixes_proto_rawDescGZIP(), []int{2}
}

func (x *GetMixesRes) GetMixes() []*Mix {
	if x != nil {
		return x.Mixes
	}
	return nil
}

var File_api_proto_v1_mixes_proto protoreflect.FileDescriptor

var file_api_proto_v1_mixes_proto_rawDesc = []byte{
	0x0a, 0x18, 0x61, 0x70, 0x69, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x76, 0x31, 0x2f, 0x6d,
	0x69, 0x78, 0x65, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x02, 0x76, 0x31, 0x1a, 0x1f,
	0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f,
	0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22,
	0xd1, 0x01, 0x0a, 0x03, 0x4d, 0x69, 0x78, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x1a, 0x0a, 0x08, 0x63, 0x61, 0x74, 0x65, 0x67,
	0x6f, 0x72, 0x79, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x63, 0x61, 0x74, 0x65, 0x67,
	0x6f, 0x72, 0x79, 0x12, 0x19, 0x0a, 0x08, 0x70, 0x6f, 0x73, 0x74, 0x5f, 0x69, 0x64, 0x73, 0x18,
	0x03, 0x20, 0x03, 0x28, 0x09, 0x52, 0x07, 0x70, 0x6f, 0x73, 0x74, 0x49, 0x64, 0x73, 0x12, 0x2e,
	0x0a, 0x0a, 0x62, 0x61, 0x63, 0x6b, 0x67, 0x72, 0x6f, 0x75, 0x6e, 0x64, 0x18, 0x04, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x0e, 0x2e, 0x76, 0x31, 0x2e, 0x42, 0x61, 0x63, 0x6b, 0x67, 0x72, 0x6f, 0x75,
	0x6e, 0x64, 0x52, 0x0a, 0x62, 0x61, 0x63, 0x6b, 0x67, 0x72, 0x6f, 0x75, 0x6e, 0x64, 0x12, 0x3d,
	0x0a, 0x0c, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x65, 0x64, 0x5f, 0x61, 0x74, 0x18, 0x05,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70,
	0x52, 0x0b, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x65, 0x64, 0x41, 0x74, 0x12, 0x14, 0x0a,
	0x05, 0x74, 0x69, 0x74, 0x6c, 0x65, 0x18, 0x06, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x74, 0x69,
	0x74, 0x6c, 0x65, 0x22, 0x53, 0x0a, 0x0a, 0x42, 0x61, 0x63, 0x6b, 0x67, 0x72, 0x6f, 0x75, 0x6e,
	0x64, 0x12, 0x1a, 0x0a, 0x08, 0x66, 0x61, 0x6c, 0x6c, 0x62, 0x61, 0x63, 0x6b, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x08, 0x66, 0x61, 0x6c, 0x6c, 0x62, 0x61, 0x63, 0x6b, 0x12, 0x29, 0x0a,
	0x10, 0x62, 0x61, 0x63, 0x6b, 0x67, 0x72, 0x6f, 0x75, 0x6e, 0x64, 0x5f, 0x69, 0x6d, 0x61, 0x67,
	0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0f, 0x62, 0x61, 0x63, 0x6b, 0x67, 0x72, 0x6f,
	0x75, 0x6e, 0x64, 0x49, 0x6d, 0x61, 0x67, 0x65, 0x22, 0x2c, 0x0a, 0x0b, 0x47, 0x65, 0x74, 0x4d,
	0x69, 0x78, 0x65, 0x73, 0x52, 0x65, 0x73, 0x12, 0x1d, 0x0a, 0x05, 0x6d, 0x69, 0x78, 0x65, 0x73,
	0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x07, 0x2e, 0x76, 0x31, 0x2e, 0x4d, 0x69, 0x78, 0x52,
	0x05, 0x6d, 0x69, 0x78, 0x65, 0x73, 0x42, 0x0c, 0x5a, 0x0a, 0x70, 0x6b, 0x67, 0x2f, 0x61, 0x70,
	0x69, 0x2f, 0x76, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_api_proto_v1_mixes_proto_rawDescOnce sync.Once
	file_api_proto_v1_mixes_proto_rawDescData = file_api_proto_v1_mixes_proto_rawDesc
)

func file_api_proto_v1_mixes_proto_rawDescGZIP() []byte {
	file_api_proto_v1_mixes_proto_rawDescOnce.Do(func() {
		file_api_proto_v1_mixes_proto_rawDescData = protoimpl.X.CompressGZIP(file_api_proto_v1_mixes_proto_rawDescData)
	})
	return file_api_proto_v1_mixes_proto_rawDescData
}

var file_api_proto_v1_mixes_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_api_proto_v1_mixes_proto_goTypes = []interface{}{
	(*Mix)(nil),                 // 0: v1.Mix
	(*Background)(nil),          // 1: v1.Background
	(*GetMixesRes)(nil),         // 2: v1.GetMixesRes
	(*timestamp.Timestamp)(nil), // 3: google.protobuf.Timestamp
}
var file_api_proto_v1_mixes_proto_depIdxs = []int32{
	1, // 0: v1.Mix.background:type_name -> v1.Background
	3, // 1: v1.Mix.requested_at:type_name -> google.protobuf.Timestamp
	0, // 2: v1.GetMixesRes.mixes:type_name -> v1.Mix
	3, // [3:3] is the sub-list for method output_type
	3, // [3:3] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_api_proto_v1_mixes_proto_init() }
func file_api_proto_v1_mixes_proto_init() {
	if File_api_proto_v1_mixes_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_api_proto_v1_mixes_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Mix); i {
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
		file_api_proto_v1_mixes_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Background); i {
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
		file_api_proto_v1_mixes_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetMixesRes); i {
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
			RawDescriptor: file_api_proto_v1_mixes_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_api_proto_v1_mixes_proto_goTypes,
		DependencyIndexes: file_api_proto_v1_mixes_proto_depIdxs,
		MessageInfos:      file_api_proto_v1_mixes_proto_msgTypes,
	}.Build()
	File_api_proto_v1_mixes_proto = out.File
	file_api_proto_v1_mixes_proto_rawDesc = nil
	file_api_proto_v1_mixes_proto_goTypes = nil
	file_api_proto_v1_mixes_proto_depIdxs = nil
}