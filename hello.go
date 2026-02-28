// Package dlog provides a simple hello world functionality.
package dlog

// Hello represents a simple greeting structure.
type Hello struct{}

// SayHello prints "Hello, world!" to the console.
func (h *Hello) SayHello() {
	println("Hello, world!")
}
