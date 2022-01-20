

# Causally Consistent Reversible Debugger for MPI

_in development_

---

## Running on linux
currently only x86_64 architecture is supported

### build
```bash
make
```
### run
**regular binaries**
```sh
./bin/debug <path-to-target-binary>
```

**mpi**
```sh
mpirun -n <> xterm -e ./bin/debug <path-to-target-binary>
```




## Other platforms (use Docker)
MPI debugging is not yet supported with this configuration

### build

the compiled target binary should be moved into the source folder before building the docker image

```bash
# in the root directory
docker build -t debug .
```
### run
```bash
# the path to binary should be relative to the root directory of the project
docker run --rm -i debug <path-to-target-binary>
```

--- 

ℹ️ There's a couple of example programs included in the `examples` directory to test with

---

## Compiling programs for the debugger



The target program needs to be compliled for linux and x86 architecture, and include debugging information

### for c programs:
```bash
gcc -g -no-pie ...
```
The `-no-pie` will disable Adress Space Layout Randomization.

### MPI (c) programs
```bash
mpicc -g -no-pie ...
```

### for go programs:
```bash
go build --gcflags="all=-N -l" ...
```
