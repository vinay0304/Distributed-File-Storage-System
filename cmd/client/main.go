package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"sync"
	"time"

	pb "github.com/vinay/distributed-kv/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	addr := flag.String("addr", "localhost:50051", "Node address to connect to")
	op := flag.String("op", "bench", "Operation: put, get, bench")
	key := flag.String("key", "", "Key")
	val := flag.String("val", "", "Value")
	concurrency := flag.Int("c", 10, "Concurrency for benchmark")
	requests := flag.Int("n", 100, "Number of requests per worker")
	flag.Parse()

	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewKeyValueStoreClient(conn)

	ctx := context.Background()

	switch *op {
	case "put":
		res, err := client.Put(ctx, &pb.PutRequest{Key: *key, Value: *val, PrimaryOperation: true})
		if err != nil {
			log.Fatalf("Put failed: %v", err)
		}
		fmt.Printf("Put result: %v\n", res.Success)
	case "get":
		res, err := client.Get(ctx, &pb.GetRequest{Key: *key})
		if err != nil {
			log.Fatalf("Get failed: %v", err)
		}
		if res.Found {
			fmt.Printf("Value: %s\n", res.Value)
		} else {
			fmt.Println("Key not found")
		}
	case "bench":
		benchmark(client, *concurrency, *requests)
	default:
		log.Fatal("Unknown operation")
	}
}

func benchmark(client pb.KeyValueStoreClient, concurrency, requests int) {
	fmt.Printf("Starting benchmark: %d workers, %d requests each\n", concurrency, requests)
	
	start := time.Now()
	var wg sync.WaitGroup
	wg.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < requests; j++ {
				key := fmt.Sprintf("key-%d-%d", workerID, j)
				val := fmt.Sprintf("val-%d", j)
				
				_, err := client.Put(context.Background(), &pb.PutRequest{
					Key:              key,
					Value:            val,
					PrimaryOperation: true,
				})
				if err != nil {
					log.Printf("Worker %d failed: %v", workerID, err)
					return
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)
	total := concurrency * requests
	
	fmt.Printf("\n--- Benchmark Results ---\n")
	fmt.Printf("Total Requests: %d\n", total)
	fmt.Printf("Total Time:     %v\n", duration)
	fmt.Printf("Requests/sec:   %.2f\n", float64(total)/duration.Seconds())
	fmt.Printf("Avg Latency:    %v\n", duration/time.Duration(total))
}
