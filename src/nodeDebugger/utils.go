package main

//lint:file-ignore U1000 ignore unused helpers

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"time"
	"unsafe"

	"github.com/ottmartens/cc-rev-db/logger"
)

type fn func()

func must(err error) {
	if err != nil {
		panic(fmt.Sprintf("process %d - %v", os.Getpid(), err))
	}
}

// returns the pointer size of current arch
func ptrSize() int {
	return int(unsafe.Sizeof(uintptr(0)))
}

func executeOnProcess(ctx *processContext, targetPid int, function fn) {
	realPid := ctx.pid

	ctx.pid = targetPid

	function()

	ctx.pid = realPid
}

func cleanup() {
	removeTempFiles()
}

func precleanup() {
	// remove unexpected artefacts from previous run
	removeTempFiles()
}

func removeTempFiles() {
	logger.Debug("removing temporary files..")

	dir, _ := ioutil.ReadDir("bin/temp")

	for _, d := range dir {
		if d.Name() != ".gitkeep" {
			os.Remove(path.Join("bin/temp", d.Name()))
		}
	}
}

func PrintPSInfo(pid int) {

	cmd := exec.Command("ps", "-Flww", "-p", fmt.Sprint(pid))

	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	must(err)

	err = cmd.Wait()
	must(err)
}

func randomId() string {
	rand.Seed(time.Now().UnixNano())

	length := 10
	var letters = []rune("0123456789abcdefghijklmnopqrstuvwxyz")

	runes := make([]rune, length)
	for i := range runes {
		runes[i] = letters[rand.Intn(len(letters))]
	}
	return string(runes)
}

// Returns the directory of the currently running executable
func getExecutableDir() string {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	return filepath.Dir(ex)
}
