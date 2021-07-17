package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/devopstoday11/tarian/pkg/tarianpb"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
)

var grpcConn *grpc.ClientConn

func main() {
	fmt.Println("tarian-cluster-agent")

	var err error

	grpcConn, err = grpc.Dial("localhost:50052", grpc.WithInsecure(), grpc.WithBlock())

	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer grpcConn.Close()

	c := tarianpb.NewConfigClient(grpcConn)

	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)

		r, err := c.GetConfig(ctx, &empty.Empty{})

		if err != nil {
			log.Fatalf("error while getting config: %v", err)
		}

		fmt.Println(r)
		cancel()

		time.Sleep(3 * time.Second)
	}
}
