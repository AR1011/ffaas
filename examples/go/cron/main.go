// GOOS=wasip1 GOARCH=wasm go build -o examples/go-cron/cron.wasm examples/go-cron/main.go
package main

import (
	"fmt"
	"time"
)

func main() {
	// example of a cron job

	// do some work
	time.Sleep(10 * time.Second)

	// do some more work
	time.Sleep(1 * time.Second)

	fmt.Println("Finished Task")
}
