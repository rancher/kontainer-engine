module github.com/rancher/kontainer-driver-metadata

go 1.12

replace (
	github.com/knative/pkg => github.com/rancher/pkg v0.0.0-20190514055449-b30ab9de040e
	github.com/matryer/moq => github.com/rancher/moq v0.0.0-20190404221404-ee5226d43009
)

require github.com/rancher/types v0.0.0-20190819173748-96e6d6f30265
