!============================================================================
! mpi_ring_matmul_changed.f90 – MPI ring matrix‑multiply (F95)
! Instrumented with `call_counter()` **before** every executable statement.
!----------------------------------------------------------------------------
! Build example:
!     mpif90 -O2 -cpp -o mpi_ring_matmul_changed mpi_ring_matmul_changed.f90
!     mpirun -np 4 ./mpi_ring_matmul_changed 512 512 512
!============================================================================
module counter_mod
   implicit none
   integer(kind=8), save :: cnt = 0  ! global instruction counter
contains
   subroutine call_counter()
      implicit none
      cnt = cnt + 1
   end subroutine call_counter

   subroutine report_counter(rank)
      implicit none
      integer, intent(in) :: rank
      if (rank == 0) then
         print *, 'Instruction counter =', cnt
      end if
   end subroutine report_counter
end module counter_mod

!----------------------------------------------------------------------------
program mpi_ring_matmul_changed
   use mpi
   use counter_mod
   implicit none

   !-------------------- Declarations --------------------------------------
   integer :: ierr, rank, size
   integer :: m, s, n
   integer :: rows_per_proc, left, right, owner, iter, k_offset, p
   real,    allocatable :: A(:,:), Bblk(:,:), tmpB(:,:), C(:,:), &
                          Cfinal(:,:), Cref(:,:), Bfull(:,:)
   real :: t0, t1, max_err
   character(len=32) :: arg1, arg2, arg3

   !-------------------- MPI setup -----------------------------------------
   call call_counter(); call MPI_Init(ierr)
   call call_counter(); call MPI_Comm_rank(MPI_COMM_WORLD, rank, ierr)
   call call_counter(); call MPI_Comm_size(MPI_COMM_WORLD, size, ierr)

   !-------------------- Default problem size ------------------------------
   call call_counter(); m = 512
   call call_counter(); s = 512
   call call_counter(); n = 512

   !-------------------- Parse command‑line --------------------------------
   if (command_argument_count() >= 3) then
      call call_counter(); call get_command_argument(1, arg1)
      call call_counter(); read(arg1, *) m
      call call_counter(); call get_command_argument(2, arg2)
      call call_counter(); read(arg2, *) s
      call call_counter(); call get_command_argument(3, arg3)
      call call_counter(); read(arg3, *) n
   end if

   !-------------------- Sanity check --------------------------------------
   if (mod(s, size) /= 0) then
      call call_counter(); if (rank == 0) print *, 'Error: s must be divisible by P'
      call call_counter(); call MPI_Abort(MPI_COMM_WORLD, 1, ierr)
   end if

   call call_counter(); rows_per_proc = s / size

   !-------------------- Allocate arrays -----------------------------------
   call call_counter(); allocate(A(m, s))
   call call_counter(); allocate(Bblk(rows_per_proc, n))
   call call_counter(); allocate(tmpB(rows_per_proc, n))
   call call_counter(); allocate(C(m, n))
   call call_counter(); C = 0.0

   ! Root creates full B and random data -----------------------------------
   if (rank == 0) then
      call call_counter(); allocate(Bfull(s, n))
      call call_counter(); call random_seed()
      call call_counter(); call random_number(A)
      call call_counter(); call random_number(Bfull)
   end if

   ! Broadcast A -----------------------------------------------------------
   call call_counter(); call MPI_Bcast(A, m*s, MPI_REAL, 0, MPI_COMM_WORLD, ierr)

   ! Distribute B row‑blocks ----------------------------------------------
   if (rank == 0) then
      call call_counter(); Bblk = Bfull(1:rows_per_proc, :)
      do p = 1, size-1
         call call_counter(); call MPI_Send( Bfull(p*rows_per_proc+1 : (p+1)*rows_per_proc, :), &
                                  rows_per_proc*n, MPI_REAL, p, 99, MPI_COMM_WORLD, ierr )
      end do
   else
      call call_counter(); call MPI_Recv( Bblk, rows_per_proc*n, MPI_REAL, 0, 99, &
                                  MPI_COMM_WORLD, MPI_STATUS_IGNORE, ierr )
   end if

   ! Ring topology parameters ---------------------------------------------
   call call_counter(); left  = mod(rank-1 + size, size)
   call call_counter(); right = mod(rank+1, size)
   call call_counter(); owner = rank

   !-------------------- Core computation ----------------------------------
   call call_counter(); call MPI_Barrier(MPI_COMM_WORLD, ierr)
   call call_counter(); t0 = MPI_Wtime()

   do iter = 0, size-1
      call call_counter(); k_offset = owner * rows_per_proc
      call call_counter(); C = C + matmul( A(:, k_offset+1 : k_offset + rows_per_proc), Bblk )

      ! Rotate B blocks clockwise
      call call_counter(); call MPI_Sendrecv( Bblk, rows_per_proc*n, MPI_REAL, left,  1, &
                                              tmpB, rows_per_proc*n, MPI_REAL, right, 1, &
                                              MPI_COMM_WORLD, MPI_STATUS_IGNORE, ierr )
      call call_counter(); Bblk = tmpB
      call call_counter(); owner = mod(owner + 1, size)
   end do

   call call_counter(); call MPI_Barrier(MPI_COMM_WORLD, ierr)
   call call_counter(); t1 = MPI_Wtime()

   !-------------------- Gather & validate ---------------------------------
   if (rank == 0) then
      call call_counter(); allocate(Cfinal(m, n))
   end if
   call call_counter(); call MPI_Reduce( C, Cfinal, m*n, MPI_REAL, MPI_SUM, 0, MPI_COMM_WORLD, ierr )

   if (rank == 0) then
      call call_counter(); Cfinal = Cfinal / size
      call call_counter(); Cref   = matmul(A, Bfull)
      call call_counter(); max_err = maxval( abs(Cref - Cfinal) )

      call call_counter(); print *, '\n===== MPI Ring MatMul (F95, counter BEFORE) ====='
      call call_counter(); print *, 'Processes           :', size
      call call_counter(); print *, 'Dimensions (m,s,n)  :', m, s, n
      call call_counter(); print *, 'Elapsed time (s)    :', t1 - t0
      call call_counter(); print *, 'Max |Δ| vs serial   :', max_err
      call call_counter(); print *, '===============================================\n'
   end if

   !-------------------- Clean up ------------------------------------------
   if (rank == 0) then
      call call_counter(); deallocate(Bfull)
      call call_counter(); deallocate(Cref)
      call call_counter(); deallocate(Cfinal)
   end if
   call call_counter(); deallocate(A)
   call call_counter(); deallocate(Bblk)
   call call_counter(); deallocate(tmpB)
   call call_counter(); deallocate(C)

   call call_counter(); call report_counter(rank)
   call call_counter(); call MPI_Finalize(ierr)
end program mpi_ring_matmul_changed
