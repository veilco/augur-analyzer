// Code generated by protoc-gen-go. DO NOT EDIT.
// source: markets.proto

package markets

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type MarketType int32

const (
	MarketType_YESNO       MarketType = 0
	MarketType_CATEGORICAL MarketType = 1
	MarketType_SCALAR      MarketType = 2
)

var MarketType_name = map[int32]string{
	0: "YESNO",
	1: "CATEGORICAL",
	2: "SCALAR",
}
var MarketType_value = map[string]int32{
	"YESNO":       0,
	"CATEGORICAL": 1,
	"SCALAR":      2,
}

func (x MarketType) String() string {
	return proto.EnumName(MarketType_name, int32(x))
}
func (MarketType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_markets_b3b4de6338a2b5d3, []int{0}
}

type MarketsSummary struct {
	Block                      uint64    `protobuf:"varint,1,opt,name=block,proto3" json:"block,omitempty"`
	TotalMarkets               uint64    `protobuf:"varint,2,opt,name=total_markets,json=totalMarkets,proto3" json:"total_markets,omitempty"`
	TotalMarketsCapitalization *Price    `protobuf:"bytes,3,opt,name=total_markets_capitalization,json=totalMarketsCapitalization,proto3" json:"total_markets_capitalization,omitempty"`
	Markets                    []*Market `protobuf:"bytes,4,rep,name=markets,proto3" json:"markets,omitempty"`
	XXX_NoUnkeyedLiteral       struct{}  `json:"-"`
	XXX_unrecognized           []byte    `json:"-"`
	XXX_sizecache              int32     `json:"-"`
}

func (m *MarketsSummary) Reset()         { *m = MarketsSummary{} }
func (m *MarketsSummary) String() string { return proto.CompactTextString(m) }
func (*MarketsSummary) ProtoMessage()    {}
func (*MarketsSummary) Descriptor() ([]byte, []int) {
	return fileDescriptor_markets_b3b4de6338a2b5d3, []int{0}
}
func (m *MarketsSummary) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_MarketsSummary.Unmarshal(m, b)
}
func (m *MarketsSummary) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_MarketsSummary.Marshal(b, m, deterministic)
}
func (dst *MarketsSummary) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MarketsSummary.Merge(dst, src)
}
func (m *MarketsSummary) XXX_Size() int {
	return xxx_messageInfo_MarketsSummary.Size(m)
}
func (m *MarketsSummary) XXX_DiscardUnknown() {
	xxx_messageInfo_MarketsSummary.DiscardUnknown(m)
}

var xxx_messageInfo_MarketsSummary proto.InternalMessageInfo

func (m *MarketsSummary) GetBlock() uint64 {
	if m != nil {
		return m.Block
	}
	return 0
}

func (m *MarketsSummary) GetTotalMarkets() uint64 {
	if m != nil {
		return m.TotalMarkets
	}
	return 0
}

func (m *MarketsSummary) GetTotalMarketsCapitalization() *Price {
	if m != nil {
		return m.TotalMarketsCapitalization
	}
	return nil
}

func (m *MarketsSummary) GetMarkets() []*Market {
	if m != nil {
		return m.Markets
	}
	return nil
}

type Price struct {
	Eth                  float32  `protobuf:"fixed32,1,opt,name=eth,proto3" json:"eth,omitempty"`
	Usd                  float32  `protobuf:"fixed32,2,opt,name=usd,proto3" json:"usd,omitempty"`
	Btc                  float32  `protobuf:"fixed32,3,opt,name=btc,proto3" json:"btc,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Price) Reset()         { *m = Price{} }
func (m *Price) String() string { return proto.CompactTextString(m) }
func (*Price) ProtoMessage()    {}
func (*Price) Descriptor() ([]byte, []int) {
	return fileDescriptor_markets_b3b4de6338a2b5d3, []int{1}
}
func (m *Price) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Price.Unmarshal(m, b)
}
func (m *Price) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Price.Marshal(b, m, deterministic)
}
func (dst *Price) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Price.Merge(dst, src)
}
func (m *Price) XXX_Size() int {
	return xxx_messageInfo_Price.Size(m)
}
func (m *Price) XXX_DiscardUnknown() {
	xxx_messageInfo_Price.DiscardUnknown(m)
}

var xxx_messageInfo_Price proto.InternalMessageInfo

func (m *Price) GetEth() float32 {
	if m != nil {
		return m.Eth
	}
	return 0
}

func (m *Price) GetUsd() float32 {
	if m != nil {
		return m.Usd
	}
	return 0
}

func (m *Price) GetBtc() float32 {
	if m != nil {
		return m.Btc
	}
	return 0
}

type Market struct {
	Id                   string        `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	MarketType           MarketType    `protobuf:"varint,2,opt,name=market_type,json=marketType,proto3,enum=markets.MarketType" json:"market_type,omitempty"`
	Name                 string        `protobuf:"bytes,3,opt,name=name,proto3" json:"name,omitempty"`
	CommentCount         uint32        `protobuf:"varint,4,opt,name=comment_count,json=commentCount,proto3" json:"comment_count,omitempty"`
	MarketCapitalization *Price        `protobuf:"bytes,5,opt,name=market_capitalization,json=marketCapitalization,proto3" json:"market_capitalization,omitempty"`
	EndDate              uint64        `protobuf:"varint,6,opt,name=end_date,json=endDate,proto3" json:"end_date,omitempty"`
	Predictions          []*Prediction `protobuf:"bytes,7,rep,name=predictions,proto3" json:"predictions,omitempty"`
	XXX_NoUnkeyedLiteral struct{}      `json:"-"`
	XXX_unrecognized     []byte        `json:"-"`
	XXX_sizecache        int32         `json:"-"`
}

func (m *Market) Reset()         { *m = Market{} }
func (m *Market) String() string { return proto.CompactTextString(m) }
func (*Market) ProtoMessage()    {}
func (*Market) Descriptor() ([]byte, []int) {
	return fileDescriptor_markets_b3b4de6338a2b5d3, []int{2}
}
func (m *Market) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Market.Unmarshal(m, b)
}
func (m *Market) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Market.Marshal(b, m, deterministic)
}
func (dst *Market) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Market.Merge(dst, src)
}
func (m *Market) XXX_Size() int {
	return xxx_messageInfo_Market.Size(m)
}
func (m *Market) XXX_DiscardUnknown() {
	xxx_messageInfo_Market.DiscardUnknown(m)
}

var xxx_messageInfo_Market proto.InternalMessageInfo

func (m *Market) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

func (m *Market) GetMarketType() MarketType {
	if m != nil {
		return m.MarketType
	}
	return MarketType_YESNO
}

func (m *Market) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *Market) GetCommentCount() uint32 {
	if m != nil {
		return m.CommentCount
	}
	return 0
}

func (m *Market) GetMarketCapitalization() *Price {
	if m != nil {
		return m.MarketCapitalization
	}
	return nil
}

func (m *Market) GetEndDate() uint64 {
	if m != nil {
		return m.EndDate
	}
	return 0
}

func (m *Market) GetPredictions() []*Prediction {
	if m != nil {
		return m.Predictions
	}
	return nil
}

type Prediction struct {
	Name                 string   `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Percent              float32  `protobuf:"fixed32,2,opt,name=percent,proto3" json:"percent,omitempty"`
	Value                float32  `protobuf:"fixed32,3,opt,name=value,proto3" json:"value,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Prediction) Reset()         { *m = Prediction{} }
func (m *Prediction) String() string { return proto.CompactTextString(m) }
func (*Prediction) ProtoMessage()    {}
func (*Prediction) Descriptor() ([]byte, []int) {
	return fileDescriptor_markets_b3b4de6338a2b5d3, []int{3}
}
func (m *Prediction) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Prediction.Unmarshal(m, b)
}
func (m *Prediction) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Prediction.Marshal(b, m, deterministic)
}
func (dst *Prediction) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Prediction.Merge(dst, src)
}
func (m *Prediction) XXX_Size() int {
	return xxx_messageInfo_Prediction.Size(m)
}
func (m *Prediction) XXX_DiscardUnknown() {
	xxx_messageInfo_Prediction.DiscardUnknown(m)
}

var xxx_messageInfo_Prediction proto.InternalMessageInfo

func (m *Prediction) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *Prediction) GetPercent() float32 {
	if m != nil {
		return m.Percent
	}
	return 0
}

func (m *Prediction) GetValue() float32 {
	if m != nil {
		return m.Value
	}
	return 0
}

func init() {
	proto.RegisterType((*MarketsSummary)(nil), "markets.MarketsSummary")
	proto.RegisterType((*Price)(nil), "markets.Price")
	proto.RegisterType((*Market)(nil), "markets.Market")
	proto.RegisterType((*Prediction)(nil), "markets.Prediction")
	proto.RegisterEnum("markets.MarketType", MarketType_name, MarketType_value)
}

func init() { proto.RegisterFile("markets.proto", fileDescriptor_markets_b3b4de6338a2b5d3) }

var fileDescriptor_markets_b3b4de6338a2b5d3 = []byte{
	// 415 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x6c, 0x52, 0xcb, 0x8e, 0xd3, 0x30,
	0x14, 0x25, 0x6e, 0x1e, 0xf4, 0x66, 0x9a, 0x89, 0x3c, 0x83, 0x64, 0x10, 0x8b, 0xa8, 0x6c, 0x02,
	0x8b, 0x59, 0x94, 0x61, 0xc9, 0x22, 0x0a, 0x23, 0x84, 0x34, 0x30, 0x95, 0xdb, 0x0d, 0xab, 0xc8,
	0x75, 0x2c, 0x11, 0x35, 0x2f, 0xa5, 0x0e, 0x52, 0xf9, 0x18, 0xbe, 0x87, 0xcf, 0x42, 0x76, 0x1e,
	0x4d, 0xab, 0xd9, 0x9d, 0xfb, 0x3a, 0xf7, 0xdc, 0x63, 0xc3, 0xa2, 0x60, 0xcd, 0x5e, 0xc8, 0xc3,
	0x5d, 0xdd, 0x54, 0xb2, 0xc2, 0x4e, 0x1f, 0x2e, 0xff, 0x19, 0xe0, 0x7d, 0xef, 0xf0, 0xa6, 0x2d,
	0x0a, 0xd6, 0x1c, 0xf1, 0x2d, 0x58, 0xbb, 0xbc, 0xe2, 0x7b, 0x62, 0x04, 0x46, 0x68, 0xd2, 0x2e,
	0xc0, 0xef, 0x60, 0x21, 0x2b, 0xc9, 0xf2, 0xa4, 0x9f, 0x24, 0x48, 0x57, 0xaf, 0x74, 0xb2, 0x67,
	0xc0, 0x6b, 0x78, 0x7b, 0xd6, 0x94, 0x70, 0x56, 0x67, 0x92, 0xe5, 0xd9, 0x1f, 0x26, 0xb3, 0xaa,
	0x24, 0xb3, 0xc0, 0x08, 0xdd, 0x95, 0x77, 0x37, 0x88, 0x59, 0x37, 0x19, 0x17, 0xf4, 0xcd, 0x94,
	0x23, 0x3e, 0x9b, 0xc0, 0xef, 0x61, 0x90, 0x4a, 0xcc, 0x60, 0x16, 0xba, 0xab, 0xeb, 0x71, 0xb8,
	0x1b, 0xa0, 0xe3, 0x29, 0x9f, 0xc1, 0xd2, 0x7c, 0xd8, 0x87, 0x99, 0x90, 0xbf, 0xb4, 0x7c, 0x44,
	0x15, 0x54, 0x99, 0xf6, 0x90, 0x6a, 0xc9, 0x88, 0x2a, 0xa8, 0x32, 0x3b, 0xc9, 0xb5, 0x20, 0x44,
	0x15, 0x5c, 0xfe, 0x45, 0x60, 0x77, 0x94, 0xd8, 0x03, 0x94, 0xa5, 0x7a, 0x7e, 0x4e, 0x51, 0x96,
	0xe2, 0x7b, 0x70, 0xbb, 0x25, 0x89, 0x3c, 0xd6, 0x42, 0xd3, 0x78, 0xab, 0x9b, 0x0b, 0x21, 0xdb,
	0x63, 0x2d, 0x28, 0x14, 0x23, 0xc6, 0x18, 0xcc, 0x92, 0x15, 0x42, 0xef, 0x98, 0x53, 0x8d, 0x95,
	0x8b, 0xbc, 0x2a, 0x0a, 0x51, 0xca, 0x84, 0x57, 0x6d, 0x29, 0x89, 0x19, 0x18, 0xe1, 0x82, 0x5e,
	0xf5, 0xc9, 0x58, 0xe5, 0x70, 0x0c, 0xaf, 0xfa, 0x75, 0x17, 0xf6, 0x59, 0xcf, 0xda, 0x77, 0xdb,
	0x85, 0x17, 0xc6, 0xbd, 0x86, 0x97, 0xa2, 0x4c, 0x93, 0x94, 0x49, 0x41, 0x6c, 0xfd, 0x54, 0x8e,
	0x28, 0xd3, 0x2f, 0x4c, 0x0a, 0xfc, 0x09, 0xdc, 0xba, 0x11, 0x69, 0xc6, 0x55, 0xe3, 0x81, 0x38,
	0xda, 0xd7, 0x9b, 0x09, 0xeb, 0x50, 0xa3, 0xd3, 0xbe, 0xe5, 0x1a, 0xe0, 0x54, 0x1a, 0xaf, 0x33,
	0x26, 0xd7, 0x11, 0x70, 0x6a, 0xd1, 0x70, 0x51, 0xca, 0xde, 0xea, 0x21, 0x54, 0x7f, 0xea, 0x37,
	0xcb, 0x5b, 0xd1, 0x1b, 0xde, 0x05, 0x1f, 0xee, 0x01, 0x4e, 0xde, 0xe1, 0x39, 0x58, 0x3f, 0x1f,
	0x36, 0x3f, 0x9e, 0xfc, 0x17, 0xf8, 0x1a, 0xdc, 0x38, 0xda, 0x3e, 0x7c, 0x7d, 0xa2, 0xdf, 0xe2,
	0xe8, 0xd1, 0x37, 0x30, 0x80, 0xbd, 0x89, 0xa3, 0xc7, 0x88, 0xfa, 0x68, 0x67, 0xeb, 0x2f, 0xfc,
	0xf1, 0x7f, 0x00, 0x00, 0x00, 0xff, 0xff, 0xa6, 0xbe, 0x33, 0x74, 0xd3, 0x02, 0x00, 0x00,
}
