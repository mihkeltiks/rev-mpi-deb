#include <mpi.h>
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>

int size;
int rank;
int global = 420;

void stuff()
{
    printf("Let's go\n");

    MPI_Init(NULL, NULL);

    MPI_Comm_size(MPI_COMM_WORLD, &size);
    MPI_Comm_rank(MPI_COMM_WORLD, &rank);

    printf("Hello world from processor rank %d\n", rank);

    int sendNumber;
    int recvNumber;

    int otherProcessRank;

    if (rank == 0)
    {
        otherProcessRank = 1;
        sendNumber = 123;
    }
    else
    {
        otherProcessRank = 0;
        sendNumber = 456;
    }

    MPI_Send(&sendNumber, 1, MPI_INT, otherProcessRank, 0, MPI_COMM_WORLD);
    printf("%d: sending value %d to %d\n", rank, sendNumber, otherProcessRank);

    global = 840;
    printf("mid\n");

    MPI_Recv(&recvNumber, 1, MPI_INT, otherProcessRank, 0, MPI_COMM_WORLD, MPI_STATUS_IGNORE);
    printf("%d: received value %d from %d\n", rank, recvNumber, otherProcessRank);

    printf("end\n");

    MPI_Finalize();
}

void does()
{
    stuff();
}

int main(int argc, char **argv)
{
    does();
}