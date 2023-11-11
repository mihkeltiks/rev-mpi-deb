#include <stdio.h>
#include <mpi.h>

int main(int argc, char *argv[]) {
    int rank, size, a, b, c, d;
    MPI_Init(&argc, &argv);
    MPI_Comm_rank(MPI_COMM_WORLD, &rank);
    MPI_Comm_size(MPI_COMM_WORLD, &size);
    printf("Hello from process %d of %d\n", rank, size);
    c = 3;
    d = 5;
    c = 2;
    d = 2;
    MPI_Finalize();
    d = 4;
    c = 2;
    return 0;
}
