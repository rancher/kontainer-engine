package drivers

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

// NewClient creates a grpc client for a driver plugin
func NewClient(driverName string, addr string) (*GrpcClient, error) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	c := NewDriverClient(conn)
	return &GrpcClient{
		client:     c,
		driverName: driverName,
	}, nil
}

// GrpcClient defines the grpc client struct
type GrpcClient struct {
	client     DriverClient
	driverName string
}

// Create call grpc create
func (rpc *GrpcClient) Create(opts *DriverOptions) (*ClusterInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
	defer cancel()
	return rpc.client.Create(ctx, opts)
}

// Update call grpc update
func (rpc *GrpcClient) Update(clusterInfo *ClusterInfo, opts *DriverOptions) (*ClusterInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
	defer cancel()
	return rpc.client.Update(ctx, &UpdateRequest{
		ClusterInfo:   clusterInfo,
		DriverOptions: opts,
	})
}

func (rpc *GrpcClient) PostCheck(clusterInfo *ClusterInfo) (*ClusterInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	return rpc.client.PostCheck(ctx, clusterInfo)
}

// Remove call grpc remove
func (rpc *GrpcClient) Remove(clusterInfo *ClusterInfo) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()
	_, err := rpc.client.Remove(ctx, clusterInfo)
	return err
}

// GetDriverCreateOptions call grpc getDriverCreateOptions
func (rpc *GrpcClient) GetDriverCreateOptions() (DriverFlags, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	flags, err := rpc.client.GetDriverCreateOptions(ctx, &Empty{})
	if err != nil {
		return DriverFlags{}, err
	}
	return *flags, nil
}

// GetDriverUpdateOptions call grpc getDriverUpdateOptions
func (rpc *GrpcClient) GetDriverUpdateOptions() (DriverFlags, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	flags, err := rpc.client.GetDriverUpdateOptions(ctx, &Empty{})
	if err != nil {
		return DriverFlags{}, err
	}
	return *flags, nil
}

// DriverName returns the driver name
func (rpc *GrpcClient) DriverName() string {
	return rpc.driverName
}
