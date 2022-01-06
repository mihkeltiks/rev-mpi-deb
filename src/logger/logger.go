package logger

import (
	"fmt"
	"time"
)

type Color string

const (
	cyan  Color = "\033[36m"
	reset Color = "\033[0m"
)

func Info(str string, args ...interface{}) {
	fmt.Print(cyan)
	fmt.Print(timeString())
	fmt.Printf(str, args...)
	fmt.Println(reset)
}

func timeString() string {
	hour, min, sec := time.Now().Clock()

	return fmt.Sprintf("%02d:%02d:%02d ", hour, min, sec)
}
