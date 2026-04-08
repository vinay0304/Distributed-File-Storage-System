package hashring

import (
	"hash/crc32"
	"sort"
	"strconv"
	"sync"
)

// HashRing implements consistent hashing with virtual nodes.
type HashRing struct {
	virtualNodes int
	nodes        []uint32
	nodeMap      map[uint32]string
	sync.RWMutex
}

// New creates a new HashRing with a specific number of virtual nodes per physical node.
func New(virtualNodes int) *HashRing {
	return &HashRing{
		virtualNodes: virtualNodes,
		nodeMap:      make(map[uint32]string),
	}
}

// AddNode adds a physical node to the ring.
func (r *HashRing) AddNode(nodeID string) {
	r.Lock()
	defer r.Unlock()

	for i := 0; i < r.virtualNodes; i++ {
		hash := r.hash(nodeID + ":" + strconv.Itoa(i))
		r.nodes = append(r.nodes, hash)
		r.nodeMap[hash] = nodeID
	}
	sort.Slice(r.nodes, func(i, j int) bool {
		return r.nodes[i] < r.nodes[j]
	})
}

// RemoveNode removes a physical node from the ring.
func (r *HashRing) RemoveNode(nodeID string) {
	r.Lock()
	defer r.Unlock()

	newNodes := make([]uint32, 0)
	for _, hash := range r.nodes {
		if r.nodeMap[hash] != nodeID {
			newNodes = append(newNodes, hash)
		} else {
			delete(r.nodeMap, hash)
		}
	}
	r.nodes = newNodes
}

// GetPrimary returns the physical node ID responsible for the given key.
func (r *HashRing) GetPrimary(key string) string {
	r.RLock()
	defer r.RUnlock()

	if len(r.nodes) == 0 {
		return ""
	}

	hash := r.hash(key)
	idx := sort.Search(len(r.nodes), func(i int) bool {
		return r.nodes[i] >= hash
	})

	if idx == len(r.nodes) {
		idx = 0
	}

	return r.nodeMap[r.nodes[idx]]
}

// GetReplicas returns n unique replica nodes (including primary) for a key.
func (r *HashRing) GetReplicas(key string, n int) []string {
	r.RLock()
	defer r.RUnlock()

	if len(r.nodes) == 0 {
		return nil
	}

	hash := r.hash(key)
	idx := sort.Search(len(r.nodes), func(i int) bool {
		return r.nodes[i] >= hash
	})

	if idx == len(r.nodes) {
		idx = 0
	}

	replicas := make([]string, 0)
	seen := make(map[string]bool)

	// Traverse the ring to find n unique physical nodes
	for i := 0; i < len(r.nodes) && len(replicas) < n; i++ {
		currIdx := (idx + i) % len(r.nodes)
		nodeID := r.nodeMap[r.nodes[currIdx]]
		if !seen[nodeID] {
			replicas = append(replicas, nodeID)
			seen[nodeID] = true
		}
	}

	return replicas
}

func (r *HashRing) hash(key string) uint32 {
	return crc32.ChecksumIEEE([]byte(key))
}
