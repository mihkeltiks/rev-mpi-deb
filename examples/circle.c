#include <mpi.h>
#include <stdio.h>

int rank;
int size;
int value = 0;

void passMessages(){
    if (size < 2){
        printf("not enough processes to do message passing\n");
        return;}

    int previousRank = rank == 0 ? size - 1 : rank - 1;
    int nextRank = rank == size - 1 ? 0 : rank + 1;
    value = 10;

    // first process sends the value
    if (rank == 0){
        value = 123;

        printf("Node %d: initiating communication\n", rank);
        MPI_Send(&value, 1, MPI_INT, nextRank, 0, MPI_COMM_WORLD);
        MPI_Recv(&value, 1, MPI_INT, previousRank, 0,MPI_COMM_WORLD, MPI_STATUS_IGNORE);
        printf("Node %d: received value, end of communication\n", rank);
    }else{
        MPI_Recv(&value, 1, MPI_INT, previousRank, 0,MPI_COMM_WORLD, MPI_STATUS_IGNORE);
        printf("Node %d: passing the message forward\n", rank);
        MPI_Send(&value, 1, MPI_INT, nextRank, 0, MPI_COMM_WORLD);}
    
    printf("Node %d: value: %d\n", rank, value);
    return;
}

int main(int argc, char **argv)
{
    MPI_Init(NULL, NULL);

    MPI_Comm_rank(MPI_COMM_WORLD, &rank);
    MPI_Comm_size(MPI_COMM_WORLD, &size);

    passMessages();

    MPI_Finalize();

    return 0;}







