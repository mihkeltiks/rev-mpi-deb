package gui

import (
	"fmt"
	"os/exec"

	"logger"
	"utils"
)

var guiProcess *exec.Cmd

func Start() {
	npmPath, err := exec.LookPath("npm")
	if err != nil {
		logger.Error("Cannot find npm: %v:", err)
		return
	}

	guiDir := fmt.Sprintf("%s/../gui", utils.GetExecutableDir())

	cmd := exec.Cmd{
		Path: npmPath,
		Args: []string{"npm", "run", "start:open"},
		Dir:  guiDir,
	}

	// cmd.Stdout = os.Stdout
	// cmd.Stderr = os.Stderr

	logger.Verbose("Starting the gui - %v", cmd.String())

	err = cmd.Start()

	if err != nil {
		logger.Error("Error starting the graphical user interface: %v", err)
		return
	}

	guiProcess = &cmd
}

func Stop() {
	if guiProcess != nil {
		logger.Info("Stopping gui")

		// unfortunately the gui process is very hard to kill, using this as a workaround
		exec.Command("pkill", "-f", "webpack").Run()
	}
}

func openInBrowser(url string) error {
	return exec.Command("xdg-open", url).Start()
}
