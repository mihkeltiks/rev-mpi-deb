build:
	make buildDebugger
	make buildCompiler

buildDebugger:
	cd src && go build -o ../bin/debug *.go
	
buildCompiler:
	cd compiler && go build -o ../bin/compiler *.go


