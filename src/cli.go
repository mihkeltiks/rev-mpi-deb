package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func askForInput() *command {
	printPrompt()

	userInput := getUserInputLine()

	command := parseCommandFromString(userInput)

	if command == nil {
		fmt.Println(`Invalid input. Type "help" to see available commands`)
		return askForInput()
	}

	return command
}

// returns lowercase string user inputted from the cli
func getUserInputLine() string {

	reader := bufio.NewReader(os.Stdin)

	text, _ := reader.ReadString('\n')

	text = strings.Replace(text, "\n", "", 1)

	text = strings.ToLower(text)

	return text
}

func printPrompt() {
	fmt.Printf("insert command > ")
}

func printInstructions() {

	fmt.Print("\nAvailable commands:\n\n")

	fmt.Println("  b <lineNr> \t set breakpoint")
	fmt.Println("  s  \t\t single step forward")
	fmt.Println("  c  \t\t continue execution")
	fmt.Println("  p <var>  \t print a variable")
	fmt.Println("  q  \t\t quit")
	fmt.Println("  help  \t show this again")
	fmt.Println()
}
