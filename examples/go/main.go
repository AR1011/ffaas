package main

import (
	"fmt"
	"net/http"

	ffaas "github.com/anthdm/ffaas/sdk"
)

func myHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("hello from tinder swiper")
	w.Write([]byte("from tinder swiper"))
}

func main() {
	ffaas.HandleFunc(myHandler)
}
