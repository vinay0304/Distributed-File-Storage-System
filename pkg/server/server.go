package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/vinay/distributed-kv/pkg/hashring"
	"github.com/vinay/distributed-kv/pkg/storage"
	pb "github.com/vinay/distributed-kv/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Node struct {
	pb.UnimplementedKeyValueStoreServer
	ID         string
	Addr       string
	Ring       *hashring.HashRing
	Store      *storage.MemoryStore
	Nodes      map[string]string // ID -> Addr
	ReplFactor int
	
	clients   map[string]pb.KeyValueStoreClient
	clientsMu sync.RWMutex
}

func NewNode(id string, addr string, nodes map[string]string, replFactor int) *Node {
	ring := hashring.New(100)
	for rid := range nodes {
		ring.AddNode(rid)
	}

	return &Node{
		ID:         id,
		Addr:       addr,
		Ring:       ring,
		Store:      storage.NewMemoryStore(),
		Nodes:      nodes,
		ReplFactor: replFactor,
		clients:    make(map[string]pb.KeyValueStoreClient),
	}
}

func (n *Node) getClient(nodeID string) (pb.KeyValueStoreClient, error) {
	n.clientsMu.RLock()
	client, ok := n.clients[nodeID]
	n.clientsMu.RUnlock()
	if ok {
		return client, nil
	}

	n.clientsMu.Lock()
	defer n.clientsMu.Unlock()

	// Double check
	if client, ok := n.clients[nodeID]; ok {
		return client, nil
	}

	addr, ok := n.Nodes[nodeID]
	if !ok {
		return nil, fmt.Errorf("node %s not found in configuration", nodeID)
	}

	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	client = pb.NewKeyValueStoreClient(conn)
	n.clients[nodeID] = client
	return client, nil
}

func (n *Node) Put(ctx context.Context, req *pb.PutRequest) (*pb.PutResponse, error) {
	primary := n.Ring.GetPrimary(req.Key)
	
	if primary != n.ID {
		// Forward to primary node
		client, err := n.getClient(primary)
		if err != nil {
			return nil, err
		}
		return client.Put(ctx, req)
	}

	// I am the primary. Handle replication.
	replicas := n.Ring.GetReplicas(req.Key, n.ReplFactor)
	
	var wg sync.WaitGroup
	errChan := make(chan error, len(replicas))

	for _, replicaID := range replicas {
		if replicaID == n.ID {
			n.Store.Put(req.Key, req.Value)
			continue
		}

		wg.Add(1)
		go func(rid string) {
			defer wg.Done()
			client, err := n.getClient(rid)
			if err != nil {
				errChan <- err
				return
			}
			_, err = client.InternalPut(ctx, &pb.PutRequest{
				Key:   req.Key,
				Value: req.Value,
				PrimaryOperation: false,
			})
			if err != nil {
				errChan <- err
			}
		}(replicaID)
	}

	wg.Wait()
	close(errChan)

	if len(errChan) > 0 {
		return &pb.PutResponse{Success: false, Message: "Replication failed"}, nil
	}

	return &pb.PutResponse{Success: true}, nil
}

func (n *Node) InternalPut(ctx context.Context, req *pb.PutRequest) (*pb.PutResponse, error) {
	n.Store.Put(req.Key, req.Value)
	return &pb.PutResponse{Success: true}, nil
}

func (n *Node) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	val, ok := n.Store.Get(req.Key)
	if ok {
		return &pb.GetResponse{Value: val, Found: true}, nil
	}

	// In a real system, we might query replicas if not found here (eventual consistency)
	// or only query the primary/replicas specifically.
	// For simplicity, let's just check if we are a replica.
	replicas := n.Ring.GetReplicas(req.Key, n.ReplFactor)
	isReplica := false
	for _, rid := range replicas {
		if rid == n.ID {
			isReplica = true
			break
		}
	}

	if !isReplica {
		// Forward to a node that should have it
		client, err := n.getClient(replicas[0])
		if err != nil {
			return nil, err
		}
		return client.Get(ctx, req)
	}

	return &pb.GetResponse{Found: false}, nil
}

func (n *Node) Delete(ctx context.Context, req *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	primary := n.Ring.GetPrimary(req.Key)
	
	if primary != n.ID {
		client, err := n.getClient(primary)
		if err != nil {
			return nil, err
		}
		return client.Delete(ctx, req)
	}

	replicas := n.Ring.GetReplicas(req.Key, n.ReplFactor)
	
	var wg sync.WaitGroup
	for _, replicaID := range replicas {
		if replicaID == n.ID {
			n.Store.Delete(req.Key)
			continue
		}

		wg.Add(1)
		go func(rid string) {
			defer wg.Done()
			client, err := n.getClient(rid)
			if err == nil {
				client.InternalDelete(ctx, &pb.DeleteRequest{Key: req.Key, PrimaryOperation: false})
			}
		}(replicaID)
	}

	wg.Wait()
	return &pb.DeleteResponse{Success: true}, nil
}

func (n *Node) InternalDelete(ctx context.Context, req *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	n.Store.Delete(req.Key)
	return &pb.DeleteResponse{Success: true}, nil
}

func (n *Node) Start() error {
	lis, err := net.Listen("tcp", n.Addr)
	if err != nil {
		return err
	}

	s := grpc.NewServer()
	pb.RegisterKeyValueStoreServer(s, n)

	log.Printf("Node %s listening on %s", n.ID, n.Addr)
	return s.Serve(lis)
}
