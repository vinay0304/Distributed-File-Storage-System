package main

import (
	"flag"
	"log"
	"os"
	"strings"

	"github.com/vinay/distributed-kv/pkg/server"
)

func main() {
	id := flag.String("id", "", "Unique ID for this node")
	addr := flag.String("addr", ":50051", "Address to listen on")
	cluster := flag.String("cluster", "", "Comma-separated list of nodeID:addr")
	replFactor := flag.Int("replica", 3, "Replication factor")
	flag.Parse()

	if *id == "" {
		*id = os.Getenv("NODE_ID")
	}
	if *cluster == "" {
		*cluster = os.Getenv("CLUSTER_NODES")
	}

	if *id == "" || *cluster == "" {
		log.Fatal("Node ID and Cluster nodes must be provided via flags or environment variables")
	}

	nodes := make(map[string]string)
	pairList := strings.Split(*cluster, ",")
	for _, pair := range pairList {
		parts := strings.Split(pair, "=")
		if len(parts) == 2 {
			nodes[parts[0]] = parts[1]
		} else {
			// Fallback if it's just id:addr
			idx := strings.Index(pair, ":")
			if idx > 0 {
				nodes[pair[:idx]] = pair[idx+1:]
			}
		}
	}

	node := server.NewNode(*id, *addr, nodes, *replFactor)
	if err := node.Start(); err != nil {
		log.Fatalf("Failed to start node: %v", err)
	}
}
