package main

import (
	"fmt"
	"log"
	"net"

	"github.com/devopstoday11/tarian/pkg/server"
	"github.com/devopstoday11/tarian/pkg/tarianpb"
	"google.golang.org/grpc"
)

const (
	port = ":50051"
)

func main() {
	fmt.Println("tarian-server")

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	tarianpb.RegisterConfigServer(s, server.NewServer())

	log.Printf("server listening at %v", lis.Addr())

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
