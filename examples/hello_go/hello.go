package main

var i int

func main() {
	i = 0
	hello()
}
func hello() {
	print("hello ")
	world()
}
func world() {
	print("world")
	if do(i) > 3 {
		more()
	}
}
func do(value int) int {
	print("!\n")
	return value + 5
}
func more() {
	i = 420
	stuff()
}
func stuff() {
	print("i is ", i, "\n")
}
