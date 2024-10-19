package geecache

import (
	"context"
	"fmt"
	"geecache/internal/hash"
	"geecache/pb"
	"log"
	"net"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	defaultReplicas = 50
	defaultIP       = "localhost"
)

var _ PeerGetter = (*grpcGetter)(nil)

type grpcGetter struct {
	addr string
}

func (g *grpcGetter) Get(in *pb.Request, out *pb.Response) error {
	conn, err := grpc.NewClient(g.addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	log.Printf("g.addr=%s", g.addr)

	client := pb.NewGroupCacheClient(conn)
	resp, err := client.Get(context.Background(), in)
	if err != nil {
		return err
	}
	out.Value = resp.Value
	return nil
}

var _ PeerPicker = (*GrpcPool)(nil)

type GrpcPool struct {
	pb.UnimplementedGroupCacheServer
	self string

	mu      sync.Mutex
	peers   *hash.Map
	getters map[string]*grpcGetter
}

func NewGrpcPool(self string) *GrpcPool {
	return &GrpcPool{
		self: self,
	}
}

func (p *GrpcPool) Log(format string, v ...any) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// 添加节点
func (p *GrpcPool) AddPeers(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.peers == nil {
		p.peers = hash.New(defaultReplicas, nil)
		p.getters = make(map[string]*grpcGetter)
	}

	p.peers.Add(peers...)
	for _, peer := range peers {
		p.getters[peer] = &grpcGetter{addr: fmt.Sprintf("%s:%s", defaultIP, peer)}
	}
}

func (p *GrpcPool) PickPeer(key string) PeerGetter {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 需要确保选择的peer不是自己
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick Peer %s", peer)
		return p.getters[peer]
	}

	return nil
}

func (p *GrpcPool) Get(ctx context.Context, in *pb.Request) (*pb.Response, error) {
	p.Log("%s %s", in.Group, in.Key)

	group := GetGroup(in.Group)
	if group == nil {
		return nil, fmt.Errorf("group %s inexistent", in.Group)
	}

	value, err := group.Get(in.Key)
	if err != nil {
		return nil, err
	}

	return &pb.Response{Value: value}, nil
}

func (p *GrpcPool) Run() {
	lis, err := net.Listen("tcp", ":"+p.self)
	if err != nil {
		panic(err)
	}

	server := grpc.NewServer()
	pb.RegisterGroupCacheServer(server, p)

	// reflection.Register(server)
	err = server.Serve(lis)
	if err != nil {
		panic(err)
	}
}
