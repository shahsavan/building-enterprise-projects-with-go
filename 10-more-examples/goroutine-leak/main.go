// You can edit this code!
// Click here and start typing.
package main

import (
	"fmt"
	"runtime"
	"time"
)

func main() {
	leakyDispatcher()
}

// leakyDispatcher shows a goroutine leak in a transport dispatcher.
// A sender keeps pushing route IDs but we stop receiving, so the goroutine stays blocked.
func leakyDispatcher() {
	fmt.Println("goroutines before leak:", runtime.NumGoroutine())

	routes := make(chan string)

	// Dispatch loop that never stops sending.
	go func() {
		for i := 0; ; i++ {
			routes <- fmt.Sprintf("route-%02d", i) // blocks when no receiver exists
		}
	}()

	// Consumer stops early, leaving the dispatcher goroutine blocked on send.
	for r := range routes {
		fmt.Println("assigned:", r)
		if r == "route-05" {
			break
		}
	}

	time.Sleep(time.Second)
	fmt.Println("goroutines after leak:", runtime.NumGoroutine())
}
