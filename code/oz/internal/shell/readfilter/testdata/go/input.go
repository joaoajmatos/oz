package main

// This file exists to test comment stripping.
// It intentionally has many comments.
// comment 01
// comment 02
// comment 03
// comment 04
// comment 05
// comment 06
// comment 07
// comment 08
// comment 09
// comment 10
// comment 11
// comment 12
// comment 13
// comment 14
// comment 15
// comment 16
// comment 17
// comment 18
// comment 19
// comment 20

import "fmt"

/*
multiline comment
that should be removed
*/
func main() {
	// print output
	// and more comments
	// and more comments
	// and more comments
	// and more comments
	// and more comments
	fmt.Println("hello")
}
