// GOOS=wasip1 GOARCH=wasm go build -o examples/go/task/task.wasm examples/go/task/main.go
package main

import (
	"fmt"
	"time"
)

func main() {
	// example of a task

	// do some work
	time.Sleep(10 * time.Second)

	// do some more work
	time.Sleep(1 * time.Second)

	fmt.Println("Finished Task")
}
