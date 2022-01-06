# syntax=docker/dockerfile:1
FROM golang

WORKDIR /app

COPY . .

RUN  cd src && go build -o ../bin/debug

ENTRYPOINT [ "./bin/debug" ]