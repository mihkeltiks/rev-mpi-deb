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
var program string

func main() {
	logger.SetMaxLogLevel(logger.Levels.Verbose)
	numProcessesCLI, targetPath, programloc := cli.ParseArgs()
	program=programloc
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

	time.Sleep(1 * time.Second) 

	logger.Info("executing %v as an mpi job with %d processes", targetPath, numProcesses)

	var mpiProcess *exec.Cmd
	var imgDir string
	var err error
	var c *criu.Criu
	// Start the MPI job
	if program=="criu"{
		c = criu.MakeCriu()
		mpiProcess = exec.Command(
			"mpirun",
			"-np",
			fmt.Sprintf("%d", numProcesses),
			NODE_DEBUGGER_PATH,
			targetPath,
			fmt.Sprintf("localhost:%d", ORCHESTRATOR_PORT),
		)
		imgDir=""
	}else if(program=="dmtcp"){
		img := createCpDir();
		//start dmtcp coordinator
		cmd := exec.Command("dmtcp_coordinator") 
		if err := cmd.Run(); err != nil {
			logger.Error("dmtcp_coordinator exited with: %v", err)
		}
		mpiProcess = exec.Command(
			"mpirun",
			"-np",
			fmt.Sprintf("%d", numProcesses),
			"dmtcp_launch.py",
			"--ckptdir",
			img.Name(),
			NODE_DEBUGGER_PATH,
			targetPath,
			fmt.Sprintf("localhost:%d", ORCHESTRATOR_PORT),
		)	
	}

	mpiProcess.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	mpiProcess.Stdout = os.Stdout
	mpiProcess.Stderr = os.Stderr

	err = mpiProcess.Start()
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
	var wg sync.WaitGroup
	wg.Add(1)
	go connectBackToNodes(numProcesses, false, &wg)
	wg.Wait()
	nodeconnection.SaveRegisteredNodes()

	pid = mpiProcess.Process.Pid

	checkpointDir := checkpoint(imgDir, c)
	checkpoints = append(checkpoints, checkpointDir)
	checkpointmanager.AddCheckpointLog()
	websocket.HandleCriuCheckpoint()
	time.Sleep(time.Duration(200) * time.Millisecond)

	rootCheckpointTree = *checkpointmanager.MakeCheckpointTree(
		nil,
		nil,
		[]*checkpointmanager.CheckpointTree{},
		checkpointDir,
		nil,
		make([]int, numProcesses))

	currentCheckpointTree = &rootCheckpointTree

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
		case command.Checkpoint:
			checkpointDir = checkpoint(imgDir, c)

			checkpoints = append(checkpoints, checkpointDir)
			checkpointmanager.AddCheckpointLog()

			websocket.HandleCriuCheckpoint()

			currentCheckpointTree = checkpointmanager.MakeCheckpointTree(
				checkpointmanager.GetCheckpointLog(),
				currentCheckpointTree,
				[]*checkpointmanager.CheckpointTree{},
				checkpointDir,
				currentCommandlog,
				nodeconnection.GetAllNodeCounters())

			currentCheckpointTree.GetParentTree().AddChildTree(currentCheckpointTree)

			currentCommandlog = []command.Command{}

		case command.GRestore:
			index := cmd.Argument.(int)

			restore(checkpoints[index], pid, numProcesses)

			websocket.HandleCriuRestore(index)
			checkpointmanager.SetCheckpointLog(index)

			currentCheckpointTree = findTreeByDir(&rootCheckpointTree, checkpoints[index])
			currentCommandlog = []command.Command{}

			var wg sync.WaitGroup
			wg.Add(1)
			connectBackToNodes(numProcesses, true, &wg)
			wg.Wait()

		case command.ReverseSingleStep:
			calculateReverseStepCommands(cmd)
		case command.ReverseCont:
			calculateReverseContinueCommands(cmd)
		default:
			nodeconnection.HandleRemotely(cmd)
			time.Sleep(time.Second)
		}
	}
}
func createCpDir() *os.File {
			//we create the checkpoint dir
			imgDir, err := os.MkdirTemp(fmt.Sprintf("%v/temp", utils.GetExecutableDir()), "cp-*")
		
			if err != nil {
				logger.Error("Error creating folder, %v", err)
			}
			img, err := os.Open(imgDir)//, os.O_RDWR|os.O_CREATE, 0644)
			if err != nil {
				logger.Error("Can't open image dir: %v", err)
			}
			return img	
}
func findTreeByDir(tree *checkpointmanager.CheckpointTree, dir string) *checkpointmanager.CheckpointTree {
	if dir == tree.GetCheckpointDir() {
		return tree
	}
	for _, child := range tree.GetChildrenTrees() {
		if child.GetCheckpointDir() == dir {
			return child
		}
		recurse := findTreeByDir(child, dir)
		if recurse != nil {
			return recurse
		}
	}
	return nil
}
func calculateReverseStepCommands(cmd *command.Command) {
	nodeconnection.HandleRemotely(&command.Command{NodeId: -1, Code: command.RetrieveBreakpoints})
	nodeconnection.HandleRemotely(&command.Command{NodeId: -1, Code: command.Retrieve, Argument: "counter"})
	counters := nodeconnection.GetAllNodeCounters()

	for {
		stop := true

		for _, counter := range counters {
			if counter == -1 {
				stop = false
			}
		}
		if stop {
			break
		}
		time.Sleep(50 * time.Millisecond)
		counters = nodeconnection.GetAllNodeCounters()
	}
	bpmap := make(map[int][]int)
	for i := 0; i < numProcesses; i++ {
		bpmap[i] = nodeconnection.GetBreakpoints(i)
	}

	// tree, _ := findTreeCandidateCounter(cmd, *currentCheckpointTree)
	restore(rootCheckpointTree.GetCheckpointDir(), pid, numProcesses)
	websocket.HandleCriuRestore(0)
	checkpointmanager.SetCheckpointLog(0)

	var gg sync.WaitGroup
	gg.Add(1)
	connectBackToNodes(numProcesses, true, &gg)
	gg.Wait()

	if cmd.NodeId == -1 {
		for index := range counters {
			counters[index] -= 1
		}
	} else {
		counters[cmd.NodeId] -= 1
	}

	nodeconnection.HandleRemotely(&command.Command{NodeId: -1, Code: command.RemoveBreakpoints})

	for index, counter := range counters {
		nodeconnection.HandleRemotely(&command.Command{NodeId: index, Code: command.Insert, Argument: counter})
		nodeconnection.HandleRemotely(&command.Command{NodeId: index, Code: command.Cont})
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go waitFinish(&wg)
	wg.Wait()

	for index := range counters {
		nodeconnection.HandleRemotely(&command.Command{NodeId: index, Code: command.Insert, Argument: 2000000})
	}
	for i := 0; i < numProcesses; i++ {
		nodeconnection.HandleRemotely(&command.Command{NodeId: i, Code: command.ChangeBreakpoints, Argument: bpmap[i]})
	}

	nodeconnection.ResetAllNodeCounters()
}

func calculateReverseContinueCommands(cmd *command.Command) {
	nodeconnection.HandleRemotely(&command.Command{NodeId: -1, Code: command.RetrieveBreakpoints})
	nodeconnection.HandleRemotely(&command.Command{NodeId: -1, Code: command.Retrieve, Argument: "counter"})
	counters := nodeconnection.GetAllNodeCounters()

	for {
		stop := true

		for _, counter := range counters {
			if counter == -1 {
				stop = false
			}
		}
		if stop {
			break
		}
		time.Sleep(50 * time.Millisecond)
		counters = nodeconnection.GetAllNodeCounters()
	}

	bpmap := make(map[int][]int)
	for i := 0; i < numProcesses; i++ {
		bpmap[i] = nodeconnection.GetBreakpoints(i)
	}

	// tree, _ := findTreeCandidateCounter(cmd, *currentCheckpointTree)
	breakpointHitMap := reverseContLoop(cmd, rootCheckpointTree.GetCheckpointDir(), counters, bpmap, nil, false)
	reverseContLoop(cmd, rootCheckpointTree.GetCheckpointDir(), counters, bpmap, breakpointHitMap, true)

	// Remove the breakpoint that was hit
	for i := 0; i < numProcesses; i++ {
		if cmd.NodeId == i || cmd.NodeId == -1 {
			length := len(breakpointHitMap[i])
			if length > 0 {
				bpmap[i] = removeElement(bpmap[i], breakpointHitMap[i][length-1])
			}
		}
	}
	nodeconnection.HandleRemotely(&command.Command{NodeId: cmd.NodeId, Code: command.Insert, Argument: 2000000})

}

func reverseContLoop(cmd *command.Command, checkpointDirRestore string, counters []int, bpmap map[int][]int, firstRunhitMap map[int][]int, secondRun bool) map[int][]int {
	checkpointmanager.SetCheckpointLog(0)
	websocket.HandleCriuRestore(0)
	restore(checkpointDirRestore, pid, numProcesses)
	var wg sync.WaitGroup
	wg.Add(1)
	connectBackToNodes(numProcesses, true, &wg)
	wg.Wait()
	// Set new breakpoints
	for i := 0; i < numProcesses; i++ {
		nodeconnection.HandleRemotely(&command.Command{NodeId: i, Code: command.RemoveBreakpoints})
		if cmd.NodeId == i || cmd.NodeId == -1 {
			nodeconnection.HandleRemotely(&command.Command{NodeId: i, Code: command.ChangeBreakpoints, Argument: bpmap[i]})
		}
	}

	// Initialize targets and continue
	for index, counter := range counters {
		nodeconnection.HandleRemotely(&command.Command{NodeId: index, Code: command.Insert, Argument: counter - 1})
		nodeconnection.HandleRemotely(&command.Command{NodeId: index, Code: command.Cont, Argument: 1})
	}
	breakpointHitMap := make(map[int][]int)
	for i := 0; i < numProcesses; i++ {
		breakpointHitMap[i] = []int{}
	}
	if cmd.NodeId == -1 {
		var unCompletedNodes []int
		for i := 1; i < numProcesses; i++ {
			unCompletedNodes = append(unCompletedNodes, i)
		}

		for len(unCompletedNodes) != 0 {
			node := nodeconnection.GetReadyNode()
			for node == -1 {
				time.Sleep(10 * time.Millisecond)
				node = nodeconnection.GetReadyNode()
			}
			breakpointHit := nodeconnection.GetNodeBreakpoint(node)
			if secondRun && len(breakpointHitMap[node]) == len(firstRunhitMap[node])-1 {
				unCompletedNodes = removeElement(unCompletedNodes, node)
			} else if !secondRun && breakpointHit == -5 {
				unCompletedNodes = removeElement(unCompletedNodes, node)
			} else {
				nodeconnection.HandleRemotely(&command.Command{NodeId: node, Code: command.Bpoint, Argument: -breakpointHit})
				breakpointHitMap[node] = append(breakpointHitMap[node], breakpointHit)
				nodeconnection.HandleRemotely(&command.Command{NodeId: node, Code: command.Cont, Argument: 1})
			}
		}

	} else {
		for {
			for nodeconnection.GetNodePending(cmd.NodeId) {
				time.Sleep(10 * time.Millisecond)
			}
			breakpointHit := nodeconnection.GetNodeBreakpoint(cmd.NodeId)

			if secondRun && len(breakpointHitMap[cmd.NodeId]) == len(firstRunhitMap[cmd.NodeId])-1 {
				break
			} else if !secondRun && breakpointHit == -5 {
				break
			} else {
				nodeconnection.HandleRemotely(&command.Command{NodeId: cmd.NodeId, Code: command.Bpoint, Argument: -breakpointHit})
				breakpointHitMap[cmd.NodeId] = append(breakpointHitMap[cmd.NodeId], breakpointHit)
			}
			nodeconnection.HandleRemotely(&command.Command{NodeId: cmd.NodeId, Code: command.Cont, Argument: 1})
		}
	}
	return breakpointHitMap
}

func removeElement(array []int, value int) []int {
	// Initialize a new slice to hold the result
	var result []int

	// Iterate over the original array
	for _, elem := range array {
		// If the element is not the one to be removed, append it to the result slice
		if elem != value {
			result = append(result, elem)
		}
	}

	// Return the result slice
	return result
}

func waitFinish(wg *sync.WaitGroup) {
	defer wg.Done()

	for nodeconnection.GetNodePending(-1) {
	}
	// logger.Verbose("DONE WAIT FINISH")
}

func checkCounters(counters []int, nodeId int) bool {
	currentCounters := nodeconnection.GetAllNodeCounters()
	if nodeId == -1 {
		for index, counter := range counters {
			if currentCounters[index]-counter == 0 {
				return false
			}
		}
		return true
	}
	return currentCounters[nodeId]-counters[nodeId] > 0

}
func findTreeCandidateCounter(cmd *command.Command, tree checkpointmanager.CheckpointTree) (result checkpointmanager.CheckpointTree, counters []int) {
	if checkCounters(currentCheckpointTree.GetCounters(), cmd.NodeId) {
		return *currentCheckpointTree, currentCheckpointTree.GetCounters()
	}
	tree.Print()
	for tree.HasParent() {
		tree = *tree.GetParentTree()
		if checkCounters(tree.GetCounters(), cmd.NodeId) {
			return tree, tree.GetCounters()
		}

	}
	return tree, nil
}

func connectBackToNodes(numProcesses int, attach bool, wg *sync.WaitGroup) {
	defer wg.Done()

	for nodeconnection.GetRegisteredNodesLen() < numProcesses {
	}
	nodeconnection.ConnectToAllNodes(numProcesses)
	if attach {
		nodeconnection.Attach()
	}
	// logger.Verbose("DONE WITH CONNECT")
}


func restore(checkpointDir string, pid int, numProcesses int) *os.File {
	if checkpointDir == "" {
		if len(checkpoints) == 1 {
			checkpointDir = checkpoints[0]
		} else {
			checkpointDir = currentCheckpointTree.GetParentTree().GetCheckpointDir()
		}
	}
	nodeconnection.Kill()
	nodeconnection.DisconnectAllNodes()
	nodeconnection.Empty()

	if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil {
		fmt.Println("Error killing process:", err)
	}
	syscall.Wait4(pid, nil, 0, nil)
	// logger.Info("RESTORING %s", checkpointDir)

	if program=="criu"{
		return restoreCriu(checkpointDir, pid, numProcesses);
	}else{ //} if(program=="dmtcp"){
		return restoreDmtcp(checkpointDir, pid, numProcesses);
	}
}

func restoreDmtcp(checkpointDir string, pid int, numProcesses int) *os.File {
	img := createCpDir();
	cmd := exec.Command("dmtcp_restart.py", "--ckptdir", img.Name(), "--restartdir", checkpointDir) //"--tcp-established",

	f, err := pty.Start(cmd)
	if err != nil {
		logger.Info("ERROR WITH PTY %s", err)
	}
	go func() {
		io.Copy(os.Stdout, f)
	}()

	return f
}

func restoreCriu(checkpointDir string, pid int, numProcesses int) *os.File {

	cmd := exec.Command("criu", "restore", "-v4", "--unprivileged", "-o", "restore.log", "-j", "-D", checkpointDir) //"--tcp-established",

	f, err := pty.Start(cmd)
	if err != nil {
		logger.Info("ERROR WITH PTY %s", err)
	}
	go func() {
		io.Copy(os.Stdout, f)
	}()

	return f
}

func checkpoint(checkpointDir string, c *criu.Criu) string{
	if program=="criu"{
		return checkpointCRIU(numProcesses, c, pid, true)
	}else{ // if program=="dmtcp"
		return checkpointDmtcp(checkpointDir)
	}
}

func checkpointDmtcp(checkpointDir string) string{
	cmd := exec.Command("dmtcp_command", "--checkpoint") 
	if err := cmd.Run(); err != nil {
		logger.Error("dmtcp_command exited with: %v", err)
	}
	return checkpointDir
}

func checkpointCRIU(numProcesses int, c *criu.Criu, pid int, leave_running bool) string {
	nodeconnection.Stop()
	nodeconnection.Detach()
	nodeconnection.Reset()
	nodeconnection.DisconnectAllNodes()
	nodeconnection.Empty()
	// time.Sleep(1000 * time.Millisecond)

	imgDir := createCpDir()
	/*checkpointDir, err := os.MkdirTemp(fmt.Sprintf("%v/temp", utils.GetExecutableDir()), "cp-*")
	if err != nil {
		logger.Error("Error creating folder, %v", err)
	}*/
	logger.Info(imgDir.Name())

	// Calls CRIU, saves process data to checkpointDir
	Dump(c, strconv.Itoa(pid), false, imgDir.Name(), "", leave_running)

	var wg sync.WaitGroup
	wg.Add(1)
	go connectBackToNodes(numProcesses, true, &wg)
	wg.Wait()
	// logger.Verbose("UPPER CP FINISH")
	return imgDir.Name()
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
		Pid:          proto.Int32(int32(pid)),
		ImagesDirFd:  proto.Int32(int32(img.Fd())),
		LogLevel:     proto.Int32(4),
		ShellJob:     proto.Bool(true),
		LogToStderr:  proto.Bool(true),
		LeaveRunning: proto.Bool(leave_running),
		LogFile:      proto.String("dump.log"),
		// ExtUnixSk:    proto.Bool(true),
		// TcpEstablished: proto.Bool(true),
		Unprivileged: proto.Bool(true),
		GhostLimit:   proto.Uint32(1048576 * 64),
	}

	logger.Info(imgDir)

	if prevImg != "" {
		opts.ParentImg = proto.String(prevImg)
		opts.TrackMem = proto.Bool(true)
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
	// logger.Verbose("LOWER CP FINISH")

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

	logger.Info("👋 exiting")
	os.Exit(0)
}

// func reverseContLoop(cmd *command.Command, secondRun bool, hitcount []int, wg *sync.WaitGroup, log checkpointmanager.CommandLog) []int {
// 	defer wg.Done()
// 	hitArray := make([]int, numProcesses)
// 	target := cmd.Argument.(int)
// 	nodeId := cmd.NodeId

// 	for i := 0; i < len(currentCommandlog); i++ {
// 		cmd := &currentCommandlog[i]
// 		cmd.Print()

// 		forward := cmd.IsForwardProgressCommand()
// 		nodeconnection.HandleRemotely(cmd)

// 		if forward {
// 			logger.Verbose("AM HERE FINALLY")

// 			targetNodes := checkResult(target, []int{cmd.NodeId}, []int{nodeId})

// 			for len(targetNodes) > 0 {
// 				logger.Verbose("THROUGH CHECK %v", targetNodes)
// 				logger.Verbose("AA")

// 				logger.Verbose("BB")
// 				for i := 0; i < len(targetNodes); i++ {
// 					logger.Verbose("he!")
// 					val := targetNodes[i]
// 					if secondRun && hitcount[val] == hitArray[val] {
// 						continue
// 					}
// 					hitArray[val]++
// 					if secondRun && hitcount[val] == hitArray[val] {
// 						continue
// 					}
// 					logger.Verbose("CONTINUING NODE %v,", val)
// 					nodeconnection.HandleRemotely(&command.Command{NodeId: val, Code: command.Cont})
// 				}
// 				logger.Verbose("CC")
// 				if secondRun {
// 					logger.Verbose("Secondruncheck!")
// 					done := true

// 					for i := 0; i < len(hitcount); i++ {
// 						if hitcount[i] > hitArray[i] {
// 							done = false
// 							continue
// 						}
// 					}
// 					if done {
// 						logger.Verbose("Reverse loop executed successfully!")
// 						return hitArray
// 					}
// 				}
// 				targetNodes = checkResult(target, targetNodes, targetNodes)
// 			}
// 		}
// 	}
// 	logger.Verbose("DONE REVERSE LOOP %v", hitArray)
// 	return hitArray
// }
// func calculateReverseContinueCommands(cmd *command.Command) {
// 	var wg sync.WaitGroup
// 	tree, log := findTreeCandidate(cmd, *currentCheckpointTree)
// 	logger.Verbose("HE")
// 	restoreCriu(tree.GetCheckpointDir(), pid, numProcesses)
// 	logger.Verbose("HERE")
// 	connectBackToNodes(numProcesses, true)
// 	logger.Verbose("HEREASTILL")

// 	nodeconnection.HandleRemotely(&command.Command{NodeId: cmd.NodeId, Code: command.Bpoint, Argument: cmd.Argument.(int)})

// 	wg.Add(1)
// 	hitcount := reverseContLoop(cmd, false, make([]int, numProcesses), &wg, log)
// 	if cmd.NodeId == -1 {
// 		found := false
// 		for _, hit := range hitcount {
// 			if hit != 0 {
// 				found = true
// 				break
// 			}
// 		}
// 		if !found {
// 			logger.Verbose("DIDNT FIND SHIT")
// 		}
// 	} else {
// 		if hitcount[cmd.NodeId] == 0 {
// 			logger.Verbose("DIDNT FIND SHIT")

// 		}
// 	}
// 	logger.Verbose("HITCOUNT RESULT FIRST LOOOOOOOOOOOOOOOOOOOOOOP %v", hitcount)
// 	wg.Wait()

// 	wg.Add(1)
// 	go waitFinish(&wg)
// 	wg.Wait()

// 	restoreCriu(tree.GetCheckpointDir(), pid, numProcesses)
// 	connectBackToNodes(numProcesses, true)

// 	nodeconnection.HandleRemotely(&command.Command{NodeId: cmd.NodeId, Code: command.Bpoint, Argument: cmd.Argument.(int)})
// 	wg.Add(1)

// 	hitcount = reverseContLoop(cmd, true, hitcount, &wg, log)
// 	currentCheckpointTree = &tree
// 	currentCommandlog = log
// 	wg.Wait()
// 	logger.Verbose("DONE HERE! %v", hitcount)
// }

// func findTreeCandidate(cmd *command.Command, tree checkpointmanager.CheckpointTree) (result checkpointmanager.CheckpointTree, log checkpointmanager.CommandLog) {
// 	logger.Verbose("he")
// 	if checkTreeForProgress(cmd, currentCommandlog) {
// 		return *currentCheckpointTree, currentCommandlog
// 	}
// 	logger.Verbose("hehe")
// 	tree.Print()
// 	for tree.HasParent() {
// 		logger.Verbose("hehehe")
// 		treecmdlog := tree.GetCommandlog()
// 		logger.Verbose("hehehehe")
// 		log = append(*treecmdlog, log...)
// 		tree = *tree.GetParentTree()
// 		logger.Verbose("hehehehe")
// 		if checkTreeForProgress(cmd, currentCommandlog) {
// 			return tree, log
// 		}
// 		logger.Verbose("hehehehe")
// 	}
// 	return tree, nil
// }

// func checkTreeForProgress(cmd *command.Command, commandLog checkpointmanager.CommandLog) bool {
// 	for i := len(commandLog) - 1; i >= 0; i-- {
// 		checkCmd := currentCommandlog[i]
// 		if checkCmd.IsForwardProgressCommand() && (checkCmd.NodeId == -1 || checkCmd.NodeId == cmd.NodeId) {
// 			return true
// 		}
// 	}
// 	return false
// }

// func checkResult(target int, commandNodeIds []int, breakpointNodeIds []int) []int {
// 	logger.Verbose("WHAT HGOPING ON")
// 	targetnodes := breakpointNodeIds

// 	if breakpointNodeIds[0] == -1 {
// 		targetnodes = nodeconnection.GetRegisteredIds()
// 	}
// 	if commandNodeIds[0] == -1 {
// 		commandNodeIds = nodeconnection.GetRegisteredIds()
// 	}
// 	logger.Verbose("ASD %v", commandNodeIds)
// 	for nodeconnection.GetNodesPending(commandNodeIds) {
// 		time.Sleep(100 * time.Millisecond)

// 	}
// 	logger.Verbose("BSF")

// 	var newTargetNodes []int
// 	logger.Verbose("ASAAAAAAD %v,", targetnodes)
// 	for _, node := range targetnodes {
// 		logger.Verbose("NODE %v,", node)
// 		if nodeconnection.GetNodeBreakpoint(node) == target {
// 			logger.Verbose("NODE HIT %v,", node)
// 			nodeconnection.SetNodeBreakpoint(node, -1)
// 			nodeconnection.HandleRemotely(&command.Command{NodeId: node, Code: command.Bpoint, Argument: -target})
// 			newTargetNodes = append(newTargetNodes, node)
// 		}
// 	}
// 	logger.Verbose("loop done %v", newTargetNodes)
// 	return newTargetNodes
// }
