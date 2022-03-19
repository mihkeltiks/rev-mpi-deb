build:
	cd src/debugger && go build -o ../../bin/debug *.go	
	
compiler:
	cd src/compiler && go build -o ../../bin/compiler *.go

docker:
	make && docker build -t mpi-debugger .

testRunner:
	cd src/testRunner && GOOS=darwin go build -o ../../bin/testRunner *.go
