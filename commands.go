package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type command struct {
	code   commandCode
	lineNr int
}

// type Weekday string

// const (
// 	Sunday    Weekday = "Sunday"
// 	Monday    Weekday = "Monday"
// 	Tuesday   Weekday = "Tuesday"
// 	Wednesday Weekday = "Wednesday"
// 	Thursday  Weekday = "Thursday"
// 	Friday    Weekday = "Friday"
// 	Saturday  Weekday = "Saturday"
// )

type commandCode int

const (
	bpoint commandCode = iota
	step
	cont
	quit
)

func (c command) String() string {
	commandStrings := map[commandCode]string{
		bpoint: "bpoint",
		step:   "step",
		cont:   "continue",
		quit:   "quit",
	}

	bpointString := ""
	if c.code == bpoint {
		bpointString = fmt.Sprintf(", line %d", c.lineNr)
	}

	return fmt.Sprintf("Command{%s%s} \n", commandStrings[c.code], bpointString)
}

func askForInput() *command {
	printInstructions()

	command := getCommandFromInput()

	if command == nil {
		fmt.Println("Invalid input")
		return askForInput()
	}

	return command
}

func getCommandFromInput() *command {

	reader := bufio.NewReader(os.Stdin)

	text, _ := reader.ReadString('\n')

	text = strings.Replace(text, "\n", "", 1)

	text = strings.ToLower(text)

	return parseCommandFromString(text)
}

func parseCommandFromString(input string) (c *command) {

	breakPointRegexp := regexp.MustCompile("^b \\d+$")

	switch {
	case breakPointRegexp.Match([]byte(input)):
		lineNr, _ := strconv.Atoi(strings.Split(input, " ")[1])

		return &command{code: bpoint, lineNr: lineNr}

	case input == "c":
		return &command{code: cont}

	case input == "s":
		return &command{code: step}

	case input == "q":
		return &command{code: quit}

	default:
		return nil
	}
}

func printInstructions() {

	fmt.Print("\nAvailable commands:\n\n")

	fmt.Println("  b <lineNr> \t set breakpoint")
	fmt.Println("  s  \t\t single step forward")
	fmt.Println("  c  \t\t continue execution")
	fmt.Println("  q  \t\t quit")
	fmt.Println()
}
