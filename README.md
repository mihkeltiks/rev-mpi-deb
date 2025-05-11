# Causally Consistent Reversible Debugger for MPI

---

## Running on linux
Currently only x86_64 architecture is supported. Since CRIU requires root it is recommended to run as a container or on a virtual machine. This has been tested with Go/1.21, MPICH/3.4.3, and requires an MPI compiler that is capable of producing level 4 DWARF data. Running might require editing root user $PATH to execute mpirun and CRIU or DMTCP.

Details for both backends, CRIU and DMTCP, are found in [manual.md](manual.md).

### build

```bash
make
```
### compile target for debugger
There is a compiler included that wraps the mpi library calls, in order to enable the debugger to intercept and record them.
Programs must be compiled with the included compiler script:

```sh
bin/compiler <path-to-target-MPI-program>
```
The compiled binary will be written to `./bin/targets/<source-file-name>`. This path should be given to the debugger as input.

### run
```sh
bin/orchestrator <num_processes> <path-to-target-mpi-application-binary> <criu|dmtcp>
```


There's a couple of example programs included in the `examples` directory to test with.
Compile them first (`bin/compiler examples/<example-application-file>`)



<br>
<br>
