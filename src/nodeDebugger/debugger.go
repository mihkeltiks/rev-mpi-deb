package main

import (
	"io"
	"os"
	"os/exec"
	"runtime"
	"syscall"

	"github.com/mihkeltiks/rev-mpi-deb/logger"
	"github.com/mihkeltiks/rev-mpi-deb/nodeDebugger/dwarf"
	"github.com/mihkeltiks/rev-mpi-deb/rpc"
	"github.com/mihkeltiks/rev-mpi-deb/utils/command"
)

const MAIN_FN = "main"

type processContext struct {
	targetFile     string           // the executing binary file
	sourceFile     string           // source code file
	dwarfData      *dwarf.DwarfData // dwarf debug information about the binary
	process        *exec.Cmd        // the running binary
	pid            int              // the process id of the running binary
	bpointData     breakpointData   // holds the instuctions for currently replaced by breakpoints
	cpointData     checkpointData   // holds data about currently recorded checkppoints
	checkpointMode CheckpointMode   // whether checkpoints are recorded in files or in forked processes
	stack          programStack     // current call stack of the target. updated after each command execution
	nodeData       *nodeData        // data about connection with the orchestrator
}

type nodeData struct {
	id        int            // designated by the orchestrator
	rpcClient *rpc.RPCClient // rpc client for communicating with the orchestrator
}

func main() {
	// As ptrace calls depend on per-thread state, we must lock the thread
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	precleanup()

	targetFile, checkpointMode, orchestratorAddress, standaloneMode := getValuesFromArgs()

	ctx := &processContext{
		targetFile:     targetFile,
		checkpointMode: checkpointMode,
		bpointData:     breakpointData{}.New(),
		cpointData:     checkpointData{}.New(),
	}

	if !standaloneMode {
		// connect to orchestrator
		ctx.nodeData = &nodeData{
			rpcClient: rpc.Connect(orchestratorAddress),
		}

		ctx.nodeData.id = reportAsHealthy(ctx)
		logger.SetRemoteClient(ctx.nodeData.rpcClient, ctx.nodeData.id)

		logger.Info("Process (pid: %d) registered", os.Getpid())
	}

	// parse debugging data
	ctx.dwarfData = dwarf.ParseDwarfData(ctx.targetFile)
	ctx.dwarfData.ResolveMPIDebugInfo()
	ctx.sourceFile = ctx.dwarfData.FindEntrySourceFile(MAIN_FN)

	// start target binary
	ctx.process = startBinary(ctx.targetFile)
	ctx.pid = ctx.process.Process.Pid

	// set up automatic breakpoints
	insertMPIBreakpoints(ctx)

	if standaloneMode {
		handleCLIWorkflow(ctx)
	} else {
		handleRemoteWorkflow(ctx)
	}

}

func handleRemoteWorkflow(ctx *processContext) {
	// channel for commands scheduled for execution by orchestrator
	commandQueue := make(chan *command.Command, 10)

	go func() {
		port := 3500 + ctx.nodeData.id
		rpc.InitializeServer(port, func(register rpc.Registrator) {
			logger.Verbose("Registering debugging methods for remote use")

			register(&RemoteCmdHandler{ctx, commandQueue})
		})
	}()

	for {
		cmd := <-commandQueue
		handleCommand(ctx, cmd)

		reportCommandResult(ctx, cmd)

		if cmd.Result.Exited {
			logger.Info("Exiting")
			break
		}
	}
}

func handleCLIWorkflow(ctx *processContext) {
	printInstructions()

	for {
		cmd := askForInput()

		handleCommand(ctx, cmd)

		if cmd.Result.Exited { // binary exited
			break
		}
	}
}

type LoggerWriter struct{}

func (l LoggerWriter) Write(p []byte) (n int, err error) {
	logger.Verbose("Writing d bytes %d", len(p))
	return os.Stdout.Write(p)
}

var pipe io.ReadCloser

func startBinary(target string) *exec.Cmd {

	cmd := exec.Command(target)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	pipe, _ = cmd.StdoutPipe()

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Ptrace: true,
		Setsid: true,
	}

	cmd.Start()
	err := cmd.Wait()

	if err != nil {
		// arrived at auto-inserted initial breakpoint trap
		logger.Debug("child: %v", err)
		logger.Info("binary started, waiting for command")
	}

	return cmd
}
