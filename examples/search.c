// https://hpc.nmsu.edu/discovery/mpi/programming-with-mpi/#_mpi_sequential_search_example
#include <stdio.h>
#include <stdlib.h>
#include <time.h>
#include <mpi.h>

#define ARRAY_SIZE 100000

int num = 3;
int index, i, pid, number_of_processes, elements_per_process, num_of_elements_recieved, elements_left;
static int list_of_numbers[ARRAY_SIZE];
static int buffer[ARRAY_SIZE];
unsigned long frequency;
time_t t;

int main(int argc, char** argv)
{
	// Use current time as seed for random generator
	srand((unsigned) time(&t));

	//Fill the array with numbers randomly generated
	for( i = 0 ; i < ARRAY_SIZE ; ++i )
		list_of_numbers[i] = rand() % 100;

	// a data struct that provides more information on the received  message
	MPI_Status status;

	// Initialize the MPI environment
	MPI_Init(NULL, NULL);

	// Get the rank of the process
	MPI_Comm_rank(MPI_COMM_WORLD, &pid);

	// Get the number of processes
	MPI_Comm_size(MPI_COMM_WORLD, &number_of_processes);

	if (pid == 0) {
		// master process
		elements_per_process = ARRAY_SIZE / number_of_processes;

		// check if more than 1 processes are running
		if (number_of_processes > 1) {
			// distributes the portion of the array among all processes
			for (i = 1; i < number_of_processes - 1; i++) {
				index = i * elements_per_process;

				MPI_Send(&elements_per_process,1, MPI_INT, i, 0,MPI_COMM_WORLD);
				MPI_Send(&list_of_numbers[index],elements_per_process,MPI_INT, i, 0,MPI_COMM_WORLD);
			}

			// last process adds remaining elements
			index = i * elements_per_process;
			elements_left = ARRAY_SIZE - index;

			MPI_Send(&elements_left,1, MPI_INT,i, 0,MPI_COMM_WORLD);
			MPI_Send(&list_of_numbers[index],elements_left,MPI_INT, i, 0,MPI_COMM_WORLD);
		}

		// master process computes the frequency in its portion of the array
		frequency = 0;
		for(i = 0; i < elements_per_process; ++i)
			if(list_of_numbers[i] == num)
				frequency += 1;

		// collect partial frequency from other processes
		unsigned long buffer = 0;
		for (i = 1; i < number_of_processes; i++) {
			MPI_Recv(&buffer, 1, MPI_INT,MPI_ANY_SOURCE, 0,MPI_COMM_WORLD,&status);
			frequency += buffer;
		}

		// print the frequency of user input in the list of numbers
		printf("The frequency of %d in the list of numbers is %ld\n", num, frequency);
	} else {
		// worker processes

		num_of_elements_recieved = 0;
		frequency = 0;

		MPI_Recv(&num_of_elements_recieved,1, MPI_INT, 0, 0,MPI_COMM_WORLD,&status);

		// store the received portion of the array in buffer
		MPI_Recv(&buffer, num_of_elements_recieved,MPI_INT, 0, 0,MPI_COMM_WORLD,&status);

		// compute the frequency in received portion of the array
		for(i = 0; i < num_of_elements_recieved; ++i)
			if(buffer[i] == num)
				frequency += 1;

		// send the computation result to the master process
		MPI_Send(&frequency, 1, MPI_INT,0, 0, MPI_COMM_WORLD);
	}

	// Finalize the MPI environment
	MPI_Finalize();
	return 0;
}
