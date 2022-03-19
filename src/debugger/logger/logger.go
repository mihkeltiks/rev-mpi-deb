package logger

import (
	"fmt"
	"time"
)

type color string

type LoggingLevel int

var MAX_LOG_LEVEL = debug

var Levels levelsStruct = levelsStruct{
	Err:     err,
	Warn:    warn,
	Info:    info,
	Verbose: verbose,
	Debug:   debug,
}

const (
	reset  color = "\033[0m"
	cyan   color = "\033[36m"
	blue   color = "\033[34m"
	yellow color = "\033[33m"
	red    color = "\033[31m"
)

const (
	err     LoggingLevel = 1
	warn    LoggingLevel = 2
	info    LoggingLevel = 3
	verbose LoggingLevel = 4
	debug   LoggingLevel = 5
)

type levelsStruct struct {
	Err     LoggingLevel
	Warn    LoggingLevel
	Info    LoggingLevel
	Verbose LoggingLevel
	Debug   LoggingLevel
}

var colorMap map[LoggingLevel]color = map[LoggingLevel]color{
	err:   red,
	warn:  yellow,
	info:  cyan,
	debug: blue,
}

func SetMaxLogLevel(level LoggingLevel) {
	MAX_LOG_LEVEL = level
}

func Error(str string, args ...interface{}) {
	logRow(err, str, args...)
}

func Warn(str string, args ...interface{}) {
	logRow(warn, str, args...)
}

func Info(str string, args ...interface{}) {
	logRow(info, str, args...)
}

func Verbose(str string, args ...interface{}) {
	logRow(info, str, args...)
}

func Debug(str string, args ...interface{}) {
	logRow(debug, str, args...)
}

func logRow(level LoggingLevel, str string, args ...interface{}) {

	if level > MAX_LOG_LEVEL {
		return
	}

	fmt.Print(colorMap[level])
	fmt.Print(timeString())
	fmt.Printf(str, args...)
	fmt.Println(reset)
}

func timeString() string {
	hour, min, sec := time.Now().Clock()

	return fmt.Sprintf("%02d:%02d:%02d  ", hour, min, sec)
}
