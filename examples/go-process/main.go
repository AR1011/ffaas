// GOOS=wasip1 GOARCH=wasm go build -o examples/go-process/process.wasm examples/go-process/main.go
package main

import (
	"time"
)

func main() {
	for {

		//simulating long running process

		// do some work
		time.Sleep(10 * time.Second)

		// do some more work
		time.Sleep(1 * time.Second)

	}
}
