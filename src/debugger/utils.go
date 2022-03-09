package main

//lint:file-ignore U1000 ignore unused helpers

import (
	"io/ioutil"
	"os"
	"path"
	"unsafe"

	"github.com/ottmartens/cc-rev-db/logger"
)

type fn func()

const MAIN_FN = "main"

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
	logger.Info("removing temporary files..")

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
