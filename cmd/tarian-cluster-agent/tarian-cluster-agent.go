package main

import (
	"fmt"
	"log"
	"net"

	"github.com/devopstoday11/tarian/pkg/clusteragent"
	"github.com/devopstoday11/tarian/pkg/tarianpb"
	"google.golang.org/grpc"
)

const (
	port = ":50052"
)

func main() {
	fmt.Println("tarian-cluster-agent")

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	clusterAgentServer := clusteragent.NewServer("localhost:50051")
	defer clusterAgentServer.Close()

	s := grpc.NewServer()

	tarianpb.RegisterConfigServer(s, clusterAgentServer)
	log.Printf("server listening at %v", lis.Addr())

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
