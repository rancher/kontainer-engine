module github.com/rancher/kontainer-engine

go 1.13

replace (
	git.apache.org/thrift.git => github.com/apache/thrift v0.12.0
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v12.2.0+incompatible
	github.com/knative/pkg => github.com/rancher/pkg v0.0.0-20190514055449-b30ab9de040e
	github.com/matryer/moq => github.com/rancher/moq v0.0.0-20190404221404-ee5226d43009
	k8s.io/client-go => k8s.io/client-go v0.18.0
)

require (
	github.com/Azure/azure-sdk-for-go v36.1.0+incompatible
	github.com/Azure/go-autorest/autorest v0.9.3-0.20191028180845-3492b2aff503
	github.com/Azure/go-autorest/autorest/adal v0.8.1-0.20191028180845-3492b2aff503
	github.com/Azure/go-autorest/autorest/to v0.3.1-0.20191028180845-3492b2aff503
	github.com/aws/aws-sdk-go v1.25.48
	github.com/coreos/go-systemd v0.0.0-20190719114852-fd7a80b32e1f // indirect
	github.com/golang/protobuf v1.3.2
	github.com/heptio/authenticator v0.0.0-20180409043135-d282f87a1972
	github.com/pkg/errors v0.8.1
	github.com/rancher/norman v0.0.0-20200609224801-7afd2e9bf37f
	github.com/rancher/rke v1.1.5-rc3
	github.com/rancher/types v0.0.0-20200723224053-ae84b42a4797
	github.com/sirupsen/logrus v1.4.2
	github.com/stretchr/testify v1.4.0
	github.com/urfave/cli v1.20.0
	golang.org/x/crypto v0.0.0-20200220183623-bac4c82f6975
	golang.org/x/net v0.0.0-20200226121028-0de0cce0169b
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	google.golang.org/api v0.14.0
	google.golang.org/grpc v1.26.0
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15
	gopkg.in/yaml.v2 v2.2.8
	k8s.io/api v0.18.0
	k8s.io/apimachinery v0.18.0
	k8s.io/client-go v12.0.0+incompatible
)
