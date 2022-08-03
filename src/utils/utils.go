package utils

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"time"
	"unsafe"
)

func Must(err error) {
	if err != nil {
		panic(fmt.Sprintf("process %d - %v", os.Getpid(), err))
	}
}

// returns the pointer size of current arch
func PtrSize() int {
	return int(unsafe.Sizeof(uintptr(0)))
}

func PrintPSInfo(pid int) {

	cmd := exec.Command("ps", "-Flww", "-p", fmt.Sprint(pid))

	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	Must(err)

	err = cmd.Wait()
	Must(err)
}

func RandomId() string {
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
func GetExecutableDir() string {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	return filepath.Dir(ex)
}

func IsRunningInContainer() bool {
	if _, err := os.Stat("/.dockerenv"); err != nil {
		return false
	}
	return true
}
