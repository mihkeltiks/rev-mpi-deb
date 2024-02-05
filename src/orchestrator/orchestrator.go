package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
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

func main() {
	logger.SetMaxLogLevel(logger.Levels.Verbose)
	numProcesses, targetPath := cli.ParseArgs()

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
	time.Sleep(time.Second)
	nodeconnection.ConnectToAllNodes(numProcesses)

	time.Sleep(time.Second)

	var checkpoints []string
	cli.PrintInstructions()

	for {
		cmd := cli.AskForInput()

		switch cmd.Code {
		case command.Quit:
			quit()
		case command.Help:
			cli.PrintInstructions()
			break
		case command.ListCheckpoints:
			checkpointmanager.ListCheckpoints()
			break
		case command.GlobalRollback:
			handleRollbackSubmission(cmd)
			break
		case command.CheckpointCRIU:
			nodeconnection.Stop()
			nodeconnection.Detach()
			time.Sleep(1 * time.Second)
			checkpointDir := checkpointCRIU(mpiProcess.Process.Pid, true)
			checkpoints = append(checkpoints, checkpointDir)
			nodeconnection.Attach()
			break
		case command.Stop:
			nodeconnection.Stop()
			break
		case command.Detach:
			nodeconnection.Detach()
			break
		case command.Attach:
			nodeconnection.Attach()
			break
		case command.Kill:
			nodeconnection.Stop()
			nodeconnection.Detach()
			time.Sleep(1 * time.Second)
			checkpointDir := checkpointCRIU(mpiProcess.Process.Pid, false)
			checkpoints = append(checkpoints, checkpointDir)
			// logger.Info("Killing %s", mpiProcess.Process.Pid)
			// nodeconnection.Kill()
			// time.Sleep(1 * time.Second)
			// if err := syscall.Kill(-mpiProcess.Process.Pid, syscall.SIGKILL); err != nil {
			// 	fmt.Println("Error killing process:", err)
			// }
			// syscall.Wait4(mpiProcess.Process.Pid, nil, 0, nil)
			break
		case command.RestoreCRIU:
			_ = restoreCriu(checkpoints[cmd.Argument.(int)])
			time.Sleep(2 * time.Second)
			// nodeconnection.ConnectToAllNodes(numProcesses)
			break
		case command.Connect:
			nodeconnection.Empty()
			nodeconnection.ConnectToAllNodes(numProcesses)
			logger.Info("HERE")
			break
		case command.Disconnect:
			nodeconnection.DisconnectAllNodes()
			break
		case command.Reset:
			nodeconnection.Stop()
			nodeconnection.Detach()
			nodeconnection.HandleRemotely(cmd)
			time.Sleep(1 * time.Second)
			checkpointDir := checkpointCRIU(mpiProcess.Process.Pid, false)
			checkpoints = append(checkpoints, checkpointDir)
			// nodeconnection.Empty()
			// time.Sleep(2 * time.Second)

			// nodeconnection.ConnectToAllNodes(numProcesses)
			break
		default:
			nodeconnection.HandleRemotely(cmd)
			time.Sleep(time.Second)
			break
		}
	}
}

func restoreCriu(checkpointDir string) *os.File {
	logger.Info("RESTORING %s", checkpointDir)

	cmd := exec.Command("/usr/local/sbin/criu", "restore", "-v4", "-o", "restore.log", "-j", "--tcp-established", "-D", checkpointDir)

	f, err := pty.Start(cmd)
	if err != nil {
		logger.Info("ERROR WITH PTY", err)
	}
	logger.Info("RESTORED")

	return f
}

func checkpointCRIU(pid int, leave_running bool) (checkpointDir string) {
	c := criu.MakeCriu()

	checkpointDir, err := os.MkdirTemp(fmt.Sprintf("%v/temp", utils.GetExecutableDir()), "cp-*")
	if err != nil {
		logger.Error("Error creating folder, %v", err)
	}
	logger.Info("Saving checkpoint into: %v", checkpointDir)

	// Calls CRIU, saves process data to checkpointDir
	Dump(c, strconv.Itoa(pid), false, checkpointDir, "", leave_running)

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
