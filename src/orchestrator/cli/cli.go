package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/mihkeltiks/rev-mpi-deb/logger"
	nodeconnection "github.com/mihkeltiks/rev-mpi-deb/orchestrator/nodeConnection"
	"github.com/mihkeltiks/rev-mpi-deb/utils"
	"github.com/mihkeltiks/rev-mpi-deb/utils/command"
)

func ParseArgs() (numProcesses int, targetPath string) {
	args := os.Args

	if len(args) > 3 || len(args) < 2 {
		panicArgs()
	}

	numProcesses, err := strconv.Atoi(args[1])

	if err != nil || numProcesses < 1 {
		panicArgs()
	}

	targetPath = args[2]
	file, err := os.Stat(targetPath)
	utils.Must(err)
	if file.IsDir() {
		panicArgs()
	}

	filepath.EvalSymlinks(targetPath)

	return numProcesses, targetPath
}

func panicArgs() {
	logger.Error("usage: orchestrator <num_processes> <target_file>")
	os.Exit(2)
}

func PrintInstructions() {

	fmt.Print("\nAvailable commands:\n\n")

	fmt.Println("  <nid/all> b <lineNr> \tset breakpoint")
	fmt.Println("  <nid/all> s \t\tsingle-step forward")
	fmt.Println("  <nid/all> rs \t\tsingle-step backward")
	fmt.Println("  <nid/all> c \t\tcontinue execution")
	fmt.Println("  <nid/all> rc \t\tcontinue execution backward")
	fmt.Println("  <nid/all> p <var>  \tprint a variable")
	fmt.Println("        cp  \t\tlist recorded checkpoints")
	fmt.Println("        r <checkpoint id>  \trollback to checkpoint")
	fmt.Println("        cpCRIU  \tissue a CRIU checkpoint")
	fmt.Println("        restoreCRIU <var>  \tissue a CRIU restore")
	fmt.Println("        q  \t\tquit")
	fmt.Println("     help  \t\tshow this again")
	fmt.Println()
	fmt.Printf("  nid (node id) in %v\n", nodeconnection.GetRegisteredIds())
	fmt.Println()
}

func AskForInput() *command.Command {
	PrintPrompt()

	userInput := getUserInputLine()

	command := parseCommandFromString(userInput)

	if command == nil {
		fmt.Println(`Invalid input. Type "help" to see available commands`)
		logger.Debug("CLI")
		return AskForInput()
	}

	return command
}

func getUserInputLine() string {

	reader := bufio.NewReader(os.Stdin)

	text, _ := reader.ReadString('\n')

	text = strings.Replace(text, "\n", "", 1)

	return text
}

func PrintPrompt() {
	fmt.Printf("insert command > ")
}

// a command line prefixed with a pid number
func matchPidRegexp(input string, exp string) bool {

	fullExpr := fmt.Sprintf(`^\d+ %v$`, exp)

	return regexp.MustCompile(fullExpr).Match([]byte(input))
}

func matchAllRegexp(input string, exp string) bool {

	fullExpr := fmt.Sprintf(`^all %v$`, exp)

	return regexp.MustCompile(fullExpr).Match([]byte(input))
}

func AskForRollbackCommit() bool {
	var s string

	fmt.Printf("Commit rollback? (y/n): ")
	_, err := fmt.Scan(&s)
	if err != nil {
		panic(err)
	}

	s = strings.TrimSpace(s)
	s = strings.ToLower(s)

	if s == "y" || s == "yes" {
		return true
	}

	return false
}
