package gke

import (
	"net"

	generic "github.com/rancher/netes-machine/driver"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type server struct {
	gkeDriver *driver
	address   string
}

func NewGkeRpcServer(gkeDriver *driver, addr string) *server {
	return &server{
		gkeDriver: gkeDriver,
		address:   addr,
	}
}

func (s *server) GetDriverCreateOptions(ctx context.Context, in *generic.Empty) (*generic.DriverFlags, error) {
	return s.gkeDriver.GetDriverCreateOptions()
}

func (s *server) GetDriverUpdateOptions(ctx context.Context, in *generic.Empty) (*generic.DriverFlags, error) {
	return s.gkeDriver.GetDriverUpdateOptions()
}

func (s *server) SetDriverOptions(ctx context.Context, in *generic.DriverOptions) (*generic.Empty, error) {
	return &generic.Empty{}, s.gkeDriver.SetDriverOptions(in)
}

func (s *server) Create(ctx context.Context, in *generic.Empty) (*generic.Empty, error) {
	return &generic.Empty{}, s.gkeDriver.Create()
}

func (s *server) Update(ctx context.Context, in *generic.Empty) (*generic.Empty, error) {
	return &generic.Empty{}, s.gkeDriver.Update()
}

func (s *server) Get(cont context.Context, request *generic.ClusterGetRequest) (*generic.ClusterInfo, error) {
	return s.gkeDriver.Get(request)
}

func (s *server) Remove(ctx context.Context, in *generic.Empty) (*generic.Empty, error) {
	return &generic.Empty{}, s.gkeDriver.Remove()
}

func (s *server) Serve(errStop chan error) {
	listen, err := net.Listen("tcp", s.address)
	if err != nil {
		logrus.Fatal(err)
	}
	grpcServer := grpc.NewServer()
	generic.RegisterDriverServer(grpcServer, s)
	reflection.Register(grpcServer)
	logrus.Debugf("RPC server listening on address %s", s.address)
	if err := grpcServer.Serve(listen); err != nil {
		logrus.Fatal(err)
	}
	return
}
