package drivers

const (
	StringType      = "string"
	BoolType        = "bool"
	IntType         = "int"
	StringSliceType = "stringSlice"
)

type RPCServer interface {
	Serve()
}
