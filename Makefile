build:
	cd src/debugger && go build -o ../../bin/debug *.go	
	
compiler:
	cd src/compiler && go build -o ../../bin/compiler *.go


