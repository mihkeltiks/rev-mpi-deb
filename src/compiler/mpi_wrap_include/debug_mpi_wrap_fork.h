#include <mpi.h>
#include <signal.h>
#include <sys/types.h>
#include <unistd.h>

void _MPI_WRAPPER_INCLUDE() {}

int _MPI_CHECKPOINT_CHILD;

void _MPI_WRAPPER_RECORD()
{
    _MPI_CHECKPOINT_CHILD = fork();
    if (_MPI_CHECKPOINT_CHILD == 0)
    {
        sigset_t set;
        (void)sigaddset(&set, 9);
        sigsuspend(&set);
    }
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
    _MPI_WRAPPER_RECORD();
    int code = MPI_Send(buf, count, datatype, dest, tag, comm);
    return code;
}

int _MPI_Recv(void *buf, int count, MPI_Datatype datatype, int source,
              int tag, MPI_Comm comm, MPI_Status *status)
{
    _MPI_WRAPPER_RECORD();
    int code = MPI_Recv(buf, count, datatype, source, tag, comm, status);
    return code;
}
