#include <mpi.h>

int process_rank;
int other_process_rank;

int send_value;
int receive_value;

int main(int argc, char **argv)
{
    MPI_Init(NULL, NULL);

    MPI_Comm_rank(MPI_COMM_WORLD, &process_rank);

    if (process_rank == 0)
    {
        other_process_rank = 1;
        send_value = 123;
    }
    else
    {
        other_process_rank = 0;
        send_value = 456;
    }

    MPI_Send(&send_value, 1, MPI_INT, other_process_rank, 0, MPI_COMM_WORLD);

    MPI_Recv(&receive_value, 1, MPI_INT, other_process_rank, 0, MPI_COMM_WORLD, MPI_STATUS_IGNORE);

    MPI_Finalize();
}