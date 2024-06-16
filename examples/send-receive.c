#include <mpi.h>
#include <stdio.h>

int rank;
int size;
int phase = 0;

void initialise()
{
    MPI_Init(NULL, NULL);

    MPI_Comm_rank(MPI_COMM_WORLD, &rank); // obtain current process rank
    MPI_Comm_size(MPI_COMM_WORLD, &size); // obtain communicator size

    printf("Node %d: hello world\n", rank);
}

void passMessages()
{
    if (size < 2)
    {
        printf("not enough processes to do message passing\n");
        return;
    }

    phase++; // phase 1

    int sendValue;
    int recvValue;

    int otherProcessRank;

    if (rank == 0){
        otherProcessRank = 1;
        sendValue = 123;
    }else if (rank == 1){
        otherProcessRank = 0;
        sendValue = 456;
    }else{
        return;}

    MPI_Send(&sendValue, 1, MPI_INT, otherProcessRank, 0, MPI_COMM_WORLD);
    printf("Node %d: sending value %d to %d\n", rank, sendValue, otherProcessRank);

    phase++; // phase 2

    MPI_Recv(&recvValue, 1, MPI_INT, otherProcessRank, 0, MPI_COMM_WORLD, MPI_STATUS_IGNORE);
    printf("Node %d: received value %d from %d\n", rank, recvValue, otherProcessRank);

    phase++; // phase 3

    return;
}

void finalise()
{
    MPI_Finalize();
    printf("Node %d: exiting\n", rank);
}

int main(int argc, char **argv)
{
    initialise();
    passMessages();
    finalise();
}
