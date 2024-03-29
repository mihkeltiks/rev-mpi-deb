#include <mpi.h>
#include <stdio.h>
int rank;
int size;
int value = 0;
int a = 1;
int b = 0;
void randomfunction(){
    b=1;
}
void passMessages(){
    int previousRank = rank == 0 ? size - 1 : rank - 1;
    int nextRank = rank == size - 1 ? 0 : rank + 1;

    if (rank == 0){
        value = 123;
        printf("Node %d: initiating communication\n", rank);
        MPI_Send(&value, 1, MPI_INT, nextRank, 0, MPI_COMM_WORLD);
        MPI_Recv(&value, 1, MPI_INT, previousRank,
                 0, MPI_COMM_WORLD, MPI_STATUS_IGNORE);
        printf("Node %d: received value\n", rank);
    } else{
        MPI_Recv(&value, 1, MPI_INT, previousRank, 
                 0, MPI_COMM_WORLD, MPI_STATUS_IGNORE);
        printf("Node %d: passing the message forward\n", rank);
        MPI_Send(&value, 1, MPI_INT, nextRank, 0, MPI_COMM_WORLD);
    }
    a = 4;
    printf("Node %d: value: %d\n", rank, value);
    return;
}

int main(int argc, char **argv){
    MPI_Init(NULL, NULL);
    int a = 1;
    int b = 2;
    int c = 3;

    if (a==0){
        printf("asdf");
    }
    printf("aaa\n");
    a = 2;
    printf("bbb\n");
    a = 3;
    printf("aaa\n");
    randomfunction();
    randomfunction();
    randomfunction();
    MPI_Comm_rank(MPI_COMM_WORLD, &rank);
    MPI_Comm_size(MPI_COMM_WORLD, &size);
    passMessages();
    passMessages();
    MPI_Finalize();

    return 0;
}



