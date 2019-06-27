// Code generated by protoc-gen-go. DO NOT EDIT.
// source: pkg/art/art.proto

package net_audiostrike_art

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

type ArtRequest struct {
	ArtistId             string   `protobuf:"bytes,1,opt,name=artist_id,json=artistId,proto3" json:"artist_id,omitempty"`
	ArtistTrackId        string   `protobuf:"bytes,2,opt,name=artist_track_id,json=artistTrackId,proto3" json:"artist_track_id,omitempty"`
	Since                uint64   `protobuf:"varint,3,opt,name=since,proto3" json:"since,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ArtRequest) Reset()         { *m = ArtRequest{} }
func (m *ArtRequest) String() string { return proto.CompactTextString(m) }
func (*ArtRequest) ProtoMessage()    {}
func (*ArtRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_a83fef21c75be787, []int{0}
}

func (m *ArtRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ArtRequest.Unmarshal(m, b)
}
func (m *ArtRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ArtRequest.Marshal(b, m, deterministic)
}
func (m *ArtRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ArtRequest.Merge(m, src)
}
func (m *ArtRequest) XXX_Size() int {
	return xxx_messageInfo_ArtRequest.Size(m)
}
func (m *ArtRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_ArtRequest.DiscardUnknown(m)
}

var xxx_messageInfo_ArtRequest proto.InternalMessageInfo

func (m *ArtRequest) GetArtistId() string {
	if m != nil {
		return m.ArtistId
	}
	return ""
}

func (m *ArtRequest) GetArtistTrackId() string {
	if m != nil {
		return m.ArtistTrackId
	}
	return ""
}

func (m *ArtRequest) GetSince() uint64 {
	if m != nil {
		return m.Since
	}
	return 0
}

type ArtReply struct {
	Artists              []*Artist `protobuf:"bytes,1,rep,name=artists,proto3" json:"artists,omitempty"`
	Albums               []*Album  `protobuf:"bytes,2,rep,name=albums,proto3" json:"albums,omitempty"`
	Tracks               []*Track  `protobuf:"bytes,3,rep,name=tracks,proto3" json:"tracks,omitempty"`
	Peers                []*Peer   `protobuf:"bytes,4,rep,name=peers,proto3" json:"peers,omitempty"`
	XXX_NoUnkeyedLiteral struct{}  `json:"-"`
	XXX_unrecognized     []byte    `json:"-"`
	XXX_sizecache        int32     `json:"-"`
}

func (m *ArtReply) Reset()         { *m = ArtReply{} }
func (m *ArtReply) String() string { return proto.CompactTextString(m) }
func (*ArtReply) ProtoMessage()    {}
func (*ArtReply) Descriptor() ([]byte, []int) {
	return fileDescriptor_a83fef21c75be787, []int{1}
}

func (m *ArtReply) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ArtReply.Unmarshal(m, b)
}
func (m *ArtReply) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ArtReply.Marshal(b, m, deterministic)
}
func (m *ArtReply) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ArtReply.Merge(m, src)
}
func (m *ArtReply) XXX_Size() int {
	return xxx_messageInfo_ArtReply.Size(m)
}
func (m *ArtReply) XXX_DiscardUnknown() {
	xxx_messageInfo_ArtReply.DiscardUnknown(m)
}

var xxx_messageInfo_ArtReply proto.InternalMessageInfo

func (m *ArtReply) GetArtists() []*Artist {
	if m != nil {
		return m.Artists
	}
	return nil
}

func (m *ArtReply) GetAlbums() []*Album {
	if m != nil {
		return m.Albums
	}
	return nil
}

func (m *ArtReply) GetTracks() []*Track {
	if m != nil {
		return m.Tracks
	}
	return nil
}

func (m *ArtReply) GetPeers() []*Peer {
	if m != nil {
		return m.Peers
	}
	return nil
}

type Artist struct {
	ArtistId             string   `protobuf:"bytes,1,opt,name=artist_id,json=artistId,proto3" json:"artist_id,omitempty"`
	Name                 string   `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	Pubkey               string   `protobuf:"bytes,3,opt,name=pubkey,proto3" json:"pubkey,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Artist) Reset()         { *m = Artist{} }
func (m *Artist) String() string { return proto.CompactTextString(m) }
func (*Artist) ProtoMessage()    {}
func (*Artist) Descriptor() ([]byte, []int) {
	return fileDescriptor_a83fef21c75be787, []int{2}
}

func (m *Artist) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Artist.Unmarshal(m, b)
}
func (m *Artist) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Artist.Marshal(b, m, deterministic)
}
func (m *Artist) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Artist.Merge(m, src)
}
func (m *Artist) XXX_Size() int {
	return xxx_messageInfo_Artist.Size(m)
}
func (m *Artist) XXX_DiscardUnknown() {
	xxx_messageInfo_Artist.DiscardUnknown(m)
}

var xxx_messageInfo_Artist proto.InternalMessageInfo

func (m *Artist) GetArtistId() string {
	if m != nil {
		return m.ArtistId
	}
	return ""
}

func (m *Artist) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *Artist) GetPubkey() string {
	if m != nil {
		return m.Pubkey
	}
	return ""
}

type Album struct {
	ArtistId             string   `protobuf:"bytes,1,opt,name=artist_id,json=artistId,proto3" json:"artist_id,omitempty"`
	ArtistAlbumId        string   `protobuf:"bytes,2,opt,name=artist_album_id,json=artistAlbumId,proto3" json:"artist_album_id,omitempty"`
	Title                string   `protobuf:"bytes,3,opt,name=title,proto3" json:"title,omitempty"`
	ArtistTrackId        []string `protobuf:"bytes,4,rep,name=artist_track_id,json=artistTrackId,proto3" json:"artist_track_id,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Album) Reset()         { *m = Album{} }
func (m *Album) String() string { return proto.CompactTextString(m) }
func (*Album) ProtoMessage()    {}
func (*Album) Descriptor() ([]byte, []int) {
	return fileDescriptor_a83fef21c75be787, []int{3}
}

func (m *Album) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Album.Unmarshal(m, b)
}
func (m *Album) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Album.Marshal(b, m, deterministic)
}
func (m *Album) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Album.Merge(m, src)
}
func (m *Album) XXX_Size() int {
	return xxx_messageInfo_Album.Size(m)
}
func (m *Album) XXX_DiscardUnknown() {
	xxx_messageInfo_Album.DiscardUnknown(m)
}

var xxx_messageInfo_Album proto.InternalMessageInfo

func (m *Album) GetArtistId() string {
	if m != nil {
		return m.ArtistId
	}
	return ""
}

func (m *Album) GetArtistAlbumId() string {
	if m != nil {
		return m.ArtistAlbumId
	}
	return ""
}

func (m *Album) GetTitle() string {
	if m != nil {
		return m.Title
	}
	return ""
}

func (m *Album) GetArtistTrackId() []string {
	if m != nil {
		return m.ArtistTrackId
	}
	return nil
}

type Track struct {
	ArtistId             string   `protobuf:"bytes,1,opt,name=artist_id,json=artistId,proto3" json:"artist_id,omitempty"`
	ArtistAlbumId        string   `protobuf:"bytes,2,opt,name=artist_album_id,json=artistAlbumId,proto3" json:"artist_album_id,omitempty"`
	ArtistTrackId        string   `protobuf:"bytes,3,opt,name=artist_track_id,json=artistTrackId,proto3" json:"artist_track_id,omitempty"`
	AlbumTrackNumber     uint32   `protobuf:"varint,4,opt,name=album_track_number,json=albumTrackNumber,proto3" json:"album_track_number,omitempty"`
	Title                string   `protobuf:"bytes,5,opt,name=title,proto3" json:"title,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Track) Reset()         { *m = Track{} }
func (m *Track) String() string { return proto.CompactTextString(m) }
func (*Track) ProtoMessage()    {}
func (*Track) Descriptor() ([]byte, []int) {
	return fileDescriptor_a83fef21c75be787, []int{4}
}

func (m *Track) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Track.Unmarshal(m, b)
}
func (m *Track) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Track.Marshal(b, m, deterministic)
}
func (m *Track) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Track.Merge(m, src)
}
func (m *Track) XXX_Size() int {
	return xxx_messageInfo_Track.Size(m)
}
func (m *Track) XXX_DiscardUnknown() {
	xxx_messageInfo_Track.DiscardUnknown(m)
}

var xxx_messageInfo_Track proto.InternalMessageInfo

func (m *Track) GetArtistId() string {
	if m != nil {
		return m.ArtistId
	}
	return ""
}

func (m *Track) GetArtistAlbumId() string {
	if m != nil {
		return m.ArtistAlbumId
	}
	return ""
}

func (m *Track) GetArtistTrackId() string {
	if m != nil {
		return m.ArtistTrackId
	}
	return ""
}

func (m *Track) GetAlbumTrackNumber() uint32 {
	if m != nil {
		return m.AlbumTrackNumber
	}
	return 0
}

func (m *Track) GetTitle() string {
	if m != nil {
		return m.Title
	}
	return ""
}

type Peer struct {
	Pubkey               string   `protobuf:"bytes,1,opt,name=pubkey,proto3" json:"pubkey,omitempty"`
	Host                 string   `protobuf:"bytes,2,opt,name=host,proto3" json:"host,omitempty"`
	Port                 uint32   `protobuf:"varint,3,opt,name=port,proto3" json:"port,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Peer) Reset()         { *m = Peer{} }
func (m *Peer) String() string { return proto.CompactTextString(m) }
func (*Peer) ProtoMessage()    {}
func (*Peer) Descriptor() ([]byte, []int) {
	return fileDescriptor_a83fef21c75be787, []int{5}
}

func (m *Peer) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Peer.Unmarshal(m, b)
}
func (m *Peer) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Peer.Marshal(b, m, deterministic)
}
func (m *Peer) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Peer.Merge(m, src)
}
func (m *Peer) XXX_Size() int {
	return xxx_messageInfo_Peer.Size(m)
}
func (m *Peer) XXX_DiscardUnknown() {
	xxx_messageInfo_Peer.DiscardUnknown(m)
}

var xxx_messageInfo_Peer proto.InternalMessageInfo

func (m *Peer) GetPubkey() string {
	if m != nil {
		return m.Pubkey
	}
	return ""
}

func (m *Peer) GetHost() string {
	if m != nil {
		return m.Host
	}
	return ""
}

func (m *Peer) GetPort() uint32 {
	if m != nil {
		return m.Port
	}
	return 0
}

func init() {
	proto.RegisterType((*ArtRequest)(nil), "net.audiostrike.art.ArtRequest")
	proto.RegisterType((*ArtReply)(nil), "net.audiostrike.art.ArtReply")
	proto.RegisterType((*Artist)(nil), "net.audiostrike.art.Artist")
	proto.RegisterType((*Album)(nil), "net.audiostrike.art.Album")
	proto.RegisterType((*Track)(nil), "net.audiostrike.art.Track")
	proto.RegisterType((*Peer)(nil), "net.audiostrike.art.Peer")
}

func init() { proto.RegisterFile("pkg/art/art.proto", fileDescriptor_a83fef21c75be787) }

var fileDescriptor_a83fef21c75be787 = []byte{
	// 407 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xac, 0x53, 0xcd, 0x8e, 0xd3, 0x30,
	0x10, 0xc6, 0xcd, 0x0f, 0xed, 0xa0, 0x0a, 0x30, 0x08, 0x85, 0x56, 0x88, 0x28, 0x07, 0x94, 0x03,
	0x4a, 0xa5, 0x22, 0x1e, 0xa0, 0x17, 0x50, 0x39, 0x20, 0x6a, 0x71, 0x47, 0x49, 0x63, 0x95, 0x28,
	0x69, 0x92, 0xb5, 0x27, 0x87, 0xbe, 0xc2, 0x3e, 0xd1, 0x3e, 0xcb, 0x3e, 0xcd, 0xca, 0xe3, 0x54,
	0x5b, 0xed, 0xa6, 0xd5, 0x1e, 0xf6, 0x10, 0x69, 0x66, 0xf2, 0x7d, 0x9e, 0x6f, 0xe6, 0xb3, 0xe1,
	0x6d, 0x5b, 0xee, 0x16, 0xa9, 0x42, 0xf3, 0x25, 0xad, 0x6a, 0xb0, 0xe1, 0xef, 0x6a, 0x89, 0x49,
	0xda, 0xe5, 0x45, 0xa3, 0x51, 0x15, 0xa5, 0x4c, 0x52, 0x85, 0xd1, 0x0e, 0x60, 0xa5, 0x50, 0xc8,
	0xab, 0x4e, 0x6a, 0xe4, 0x73, 0x98, 0xa4, 0x0a, 0x0b, 0x8d, 0xff, 0x8a, 0x3c, 0x60, 0x21, 0x8b,
	0x27, 0x62, 0x6c, 0x0b, 0xeb, 0x9c, 0x7f, 0x81, 0xd7, 0xfd, 0x4f, 0x54, 0xe9, 0xb6, 0x34, 0x90,
	0x11, 0x41, 0xa6, 0xb6, 0xfc, 0xd7, 0x54, 0xd7, 0x39, 0x7f, 0x0f, 0x9e, 0x2e, 0xea, 0xad, 0x0c,
	0x9c, 0x90, 0xc5, 0xae, 0xb0, 0x49, 0x74, 0xcb, 0x60, 0x4c, 0x9d, 0xda, 0xea, 0xc0, 0xbf, 0xc3,
	0x4b, 0xcb, 0xd1, 0x01, 0x0b, 0x9d, 0xf8, 0xd5, 0x72, 0x9e, 0x0c, 0x88, 0x4b, 0x56, 0x84, 0x11,
	0x47, 0x2c, 0x5f, 0x82, 0x9f, 0x56, 0x59, 0xb7, 0xd7, 0xc1, 0x88, 0x58, 0xb3, 0x61, 0x96, 0x81,
	0x88, 0x1e, 0x69, 0x38, 0x24, 0x57, 0x07, 0xce, 0x05, 0x0e, 0x69, 0x17, 0x3d, 0x92, 0x2f, 0xc0,
	0x6b, 0xa5, 0x54, 0x3a, 0x70, 0x89, 0xf2, 0x71, 0x90, 0xf2, 0x47, 0x4a, 0x25, 0x2c, 0x2e, 0xda,
	0x80, 0x6f, 0xb5, 0x5e, 0xde, 0x20, 0x07, 0xb7, 0x4e, 0xf7, 0xb2, 0x5f, 0x1b, 0xc5, 0xfc, 0x03,
	0xf8, 0x6d, 0x97, 0x95, 0xf2, 0x40, 0xeb, 0x9a, 0x88, 0x3e, 0x8b, 0xae, 0x19, 0x78, 0x34, 0xc9,
	0x53, 0x4d, 0xa1, 0x79, 0x1f, 0x99, 0x42, 0x47, 0x58, 0x53, 0xb0, 0xc0, 0x4a, 0xf6, 0x5d, 0x6c,
	0x32, 0x64, 0xa9, 0x19, 0xf9, 0xa1, 0xa5, 0xd1, 0x0d, 0x03, 0x8f, 0xe2, 0xe7, 0x11, 0x33, 0xd0,
	0xd6, 0x19, 0xba, 0x49, 0x5f, 0x81, 0xdb, 0x83, 0x2c, 0xac, 0xee, 0xf6, 0x99, 0x54, 0x81, 0x1b,
	0xb2, 0x78, 0x2a, 0xde, 0xd0, 0x1f, 0x42, 0xfe, 0xa6, 0xfa, 0xfd, 0x88, 0xde, 0xc9, 0x88, 0xd1,
	0x0f, 0x70, 0x8d, 0x53, 0x27, 0x7b, 0x66, 0xa7, 0x7b, 0x36, 0x9e, 0xfc, 0x6f, 0x34, 0x1e, 0x3d,
	0x31, 0xb1, 0xa9, 0xb5, 0x8d, 0x42, 0x12, 0x35, 0x15, 0x14, 0x2f, 0x37, 0xe0, 0xac, 0x14, 0xf2,
	0x5f, 0xe0, 0xff, 0x94, 0x68, 0xa2, 0xcf, 0xe7, 0xae, 0x6c, 0xff, 0x98, 0x66, 0x9f, 0xce, 0x03,
	0xda, 0xea, 0x10, 0xbd, 0xc8, 0x7c, 0x7a, 0x97, 0xdf, 0xee, 0x02, 0x00, 0x00, 0xff, 0xff, 0x1b,
	0x2d, 0x1c, 0x56, 0xac, 0x03, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// ArtClient is the client API for Art service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type ArtClient interface {
	GetArt(ctx context.Context, in *ArtRequest, opts ...grpc.CallOption) (*ArtReply, error)
}

type artClient struct {
	cc *grpc.ClientConn
}

func NewArtClient(cc *grpc.ClientConn) ArtClient {
	return &artClient{cc}
}

func (c *artClient) GetArt(ctx context.Context, in *ArtRequest, opts ...grpc.CallOption) (*ArtReply, error) {
	out := new(ArtReply)
	err := c.cc.Invoke(ctx, "/net.audiostrike.art.Art/GetArt", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ArtServer is the server API for Art service.
type ArtServer interface {
	GetArt(context.Context, *ArtRequest) (*ArtReply, error)
}

// UnimplementedArtServer can be embedded to have forward compatible implementations.
type UnimplementedArtServer struct {
}

func (*UnimplementedArtServer) GetArt(ctx context.Context, req *ArtRequest) (*ArtReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetArt not implemented")
}

func RegisterArtServer(s *grpc.Server, srv ArtServer) {
	s.RegisterService(&_Art_serviceDesc, srv)
}

func _Art_GetArt_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ArtRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ArtServer).GetArt(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/net.audiostrike.art.Art/GetArt",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ArtServer).GetArt(ctx, req.(*ArtRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Art_serviceDesc = grpc.ServiceDesc{
	ServiceName: "net.audiostrike.art.Art",
	HandlerType: (*ArtServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetArt",
			Handler:    _Art_GetArt_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "pkg/art/art.proto",
}