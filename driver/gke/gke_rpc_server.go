package gke

import (
	"net"

	generic "github.com/rancher/kontainer-engine/driver"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	gkeDriver *Driver
	address   string
}

func NewGkeRPCServer(gkeDriver Driver, addr string) *Server {
	return &Server{
		gkeDriver: &gkeDriver,
		address:   addr,
	}
}

func (s *Server) GetDriverCreateOptions(ctx context.Context, in *generic.Empty) (*generic.DriverFlags, error) {
	return s.gkeDriver.GetDriverCreateOptions()
}

func (s *Server) GetDriverUpdateOptions(ctx context.Context, in *generic.Empty) (*generic.DriverFlags, error) {
	return s.gkeDriver.GetDriverUpdateOptions()
}

func (s *Server) SetDriverOptions(ctx context.Context, in *generic.DriverOptions) (*generic.Empty, error) {
	return &generic.Empty{}, s.gkeDriver.SetDriverOptions(in)
}

func (s *Server) Create(ctx context.Context, in *generic.Empty) (*generic.Empty, error) {
	return &generic.Empty{}, s.gkeDriver.Create()
}

func (s *Server) Update(ctx context.Context, in *generic.Empty) (*generic.Empty, error) {
	return &generic.Empty{}, s.gkeDriver.Update()
}

func (s *Server) Get(cont context.Context, request *generic.ClusterGetRequest) (*generic.ClusterInfo, error) {
	return s.gkeDriver.Get(request)
}

func (s *Server) Remove(ctx context.Context, in *generic.Empty) (*generic.Empty, error) {
	return &generic.Empty{}, s.gkeDriver.Remove()
}

func (s *Server) Serve() {
	listen, err := net.Listen("tcp", s.address)
	if err != nil {
		logrus.Fatal(err)
	}
	grpcServer := grpc.NewServer()
	generic.RegisterDriverServer(grpcServer, s)
	reflection.Register(grpcServer)
	logrus.Debugf("RPC Server listening on address %s", s.address)
	if err := grpcServer.Serve(listen); err != nil {
		logrus.Fatal(err)
	}
	return
}
