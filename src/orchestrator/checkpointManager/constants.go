package checkpointmanager

var SEND_EVENTS = map[string]bool{
	"MPI_Send": true,
}

var RESTORABLE_OPERATIONS = map[string]bool{
	"MPI_Init": false,
	"MPI_Send": true,
	"MPI_Recv": true,
}
