package main

import (
	"io/ioutil"
	"os"
	"path"

	"github.com/ottmartens/cc-rev-db/logger"
)

func precleanup() {
	// remove  artefacts from previous run
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
