package main

import (
	"fmt"
	"math"
)

var i int

func main() {
	i = 0
	hello()
}
func hello() {
	fmt.Print("hello ")
	world()
}
func world() {
	fmt.Print("world")
	if do(i) > 3 {
		more()
	}
}
func do(value int) int {
	fmt.Print("!\n")
	return value + 5
}
func more() {
	i = math.MinInt + math.MaxInt
	stuff()
}
func stuff() {
	fmt.Printf("i is %d\n", i)
}
