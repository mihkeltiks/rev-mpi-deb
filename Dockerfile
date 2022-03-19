# syntax=docker/dockerfile:1
FROM okmartens/golang-mpi

WORKDIR /app

COPY . .

# RUN  make compiler
# RUN  make

# RUN bin/compiler examples/hello.c

ENTRYPOINT [ "./bin/debug" ]