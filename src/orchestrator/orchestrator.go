package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/checkpoint-restore/go-criu/v7"
	crpc "github.com/checkpoint-restore/go-criu/v7/rpc"
	"github.com/creack/pty"
	"google.golang.org/protobuf/proto"

	"github.com/mihkeltiks/rev-mpi-deb/logger"
	"github.com/mihkeltiks/rev-mpi-deb/orchestrator/checkpointmanager"
	"github.com/mihkeltiks/rev-mpi-deb/orchestrator/cli"
	"github.com/mihkeltiks/rev-mpi-deb/orchestrator/gui"
	"github.com/mihkeltiks/rev-mpi-deb/orchestrator/gui/websocket"
	nodeconnection "github.com/mihkeltiks/rev-mpi-deb/orchestrator/nodeConnection"
	"github.com/mihkeltiks/rev-mpi-deb/rpc"
	"github.com/mihkeltiks/rev-mpi-deb/utils"
	"github.com/mihkeltiks/rev-mpi-deb/utils/command"
)

var NODE_DEBUGGER_PATH = fmt.Sprintf("%s/node-debugger", utils.GetExecutableDir())

const ORCHESTRATOR_PORT = 3490

var checkpoints []string
var rootCheckpointTree checkpointmanager.CheckpointTree
var currentCheckpointTree *checkpointmanager.CheckpointTree
var currentCommandlog checkpointmanager.CommandLog

var pid int
var numProcesses int

func main() {
	logger.SetMaxLogLevel(logger.Levels.Verbose)
	numProcessesCLI, targetPath := cli.ParseArgs()
	numProcesses = numProcessesCLI

	// start goroutine for collecting checkpoint results
	checkpointRecordChan := make(chan rpc.MPICallRecord)
	go startCheckpointRecordCollector(checkpointRecordChan)

	// start rpc server in separate goroutine
	go func() {
		rpc.InitializeServer(ORCHESTRATOR_PORT, func(register rpc.Registrator) {
			register(new(logger.LoggerServer))
			register(nodeconnection.NewNodeReporter(checkpointRecordChan, quit))
		})
	}()

	logger.Info("executing %v as an mpi job with %d processes", targetPath, numProcesses)
	c := criu.MakeCriu()

	// Start the MPI job
	mpiProcess := exec.Command(
		"mpirun",
		"-np",
		fmt.Sprintf("%d", numProcesses),
		NODE_DEBUGGER_PATH,
		targetPath,
		fmt.Sprintf("localhost:%d", ORCHESTRATOR_PORT),
	)

	mpiProcess.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	mpiProcess.Stdout = os.Stdout
	mpiProcess.Stderr = os.Stderr

	err := mpiProcess.Start()
	utils.Must(err)

	defer quit()

	// start the graphical user interface
	// when running with docker, gui must be started on the host
	if !utils.IsRunningInContainer() {
		gui.Start()

		websocket.InitServer()
		websocket.WaitForClientConnection()
	}

	// asyncronously wait for the MPI job to finish
	go func() {
		mpiProcess.Wait()

		if err != nil {
			logger.Error("mpi job exited with: %v", err)
			os.Exit(1)
		}
	}()

	// wait for nodes to finish startup sequence
	connectBackToNodes(numProcesses, false)
	nodeconnection.SaveRegisteredNodes()

	pid = mpiProcess.Process.Pid

	checkpointDir := checkpointCRIU(numProcesses, c, pid, true)
	checkpoints = append(checkpoints, checkpointDir)
	checkpointmanager.AddCheckpointLog()
	websocket.HandleCriuCheckpoint()
	time.Sleep(time.Duration(500) * time.Millisecond)

	rootCheckpointTree = *checkpointmanager.MakeCheckpointTree(
		checkpointmanager.GetCheckpointLog(),
		nil,
		[]*checkpointmanager.CheckpointTree{},
		checkpointDir,
		currentCommandlog)

	currentCheckpointTree = &rootCheckpointTree
	currentCommandlog = *currentCheckpointTree.GetCommandlog()

	cli.PrintInstructions()
	for {
		cmd := cli.AskForInput()
		if cmd.Code == command.Cont || cmd.Code == command.SingleStep || cmd.Code == command.Bpoint {
			currentCommandlog = append(currentCommandlog, *cmd)
		}
		switch cmd.Code {
		case command.Quit:
			quit()
		case command.Help:
			cli.PrintInstructions()
		case command.ListCheckpoints:
			checkpointmanager.ListCheckpoints()
		case command.GlobalRollback:
			handleRollbackSubmission(cmd)
		case command.CheckpointCRIU:
			// logger.Verbose("CURRENT BEFORE")
			// currentCheckpointTree.Print()
			// logger.Verbose("ROOT BEFORE %v", rootCheckpointTree)
			// rootCheckpointTree.Print()

			checkpointDir := checkpointCRIU(numProcesses, c, pid, true)
			checkpoints = append(checkpoints, checkpointDir)
			checkpointmanager.AddCheckpointLog()
			websocket.HandleCriuCheckpoint()
			// currentCheckpointTree = checkpointmanager.MakeCheckpointTree(
			// 	checkpointmanager.GetCheckpointLog(),
			// 	currentCheckpointTree,
			// 	[]*checkpointmanager.CheckpointTree{},
			// 	checkpointDir,
			// 	nil)
			// currentCommandlog = *currentCheckpointTree.GetCommandlog()

			// logger.Verbose("CURRENT %v", currentCheckpointTree)
			// currentCheckpointTree.Print()
			// logger.Verbose("ROOT %v", rootCheckpointTree)
			// rootCheckpointTree.Print()
		case command.RestoreCRIU:
			index := cmd.Argument.(int)
			restoreCriu(checkpoints[index], pid, numProcesses)
			websocket.HandleCriuRestore(index)
			checkpointmanager.SetCheckpointLog(index)
			connectBackToNodes(numProcesses, true)
		case command.ReverseSingleStep:
			calculateReverseStepCommands()
		case command.ReverseCont:
			nodeconnection.GetRegisteredIds()
			calculateReverseContinueCommands(cmd)
		default:
			nodeconnection.HandleRemotely(cmd)
			time.Sleep(time.Second)
		}
	}
}

func calculateReverseStepCommands() {
	if len(checkpoints) == 1 && len(currentCommandlog) == 0 {
		logger.Verbose("Nothing to reverse!")
		return
	}

	var newCommandLog checkpointmanager.CommandLog
	// Copy the contents of checkpointLog into the new map
	lastCommand := command.Command{}
	lastMove := ""
	for i := len(currentCommandlog) - 1; i >= 0; i-- {
		if lastMove == "" {
			switch currentCommandlog[i].Code {
			case command.SingleStep:
				lastMove = "s"
				lastCommand = currentCommandlog[i]
				continue
			case command.Cont:
				lastMove = "c"
				lastCommand = currentCommandlog[i]
				continue
			}
		}
		newCommandLog = append(checkpointmanager.CommandLog{currentCommandlog[i]}, newCommandLog...)
	}
	logger.Verbose("%v", currentCommandlog)
	logger.Verbose("%v", newCommandLog)
	logger.Verbose("%v", lastCommand)

	if lastCommand.Code == command.SingleStep {
		restoreCriu("", pid, numProcesses)
		connectBackToNodes(numProcesses, true)
		for i := 0; i < len(newCommandLog)-1; i++ {
			logger.Verbose("Executing %v", newCommandLog[i])
			nodeconnection.HandleRemotely(&newCommandLog[i])
		}
	}
}

func waitFinish(wg *sync.WaitGroup) {
	defer wg.Done()

	for nodeconnection.GetNodePending(-1) {
	}
	logger.Verbose("DONE WAIT FINISH")
}

func calculateReverseContinueCommands(cmd *command.Command) {
	var wg sync.WaitGroup

	restoreCriu("", pid, numProcesses)
	logger.Verbose("HERE")
	connectBackToNodes(numProcesses, true)
	logger.Verbose("HEREASTILL")
	nodeconnection.HandleRemotely(&command.Command{NodeId: cmd.NodeId, Code: command.Bpoint, Argument: cmd.Argument.(int)})

	wg.Add(1)
	hitcount := reverseContLoop(cmd, false, make([]int, numProcesses), &wg)
	logger.Verbose("HITCOUNT RESULT FIRST LOOOOOOOOOOOOOOOOOOOOOOP %v", hitcount)
	wg.Wait()

	wg.Add(1)
	go waitFinish(&wg)
	wg.Wait()

	restoreCriu("", pid, numProcesses)
	connectBackToNodes(numProcesses, true)

	nodeconnection.HandleRemotely(&command.Command{NodeId: cmd.NodeId, Code: command.Bpoint, Argument: cmd.Argument.(int)})
	wg.Add(1)

	hitcount = reverseContLoop(cmd, true, hitcount, &wg)
	logger.Verbose("DONE AA!")
	wg.Wait()
	logger.Verbose("DONE HERE! %v", hitcount)
}

func reverseContLoop(cmd *command.Command, secondRun bool, hitcount []int, wg *sync.WaitGroup) []int {
	defer wg.Done()
	hitArray := make([]int, numProcesses)
	target := cmd.Argument.(int)
	nodeId := cmd.NodeId

	for i := 0; i < len(currentCommandlog); i++ {
		cmd := &currentCommandlog[i]
		cmd.Print()

		forward := cmd.IsForwardProgressCommand()
		nodeconnection.HandleRemotely(cmd)

		if forward {
			logger.Verbose("AM HERE FINALLY")

			targetNodes := checkResult(target, []int{cmd.NodeId}, []int{nodeId})

			for len(targetNodes) > 0 {
				logger.Verbose("THROUGH CHECK %v", targetNodes)
				logger.Verbose("AA")

				logger.Verbose("BB")
				for i := 0; i < len(targetNodes); i++ {
					logger.Verbose("he!")
					val := targetNodes[i]
					if secondRun && hitcount[val] == hitArray[val] {
						continue
					}
					hitArray[val]++
					if secondRun && hitcount[val] == hitArray[val] {
						continue
					}
					logger.Verbose("CONTINUING NODE %v,", val)
					nodeconnection.HandleRemotely(&command.Command{NodeId: val, Code: command.Cont})
				}
				logger.Verbose("CC")
				if secondRun {
					logger.Verbose("Secondruncheck!")
					done := true

					for i := 0; i < len(hitcount); i++ {
						if hitcount[i] > hitArray[i] {
							done = false
							continue
						}
					}
					if done {
						logger.Verbose("Reverse loop executed successfully!")
						return hitArray
					}
				}
				targetNodes = checkResult(target, targetNodes, targetNodes)
			}
		}
	}
	logger.Verbose("DONE REVERSE LOOP %v", hitArray)
	return hitArray
}

func checkResult(target int, commandNodeIds []int, breakpointNodeIds []int) []int {
	logger.Verbose("WHAT HGOPING ON")
	targetnodes := breakpointNodeIds

	if breakpointNodeIds[0] == -1 {
		targetnodes = nodeconnection.GetRegisteredIds()
	}
	if commandNodeIds[0] == -1 {
		commandNodeIds = nodeconnection.GetRegisteredIds()
	}
	logger.Verbose("ASD %v", commandNodeIds)
	for nodeconnection.GetNodesPending(commandNodeIds) {
		time.Sleep(100 * time.Millisecond)

	}
	logger.Verbose("BSF")

	var newTargetNodes []int
	logger.Verbose("ASAAAAAAD %v,", targetnodes)
	for _, node := range targetnodes {
		logger.Verbose("NODE %v,", node)
		if nodeconnection.GetNodeBreakpoint(node) == target {
			logger.Verbose("NODE HIT %v,", node)
			nodeconnection.SetNodeBreakpoint(node, -1)
			nodeconnection.HandleRemotely(&command.Command{NodeId: node, Code: command.Bpoint, Argument: -target})
			newTargetNodes = append(newTargetNodes, node)
		}
	}
	logger.Verbose("loop done %v", newTargetNodes)
	return newTargetNodes
}

func connectBackToNodes(numProcesses int, attach bool) {
	for nodeconnection.GetRegisteredNodesLen() < numProcesses {
	}
	nodeconnection.ConnectToAllNodes(numProcesses)
	if attach {
		nodeconnection.Attach()
	}
}

func restoreCriu(checkpointDir string, pid int, numProcesses int) *os.File {
	if checkpointDir == "" {
		if len(checkpoints) == 1 {

			checkpointDir = checkpoints[0]
		} else {
			logger.Verbose("ASOINAOSIN")
			checkpointDir = currentCheckpointTree.GetParentCheckpoint().GetCheckpointDir()
		}
	}
	fmt.Println("Restoring %s", checkpointDir)
	nodeconnection.Kill()
	nodeconnection.DisconnectAllNodes()
	nodeconnection.Empty()

	time.Sleep(1 * time.Second)
	if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil {
		fmt.Println("Error killing process:", err)
	}
	syscall.Wait4(pid, nil, 0, nil)
	logger.Info("RESTORING %s", checkpointDir)

	cmd := exec.Command("criu", "restore", "-v4", "--unprivileged", "-o", "restore.log", "-j", "--tcp-established", "-D", checkpointDir)

	f, err := pty.Start(cmd)
	if err != nil {
		logger.Info("ERROR WITH PTY %s", err)
	}
	go func() {
		io.Copy(os.Stdout, f)
	}()

	return f
}

func checkpointCRIU(numProcesses int, c *criu.Criu, pid int, leave_running bool) string {
	nodeconnection.Stop()
	nodeconnection.Detach()
	nodeconnection.Reset()
	nodeconnection.DisconnectAllNodes()
	nodeconnection.Empty()
	time.Sleep(time.Second)

	checkpointDir, err := os.MkdirTemp(fmt.Sprintf("%v/temp", utils.GetExecutableDir()), "cp-*")
	if err != nil {
		logger.Error("Error creating folder, %v", err)
	}

	// Calls CRIU, saves process data to checkpointDir
	Dump(c, strconv.Itoa(pid), false, checkpointDir, "", leave_running)

	time.Sleep(1 * time.Second)

	connectBackToNodes(numProcesses, true)
	return checkpointDir
}

func Dump(c *criu.Criu, pidS string, pre bool, imgDir string, prevImg string, leave_running bool) {
	pid, err := strconv.ParseInt(pidS, 10, 32)
	if err != nil {
		logger.Error("Can't parse pid: %v", err)
	}
	img, err := os.Open(imgDir)
	if err != nil {
		logger.Error("Can't open image dir: %v", err)
	}

	opts := &crpc.CriuOpts{
		Pid:            proto.Int32(int32(pid)),
		ImagesDirFd:    proto.Int32(int32(img.Fd())),
		LogLevel:       proto.Int32(4),
		ShellJob:       proto.Bool(true),
		LogToStderr:    proto.Bool(true),
		LeaveRunning:   proto.Bool(leave_running),
		LogFile:        proto.String("dump.log"),
		ExtUnixSk:      proto.Bool(true),
		TcpEstablished: proto.Bool(true),
		Unprivileged:   proto.Bool(true),
	}

	if prevImg != "" {
		opts.ParentImg = proto.String(prevImg)
		opts.TrackMem = proto.Bool(true)
		time.Sleep(5 * time.Second)
	}

	if pre {
		err = c.PreDump(opts, TestNfy{})
	} else {
		err = c.Dump(opts, TestNfy{})
	}

	if err != nil {
		logger.Error("CRIU error during checkpoint: %v", err)
	}
	img.Close()
}

type TestNfy struct {
	criu.NoNotify
}

func handleRollbackSubmission(cmd *command.Command) {
	pendingRollback := checkpointmanager.SubmitForRollback(cmd.Argument.(string))
	if pendingRollback == nil {
		return
	}

	logger.Info("Following checkpoints scheduled for rollback:")
	logger.Info("%v", pendingRollback)

	commit := cli.AskForRollbackCommit()

	if !commit {
		logger.Verbose("Cancelling pending rollback")
		checkpointmanager.ResetPendingRollback()
		return
	}

	nodeconnection.ExecutePendingRollback()
}

func startCheckpointRecordCollector(
	channel <-chan rpc.MPICallRecord,
) {
	for {
		callRecord := <-channel

		logger.Debug("Node %v reported MPI call: %v", callRecord.NodeId, callRecord.OpName)

		checkpointmanager.RecordCheckpoint(callRecord)
		websocket.SendCheckpointUpdateMessage(checkpointmanager.GetCheckpointLog())
	}
}

func quit() {
	nodeconnection.StopAllNodes()
	gui.Stop()

	time.Sleep(time.Second)
	logger.Info("👋 exiting")
	time.Sleep(time.Second)
	os.Exit(0)
}
