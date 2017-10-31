package generic_driver

const (
	StringType = "string"
	BoolType = "bool"
	IntType = "int"
	StringSliceType = "stringSlice"
)

type RpcServer interface {
	Serve(errStop chan error)
}
