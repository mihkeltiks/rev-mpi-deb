

# Causally Consistent Reversible Debugger for MPI

---

## Running on linux
currently only x86 architecture is supported

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
bin/orchestror <num_processes> <path-to-target-mpi-application-binary>
```


ℹ️ There's a couple of example programs included in the `examples` directory to test with.
Compile them first (`bin/compiler examples/<example-application-file>`)



<br>
<br>

## Other platforms (use Docker)

### build

```bash
# in the project root directory
make dockerimage
```

Compiling MPI programs is not supported when running with docker. Use the included examples in bin/targets folder.
### run
```bash
# use the included compiled examples
./runInDocker.sh <num_processes> bin/examples/<example-application-binary
```
