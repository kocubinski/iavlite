package v2

import "github.com/gogo/protobuf/proto"

var (
	_ proto.Message = (*KVPair)(nil)
	_ proto.Message = (*ChangeSet)(nil)
)

type KVPair struct {
	Delete bool   `protobuf:"varint,1,opt,name=delete,proto3" json:"delete,omitempty"`
	Key    []byte `protobuf:"bytes,2,opt,name=key,proto3" json:"key,omitempty"`
	Value  []byte `protobuf:"bytes,3,opt,name=value,proto3" json:"value,omitempty"`
}

func (K *KVPair) Reset() { *K = KVPair{} }

func (K *KVPair) String() string { return "" }

func (K *KVPair) ProtoMessage() {}

type ChangeSet struct {
	Pairs []*KVPair `protobuf:"bytes,1,rep,name=pairs,proto3" json:"pairs,omitempty"`
}

func (c *ChangeSet) Reset() {
	*c = ChangeSet{}
}

func (c *ChangeSet) String() string {
	return ""
}

func (c *ChangeSet) ProtoMessage() {
}
