#include <mpi.h>

int _MPI_WRAPPER_PROC_RANK;

void _MPI_WRAPPER_INCLUDE() {}

int _MPI_Init(int *argc, char ***argv)
{
    int ret = MPI_Init(argc, argv);
    // Record process rank on comm_world
    MPI_Comm_rank(MPI_COMM_WORLD, &_MPI_WRAPPER_PROC_RANK);
    return ret;
}

int _MPI_Comm_size(MPI_Comm comm, int *size)
{
    return MPI_Comm_size(comm, size);
}

int _MPI_Comm_rank(MPI_Comm comm, int *rank)
{
    return MPI_Comm_rank(comm, rank);
}

int _MPI_Finalize()
{
    return MPI_Finalize();
}

int _MPI_Send(const void *buf, int count, MPI_Datatype datatype, int dest,
              int tag, MPI_Comm comm)
{
    return MPI_Send(buf, count, datatype, dest, tag, comm);
}

int _MPI_Recv(void *buf, int count, MPI_Datatype datatype, int source,
              int tag, MPI_Comm comm, MPI_Status *status)
{
    return MPI_Recv(buf, count, datatype, source, tag, comm, status);
}
