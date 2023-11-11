# syntax=docker/dockerfile:1
FROM okmartens/golang-mpi

ENV GOPATH=/app
ENV GO111MODULE=off
WORKDIR /app

COPY . .

RUN  make

# Compile the example MPI programs
RUN bin/compiler examples/send-receive.c
# RUN bin/compiler examples/broadcast.c


RUN chmod -R 777 .

RUN useradd -u 1234 dockerUser
USER dockerUser

# ENTRYPOINT [ "./bin/orchestrator" ]
