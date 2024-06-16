UNAME_S := $(shell uname -s)

build:
	mkdir -p bin/temp
	cd src/nodeDebugger && go build -o ../../bin/node-debugger *.go
	cd src/orchestrator && go build -o ../../bin/orchestrator *.go
	cd src/compiler && go build -o ../../bin/compiler *.go
	cd gui && npm install
