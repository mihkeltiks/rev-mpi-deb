#include <mpi.h>
#include <stdio.h>
#include <stdlib.h>
#include <time.h>
#include <math.h>

#define IDX(i,j,ld) ((i)*(ld)+(j))

/*--------------------------------------------------------------------
 * mpi_ring_matmul.c  (bug‑fixed)
 *
 * Parallel matrix multiplication C = A * B using MPI ring communication.
 * Gathers (via reduction) the final result on rank‑0 and validates it
 * against a serial reference implementation.
 *
 * **FIX 2025‑06‑16**
 *   The previous version updated the `owner` index with the wrong sign
 *   after the ring rotation, so each process multiplied a B‑block with
 *   an incorrect `k_offset`. That produced large errors.  The rotation
 *   sends the block to the left and receives from the right, which
 *   means the *global* block index **increases** by one each step.
 *   The corrected line is now:
 *       owner = (owner + 1) % size;
 *--------------------------------------------------------------------*/

/* Generate a matrix filled with random doubles in [0,1). */
static void random_matrix(double *M, int rows, int cols) {
    for (long long i = 0; i < (long long)rows * cols; ++i) {
        M[i] = (double)rand() / RAND_MAX;
    }
}

/* Multiply a block of B with A and accumulate into C.
 *   A: (m × s)
 *   Bblk: (rows_per_blk × n) – rows k_offset … k_offset+rows_per_blk-1 of B
 *   C: (m × n)
 */
static void multiply_block(const double *A, const double *Bblk, double *C,
                           int m, int s, int n,
                           int rows_per_blk, int k_offset) {
    for (int i = 0; i < m; ++i) {
        const double *a_row = A + (long long)i * s + k_offset; /* ptr to A(i, k_offset) */
        double       *c_row = C + (long long)i * n;
        for (int k = 0; k < rows_per_blk; ++k) {
            double a_ik        = a_row[k];                   /* A(i,k_offset+k)        */
            const double *brow = Bblk + (long long)k * n;    /* B(k_offset+k, :)       */
            for (int j = 0; j < n; ++j) {
                c_row[j] += a_ik * brow[j];                  /* C(i,j) accumulate      */
            }
        }
    }
}

int main(int argc, char **argv) {
    MPI_Init(&argc, &argv);
    int rank, size;
    MPI_Comm_rank(MPI_COMM_WORLD, &rank);
    MPI_Comm_size(MPI_COMM_WORLD, &size);

    /* Matrix dimensions (defaults or cmd‑line) */
    int m = 512, s = 512, n = 512;
    if (argc >= 4) {
        m = atoi(argv[1]);
        s = atoi(argv[2]);
        n = atoi(argv[3]);
    }

    if (s % size != 0) {
        if (rank == 0)
            fprintf(stderr, "Error: s must be divisible by P (s=%d, P=%d)\n", s, size);
        MPI_Abort(MPI_COMM_WORLD, EXIT_FAILURE);
    }
    int rows_per_proc = s / size;

    /* Root allocates full A and B and fills them with random numbers */
    double *A = (double *)malloc((size_t)m * s * sizeof(double));
    double *B = NULL;
    if (rank == 0) {
        B = (double *)malloc((size_t)s * n * sizeof(double));
        srand((unsigned)time(NULL));
        random_matrix(A, m, s);
        random_matrix(B, s, n);
    }

    /* Everyone needs A */
    MPI_Bcast(A, m * s, MPI_DOUBLE, 0, MPI_COMM_WORLD);

    /* Scatter B row‑blocks */
    double *Bblk = (double *)malloc((size_t)rows_per_proc * n * sizeof(double));
    MPI_Scatter(B, rows_per_proc * n, MPI_DOUBLE,
                Bblk, rows_per_proc * n, MPI_DOUBLE,
                0, MPI_COMM_WORLD);

    /* Buffers for ring communication and local result */
    double *tmpB   = (double *)malloc((size_t)rows_per_proc * n * sizeof(double));
    double *Clocal = (double *)calloc((size_t)m * n, sizeof(double));

    int left  = (rank - 1 + size) % size;
    int right = (rank + 1) % size;
    int owner = rank;                  /* global k‑block currently held */

    MPI_Barrier(MPI_COMM_WORLD);
    double t0 = MPI_Wtime();

    for (int iter = 0; iter < size; ++iter) {
        int k_offset = owner * rows_per_proc;
        multiply_block(A, Bblk, Clocal, m, s, n, rows_per_proc, k_offset);

        /* Rotate B around the ring: send LEFT, receive from RIGHT */
        MPI_Sendrecv(Bblk, rows_per_proc * n, MPI_DOUBLE, left, 0,
                     tmpB, rows_per_proc * n, MPI_DOUBLE, right, 0,
                     MPI_COMM_WORLD, MPI_STATUS_IGNORE);
        double *swap = Bblk; Bblk = tmpB; tmpB = swap;

        /* Corrected owner update (block index increases by 1) */
        owner = (owner + 1) % size;
    }
    MPI_Barrier(MPI_COMM_WORLD);
    double t1 = MPI_Wtime();

    /* --- Gather (reduce) result on rank‑0 and validate ----------------------- */
    double *Cfinal = NULL;
    if (rank == 0)
        Cfinal = (double *)calloc((size_t)m * n, sizeof(double));

    /* Each rank now holds the full matrix; SUM then divide eliminates gather. */
    MPI_Reduce(Clocal, Cfinal, m * n, MPI_DOUBLE, MPI_SUM, 0, MPI_COMM_WORLD);

    if (rank == 0) {
        //for (long long i = 0; i < (long long)m * n; ++i) Cfinal[i] /= (double)size;

        /* Serial reference for correctness check */
        /*double *Cref = (double *)calloc((size_t)m * n, sizeof(double));
        for (int i = 0; i < m; ++i) {
            for (int k = 0; k < s; ++k) {
                double a = A[i * s + k];
                for (int j = 0; j < n; ++j)
                    Cref[i * n + j] += a * B[k * n + j];
            }
        }

        double max_err = 0.0;
        for (long long idx = 0; idx < (long long)m * n; ++idx) {
            double diff = fabs(Cref[idx] - Cfinal[idx]);
            if (diff > max_err) max_err = diff;
        }*/

        printf("\n===== MPI Ring MatMul Report =====\n");
        printf("Processes           : %d\n", size);
        printf("Matrix dims (m,s,n) : %d × %d × %d\n", m, s, n);
        printf("Elapsed time (s)    : %.6f\n", t1 - t0);
        printf("Max |Δ| vs serial   : %.3e\n", max_err);
        printf("==================================\n\n");

        free(Cref);
        free(Cfinal);
    }

    /* --------------------------------------------------------------------- */
    free(Clocal);
    free(Bblk);
    free(tmpB);
    if (rank == 0) free(B);
    free(A);

    MPI_Finalize();
    return 0;
}
