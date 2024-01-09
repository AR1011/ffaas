package main

import (
	"fmt"
	"net/http"

	raptor "github.com/anthdm/raptor/sdk"
	"github.com/go-chi/chi/v5"
)

func handleReq(w http.ResponseWriter, r *http.Request) {
	fmt.Println("can u see me")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("yo from handle"))
}

func main() {
	chi := chi.NewRouter()
	chi.Get("/", handleReq)
	raptor.Handle(chi)
}
