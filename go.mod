module github.com/rancher/kontainer-engine

go 1.12

require (
	github.com/Azure/azure-sdk-for-go v19.1.0+incompatible
	github.com/Azure/go-autorest v10.11.1+incompatible
	github.com/aws/aws-sdk-go v1.16.19
	github.com/golang/protobuf v1.3.2
	github.com/heptio/authenticator v0.0.0-20180409043135-d282f87a1972
	github.com/pkg/errors v0.8.1
	github.com/rancher/norman v0.0.0-20190819172543-9c5479f6e5ca
	github.com/rancher/rke v0.3.0-rc6.0.20190819180243-f8bac2c059d0
	github.com/rancher/types v0.0.0-20190819173748-96e6d6f30265
	github.com/sirupsen/logrus v1.4.2
	github.com/stretchr/testify v1.3.0
	github.com/urfave/cli v1.20.0
	golang.org/x/net v0.0.0-20190613194153-d28f0bde5980
	golang.org/x/oauth2 v0.0.0-20190402181905-9f3314589c9a
	google.golang.org/api v0.1.0
	google.golang.org/grpc v1.23.0
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127
	gopkg.in/yaml.v2 v2.2.2
	k8s.io/api v0.0.0-20190805182251-6c9aa3caf3d6
	k8s.io/apimachinery v0.0.0-20190404173353-6a84e37a896d
	k8s.io/client-go v11.0.1-0.20190805182715-88a2adca7e76+incompatible
)
