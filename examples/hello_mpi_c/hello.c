#include <mpi.h>
#include <stdio.h>
#include <stdlib.h>

int size;
int rank;

void stuff()
{
    MPI_Init(NULL, NULL);

    MPI_Comm_size(MPI_COMM_WORLD, &size);
    MPI_Comm_rank(MPI_COMM_WORLD, &rank);

    printf("Hello world from processor rank %d\n", rank);

    int sendNumber = 123;
    int recvNumber;
    
    MPI_Send(&sendNumber, 1, MPI_INT, rank, 0, MPI_COMM_WORLD);
    printf("%d: sending value %d to %d\n", rank, sendNumber, rank);
    
    MPI_Recv(&recvNumber, 1, MPI_INT, rank, 0, MPI_COMM_WORLD, MPI_STATUS_IGNORE);
    printf("%d: received value %d from %d\n", rank, recvNumber, rank);

    // if (size < 2)
    // {
    //     printf("Too few processes to do message passing. exiting\n");
    //     MPI_Finalize();
    //     exit(0);
    // }

    // if (rank == 0)
    // {
    //     number = 123;
    //     MPI_Send(&number, 1, MPI_INT, 1, 0, MPI_COMM_WORLD);
    //     printf("%d: sending value %d to %d\n", rank, number, 1);
    // }
    // else if (rank == 1)
    // {
    //     MPI_Recv(&number, 1, MPI_INT, 0, 0, MPI_COMM_WORLD, MPI_STATUS_IGNORE);
    //     printf("%d: received value %d from %d\n", rank, number, 0);
    // }

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