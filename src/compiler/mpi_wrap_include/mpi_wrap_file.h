#include <mpi.h>
#include <signal.h>
#include <stdio.h>
#include <sys/types.h>
#include <unistd.h>

void _MPI_WRAPPER_INCLUDE() {}

int _MPI_CURRENT_SOURCE;
int _MPI_CURRENT_DEST;
int _MPI_CURRENT_TAG;

int _MPI_CHECKPOINT_CHILD;

void _MPI_WRAPPER_RECORD(
    const void *buf,
    int count,
    MPI_Datatype datatype,
    int source,
    int dest,
    int tag,
    MPI_Comm comm,
    MPI_Status *status)
{
    _MPI_CURRENT_DEST = dest;
    _MPI_CURRENT_SOURCE = source;
    _MPI_CURRENT_TAG = tag;
}

int _MPI_Init(int *argc, char ***argv)
{
    return MPI_Init(argc, argv);
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
    int code = MPI_Send(buf, count, datatype, dest, tag, comm);
    _MPI_WRAPPER_RECORD(buf, count, datatype, -1, dest, tag, comm, NULL);
    return code;
}

int _MPI_Recv(void *buf, int count, MPI_Datatype datatype, int source,
              int tag, MPI_Comm comm, MPI_Status *status)
{
    int code = MPI_Recv(buf, count, datatype, source, tag, comm, status);
    _MPI_WRAPPER_RECORD(buf, count, datatype, source, -1, tag, comm, status);
    return code;
}