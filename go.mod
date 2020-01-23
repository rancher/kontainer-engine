module github.com/rancher/kontainer-engine

go 1.13

replace (
	git.apache.org/thrift.git => github.com/apache/thrift v0.12.0
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v12.2.0+incompatible
	k8s.io/client-go => k8s.io/client-go v0.17.2
)

require (
	github.com/Azure/azure-sdk-for-go v32.5.0+incompatible
	github.com/Azure/go-autorest/autorest v0.9.0
	github.com/Azure/go-autorest/autorest/adal v0.5.0
	github.com/Azure/go-autorest/autorest/to v0.2.0
	github.com/Azure/go-autorest/autorest/validation v0.2.0 // indirect
	github.com/aws/aws-sdk-go v1.16.19
	github.com/coreos/go-systemd v0.0.0-20190719114852-fd7a80b32e1f // indirect
	github.com/golang/protobuf v1.3.2
	github.com/grpc-ecosystem/grpc-gateway v1.9.6 // indirect
	github.com/heptio/authenticator v0.0.0-20180409043135-d282f87a1972
	github.com/pkg/errors v0.8.1
	github.com/rancher/norman v0.0.0-20200123223841-6d86f4e37a69
	github.com/rancher/rke v1.1.0-rc4.0.20200123230443-25e7f987775d
	github.com/rancher/types v0.0.0-20200123224322-9adcafc483ee
	github.com/sirupsen/logrus v1.4.2
	github.com/stretchr/testify v1.4.0
	github.com/urfave/cli v1.20.0
	golang.org/x/crypto v0.0.0-20190911031432-227b76d455e7
	golang.org/x/net v0.0.0-20191004110552-13f9640d40b9
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	google.golang.org/api v0.4.0
	google.golang.org/grpc v1.23.1
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127
	gopkg.in/yaml.v2 v2.2.4
	k8s.io/api v0.17.2
	k8s.io/apimachinery v0.17.2
	k8s.io/client-go v11.0.1-0.20190805182715-88a2adca7e76+incompatible
)
