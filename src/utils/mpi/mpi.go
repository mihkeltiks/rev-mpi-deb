package mpi

type MPICallRecord struct {
	Id         string
	OpName     string
	Parameters map[string]string
	NodeId     int
}

type MPI_OPCODE int

const (
	OP_INIT MPI_OPCODE = iota
	OP_SEND
	OP_RECV
	OP_FINALIZE
)

var MPI_OPS = map[MPI_OPCODE]string{
	OP_INIT:     "MPI_Init",
	OP_SEND:     "MPI_Send",
	OP_RECV:     "MPI_Recv",
	OP_FINALIZE: "MPI_Finalize",
}

var SEND_EVENTS = map[string]bool{
	MPI_OPS[OP_SEND]: true,
}

var RESTORABLE_OPERATIONS = map[string]bool{
	MPI_OPS[OP_SEND]: true,
	MPI_OPS[OP_RECV]: true,
}
