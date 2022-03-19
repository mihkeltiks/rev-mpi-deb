package main

//lint:file-ignore U1000 ignore unused helpers

import (
	"io/ioutil"
	"os"
	"path"
	"time"
	"unsafe"

	"github.com/ottmartens/cc-rev-db/logger"
)

type fn func()

func must(err error) {
	if err != nil {
		panic(err)
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
	logger.Debug("removing temporary files..")

	removeTempFiles()
}

func precleanup() {
	// remove unexpected artefacts from previous run
	removeTempFiles()
}

func removeTempFiles() {

	// could use os.TempDir()
	dir, _ := ioutil.ReadDir("bin/temp")

	for _, d := range dir {
		if d.Name() != ".gitkeep" {
			os.RemoveAll(path.Join([]string{"bin/temp", d.Name()}...))
		}

	}

}

// wait while preventing the thread from sleeping
// (contrary to time.Sleep() which causes issues with ptrace)
func waitWithoutSleep(d time.Duration) {
	start := time.Now().UnixNano()
	logger.Warn("starting wait")
	for {

		if time.Now().UnixNano() > start+int64(d) {
			break
		}
	}
	logger.Warn("ended wait")
}
