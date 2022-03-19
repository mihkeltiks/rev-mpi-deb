UNAME_S := $(shell uname -s)

build:
	cd src/debugger && go build -o ../../bin/debug *.go	
	
compiler:
	cd src/compiler && go build -o ../../bin/compiler *.go

docker:
	make && docker build -t mpi-debugger .

testRunner:
    ifeq ($(UNAME_S),Linux)
		cd src/testRunner && GOOS=linux go build -o ../../bin/testRunner *.go
    endif
    ifeq ($(UNAME_S),Darwin)
        cd src/testRunner && GOOS=darwin go build -o ../../bin/testRunner *.go
    endif

	
