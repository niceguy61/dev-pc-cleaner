package ui

import "fmt"

// Println writes a line using a Windows-friendly line break.
func Println(s string) {
	fmt.Print(s + lineBreak())
}
