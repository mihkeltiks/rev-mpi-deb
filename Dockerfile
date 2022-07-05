# syntax=docker/dockerfile:1
FROM okmartens/golang-mpi

WORKDIR /app

COPY . .

RUN chmod -R 755 .

RUN useradd -u 1234 dockerUser
USER dockerUser

# RUN  make compiler
# RUN  make

# RUN bin/compiler examples/hello.c

# ENTRYPOINT [ "./bin/debug" ]