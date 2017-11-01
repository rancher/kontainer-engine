package drivers

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

func NewRPCClient(driverName string, addr string) (*GRPCDriverPlugin, error) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	c := NewDriverClient(conn)
	return &GRPCDriverPlugin{
		client:     c,
		driverName: driverName,
	}, nil
}

type GRPCDriverPlugin struct {
	client     DriverClient
	driverName string
}

func (rpc *GRPCDriverPlugin) Create() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
	defer cancel()
	_, err := rpc.client.Create(ctx, &Empty{})
	return err
}

func (rpc *GRPCDriverPlugin) Update() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
	defer cancel()
	_, err := rpc.client.Update(ctx, &Empty{})
	return err
}

func (rpc *GRPCDriverPlugin) Get(name string) (ClusterInfo, error) {
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

func (rpc *GRPCDriverPlugin) Remove() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()
	_, err := rpc.client.Remove(ctx, &Empty{})
	return err
}

func (rpc *GRPCDriverPlugin) GetDriverCreateOptions() (DriverFlags, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	flags, err := rpc.client.GetDriverCreateOptions(ctx, &Empty{})
	if err != nil {
		return DriverFlags{}, err
	}
	return *flags, nil
}

func (rpc *GRPCDriverPlugin) GetDriverUpdateOptions() (DriverFlags, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	flags, err := rpc.client.GetDriverUpdateOptions(ctx, &Empty{})
	if err != nil {
		return DriverFlags{}, err
	}
	return *flags, nil
}

func (rpc *GRPCDriverPlugin) SetDriverOptions(options DriverOptions) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	_, err := rpc.client.SetDriverOptions(ctx, &options)
	return err
}

func (rpc *GRPCDriverPlugin) DriverName() string {
	return rpc.driverName
}
