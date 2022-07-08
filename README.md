

# Causally Consistent Reversible Debugger for MPI

_in development_

---

## Running on linux
currently only x86_64 architecture is supported

### build
```bash
make && make orchestrator
```
### run
```sh
bin/orchestror <num_processes> <path-to-target-mpi-binary>
```




## Other platforms (use Docker)

### build

the compiled target binary should be moved into the source folder before building the docker image

```bash
# in the project root directory
make docker
```
### run
```bash
# the path to binary should be relative to the root directory of the project
./runInDocker.sh bin/orcherstrator <num_processes> <path-to-target-binary>
```

--- 

ℹ️ There's a couple of example programs included in the `examples` directory to test with

---

## Compiling programs for the debugger

There is a compiler included that wraps the mpi library calls, in order to enable the debugger to intercept and record them.

### 1. build the compiler
```bash
make compiler
```

### 2. compile your program:
```bash
.bin/compiler <path-to-source-file>
```
The compiled binary will be written to `./bin/target/{source-file-name}`. This path should be given to the debugger as input 

