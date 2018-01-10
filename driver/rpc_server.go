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

// Driver defines the interface that each driver plugin should implement
type Driver interface {
	// GetDriverCreateOptions returns cli flags that are used in create
	GetDriverCreateOptions() (*DriverFlags, error)

	// GetDriverUpdateOptions returns cli flags that are used in update
	GetDriverUpdateOptions() (*DriverFlags, error)

	// Create creates the cluster
	Create(opts *DriverOptions) (*ClusterInfo, error)

	// Update updates the cluster
	Update(clusterInfo *ClusterInfo, opts *DriverOptions) (*ClusterInfo, error)

	// PostCheck does post action after provisioning
	PostCheck(clusterInfo *ClusterInfo) (*ClusterInfo, error)

	// Remove removes the cluster
	Remove(clusterInfo *ClusterInfo) error
}

// GrpcServer defines the server struct
type GrpcServer struct {
	driver  Driver
	address chan string
}

// NewServer creates a grpc server for a specific plugin
func NewServer(driver Driver, addr chan string) *GrpcServer {
	return &GrpcServer{
		driver:  driver,
		address: addr,
	}
}

// GetDriverCreateOptions implements grpc method
func (s *GrpcServer) GetDriverCreateOptions(ctx context.Context, in *Empty) (*DriverFlags, error) {
	return s.driver.GetDriverCreateOptions()
}

// GetDriverUpdateOptions implements grpc method
func (s *GrpcServer) GetDriverUpdateOptions(ctx context.Context, in *Empty) (*DriverFlags, error) {
	return s.driver.GetDriverUpdateOptions()
}

// Create implements grpc method
func (s *GrpcServer) Create(ctx context.Context, opts *DriverOptions) (*ClusterInfo, error) {
	return s.driver.Create(opts)
}

// Update implements grpc method
func (s *GrpcServer) Update(ctx context.Context, update *UpdateRequest) (*ClusterInfo, error) {
	return s.driver.Update(update.ClusterInfo, update.DriverOptions)
}

func (s *GrpcServer) PostCheck(ctx context.Context, clusterInfo *ClusterInfo) (*ClusterInfo, error) {
	return s.driver.PostCheck(clusterInfo)
}

// Remove implements grpc method
func (s *GrpcServer) Remove(ctx context.Context, clusterInfo *ClusterInfo) (*Empty, error) {
	return &Empty{}, s.driver.Remove(clusterInfo)
}

// Serve serves a grpc server
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
