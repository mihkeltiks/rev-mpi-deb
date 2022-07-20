package main

//lint:file-ignore U1000 ignore unused helpers

import (
	"io"
	"os"
	"os/exec"
	"runtime"
	"syscall"

	"github.com/ottmartens/cc-rev-db/command"
	"github.com/ottmartens/cc-rev-db/logger"
	"github.com/ottmartens/cc-rev-db/nodeDebugger/dwarf"
	"github.com/ottmartens/cc-rev-db/rpc"
)

const MAIN_FN = "main"

var cliMode = false
var orchestratorAddress string
var nodeId int

type processContext struct {
	targetFile     string             // the executing binary file
	sourceFile     string             // source code file
	dwarfData      *dwarf.DwarfData   // dwarf debug information about the binary
	process        *exec.Cmd          // the running binary
	pid            int                // the process id of the running binary
	bpointData     breakpointData     // holds the instuctions for currently replaced by breakpoints
	cpointData     checkpointData     // holds data about currently recorded checkppoints
	checkpointMode CheckpointMode     // whether checkpoints are recorded in files or in forked processes
	commandQueue   []*command.Command // commands scheduled for execution by orchestrator
	stack          programStack       // current call stack of the target. updated after each command execution
}

func main() {
	// As ptrace calls depend on per-thread state, we must lock the thread
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	precleanup()

	targetFile, checkpointMode, orchestratorAddress := getValuesFromArgs()

	ctx := &processContext{
		targetFile:     targetFile,
		checkpointMode: checkpointMode,
		bpointData:     breakpointData{}.New(),
		cpointData:     checkpointData{}.New(),
	}

	if !cliMode {
		// connect to orchestrator
		rpc.Client.Connect(orchestratorAddress)

		nodeId = reportAsHealthy()
		logger.SetRemoteClient(rpc.Client, nodeId)

		logger.Info("Process (pid: %d) registered", os.Getpid())
	}

	// parse debugging data
	ctx.dwarfData = dwarf.ParseDwarfData(ctx.targetFile)
	ctx.dwarfData.ResolveMPIDebugInfo(MPI_FUNCS.SIGNATURE)
	ctx.sourceFile = ctx.dwarfData.FindEntrySourceFile(MAIN_FN)

	// start target binary
	ctx.process = startBinary(ctx.targetFile)
	ctx.pid = ctx.process.Process.Pid

	// set up automatic breakpoints
	insertMPIBreakpoints(ctx)

	if cliMode {
		handleCLIWorkflow(ctx)
	} else {
		handleRemoteWorkflow(ctx)
	}

}

func handleRemoteWorkflow(ctx *processContext) {
	port := 3500 + nodeId

	go func() {
		rpc.InitializeServer(port, func(register rpc.Registrator) {
			logger.Verbose("Registering debugging methods for remote use")

			register(&RemoteCmdHandler{ctx})
		})
	}()

	for {
		if len(ctx.commandQueue) > 0 {
			cmd := ctx.commandQueue[0]

			handleCommand(ctx, cmd)

			reportCommandResult(cmd)

			if cmd.Result.Exited {
				logger.Info("Exiting")
				break
			}

			ctx.commandQueue = ctx.commandQueue[1:]
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
	logger.Verbose("Writing d bytes", len(p))
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
