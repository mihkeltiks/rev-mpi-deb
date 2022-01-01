build:
	rm -f rebug
	go build -o bin/debug *.go

run:
	bin/debug ../hello/hello

