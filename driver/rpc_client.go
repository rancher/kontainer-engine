package drivers

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

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

type GrpcClient struct {
	client     DriverClient
	driverName string
}

func (rpc *GrpcClient) Create() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
	defer cancel()
	_, err := rpc.client.Create(ctx, &Empty{})
	return err
}

func (rpc *GrpcClient) Update() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
	defer cancel()
	_, err := rpc.client.Update(ctx, &Empty{})
	return err
}

func (rpc *GrpcClient) Get(name string) (ClusterInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	info, err := rpc.client.Get(ctx, &ClusterGetRequest{
		Name: name,
	})
	if err != nil {
		return ClusterInfo{}, err
	}
	return *info, nil
}

func (rpc *GrpcClient) Remove() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()
	_, err := rpc.client.Remove(ctx, &Empty{})
	return err
}

func (rpc *GrpcClient) GetDriverCreateOptions() (DriverFlags, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	flags, err := rpc.client.GetDriverCreateOptions(ctx, &Empty{})
	if err != nil {
		return DriverFlags{}, err
	}
	return *flags, nil
}

func (rpc *GrpcClient) GetDriverUpdateOptions() (DriverFlags, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	flags, err := rpc.client.GetDriverUpdateOptions(ctx, &Empty{})
	if err != nil {
		return DriverFlags{}, err
	}
	return *flags, nil
}

func (rpc *GrpcClient) SetDriverOptions(options DriverOptions) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	_, err := rpc.client.SetDriverOptions(ctx, &options)
	return err
}

func (rpc *GrpcClient) DriverName() string {
	return rpc.driverName
}
