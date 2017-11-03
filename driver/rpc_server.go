package drivers

import (
	"net"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	listenAddr = "127.0.0.1:"
)

type Driver interface {
	GetDriverCreateOptions() (*DriverFlags, error)

	GetDriverUpdateOptions() (*DriverFlags, error)

	SetDriverOptions(driverOptions *DriverOptions) error

	Create() error

	Update() error

	Get(request *ClusterGetRequest) (*ClusterInfo, error)

	Remove() error
}

type GrpcServer struct {
	driver  Driver
	address chan string
}

func NewServer(driver Driver, addr chan string) *GrpcServer {
	return &GrpcServer{
		driver:  driver,
		address: addr,
	}
}

func (s *GrpcServer) GetDriverCreateOptions(ctx context.Context, in *Empty) (*DriverFlags, error) {
	return s.driver.GetDriverCreateOptions()
}

func (s *GrpcServer) GetDriverUpdateOptions(ctx context.Context, in *Empty) (*DriverFlags, error) {
	return s.driver.GetDriverUpdateOptions()
}

func (s *GrpcServer) SetDriverOptions(ctx context.Context, in *DriverOptions) (*Empty, error) {
	return &Empty{}, s.driver.SetDriverOptions(in)
}

func (s *GrpcServer) Create(ctx context.Context, in *Empty) (*Empty, error) {
	return &Empty{}, s.driver.Create()
}

func (s *GrpcServer) Update(ctx context.Context, in *Empty) (*Empty, error) {
	return &Empty{}, s.driver.Update()
}

func (s *GrpcServer) Get(cont context.Context, request *ClusterGetRequest) (*ClusterInfo, error) {
	return s.driver.Get(request)
}

func (s *GrpcServer) Remove(ctx context.Context, in *Empty) (*Empty, error) {
	return &Empty{}, s.driver.Remove()
}

func (s *GrpcServer) Serve() {
	listen, err := net.Listen("tcp", listenAddr)
	if err != nil {
		logrus.Fatal(err)
	}
	addr := listen.Addr().String()
	s.address <- addr
	grpcServer := grpc.NewServer()
	RegisterDriverServer(grpcServer, s)
	reflection.Register(grpcServer)
	logrus.Debugf("RPC GrpcServer listening on address %s", addr)
	if err := grpcServer.Serve(listen); err != nil {
		logrus.Fatal(err)
	}
	return
}
