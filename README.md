

## Causally Consistent Reversible Debugger for MPI

_in development_

---

### Running on linux

#### build
```bash
make
```
#### run
```sh
./bin/debug <path-to-target-binary>
```


Other platforms (use Docker)

#### build

the compiled target binary should be moved into the source folder before building the docker image

```bash
# in the root directory
docker build -t debug .
```
#### run
```bash
# the path to binary should be relative to the root directory of the project
docker run --rm -i debug <path-to-target-binary>
```

### Compiling programs for the debugger

The target program needs to be compliled for linux and x86 architecture, and include debugging information

#### for c programs:
```bash
gcc -g -no-pie ...
```
The `-no-pie` will disable Adress Space Layout Randomization.

#### for go programs:
```bash
go build --gcflags="all=-N -l" ...
```