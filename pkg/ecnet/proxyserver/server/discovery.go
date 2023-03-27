package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net"

	"google.golang.org/grpc"

	pb "github.com/flomesh-io/ErieCanal/pkg/ecnet/gen/proxy"
)

// grpcConfigServer is used to implement proxy.ConfigServer.
type grpcConfigServer struct {
	pb.UnimplementedConfigServer
}

// Poll implements proxy.ConfigServer..
func (s *grpcConfigServer) Poll(_ context.Context, in *pb.ConfigRequest) (*pb.ConfigResponse, error) {
	repoLock.RLock()
	bytes, _ := json.Marshal(latestConfig)
	defer func() {
		repoLock.RUnlock()
	}()
	return &pb.ConfigResponse{Json: string(bytes)}, nil
}

func (s *Server) DiscoveryListener(proxyServerPort uint32) {
	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", proxyServerPort))
	if err != nil {
		log.Fatal().Msgf("discoveryListener failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterConfigServer(grpcServer, &grpcConfigServer{})
	log.Log().Msgf("discoveryListener listening at %v", lis.Addr())
	if err = grpcServer.Serve(lis); err != nil {
		log.Fatal().Msgf("discoveryListener failed to serve: %v", err)
	}
}
