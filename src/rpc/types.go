package rpc

type MPICallRecord struct {
	Id         string
	OpName     string
	Parameters map[string]string
	NodeId     int
}
