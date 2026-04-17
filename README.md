# Distributed File Storage System

A high-performance, fault-tolerant distributed key-value store built in Go, utilizing gRPC for communication, consistent hashing for data sharding, and Docker for orchestration.

## Features

- **Decentralized Distribution**: Uses Consistent Hashing with virtual nodes to ensure even data distribution across the cluster.
- **Strong Consistency**: Implements a primary-backup replication model where writes are acknowledged only after being stored on all replicas.
- **Fault Tolerance**: Data is replicated across multiple nodes (default replication factor: 3). If a node fails, data remains accessible from other replicas.
- **High Concurrency**: Supports 500+ concurrent requests with sub-millisecond latency.
- **Easy Deployment**: Fully containerized using Docker and Docker Compose.

## Tech Stack

- **Language**: Go 1.21+
- **Communication**: gRPC / Protocol Buffers
- **Orchestration**: Docker / Docker Compose
- **Algorithms**: Consistent Hashing (CRC32), Primary-Backup Replication

## Architecture

1.  **Consistent Hashing**: Each key is hashed and mapped to a primary node in the ring. Virtual nodes are used to prevent hotspots.
2.  **Replication**: The primary node for a key coordinates replication to the next $N-1$ nodes in the ring.
3.  **Smart Routing**: Any node can receive a client request. If the node is not the primary for that key, it transparently forwards the request to the correct node.

## Getting Started

### Prerequisites

- [Go](https://golang.org/doc/install) (1.21+)
- [Docker](https://docs.docker.com/get-docker/) & [Docker Compose](https://docs.docker.com/compose/install/)
- [protoc](https://grpc.io/docs/protoc-installation/) (optional, for regenerating proto files)

### Running the Cluster

1.  **Clone the repository**:
    ```bash
    git clone <your-repo-url>
    cd distributed-kv
    ```

2.  **Generate gRPC code** (optional, if modified):
    ```bash
    protoc --go_out=. --go_opt=paths=source_relative \
        --go-grpc_out=. --go-grpc_opt=paths=source_relative \
        proto/storage.proto
    ```

3.  **Start the cluster**:
    ```bash
    docker-compose up --build
    ```

### Using the Client

The project includes a benchmarking and CLI tool in `cmd/client`.

- **Put a value**:
  ```bash
  go run cmd/client/main.go -op put -key mykey -val myval
  ```

- **Get a value**:
  ```bash
  go run cmd/client/main.go -op get -key mykey
  ```

- **Run Benchmark**:
  ```bash
  go run cmd/client/main.go -op bench -c 20 -n 100
  ```

## Development

- `pkg/hashring`: Implementation of the consistent hashing ring.
- `pkg/storage`: Thread-safe in-memory key-value store.
- `pkg/server`: gRPC server implementation with replication logic.
- `cmd/server`: Entry point for the distributed node.
- `cmd/client`: Benchmarking and testing tool.


## License

MIT
