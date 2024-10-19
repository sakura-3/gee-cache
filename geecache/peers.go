package geecache

import "geecache/pb"

// 根据key选择所在的peer,返回其对应的getter
// 若不存在，返回nil
type PeerPicker interface {
	PickPeer(key string) PeerGetter
}

type PeerGetter interface {
	Get(in *pb.Request, out *pb.Response) error
}
