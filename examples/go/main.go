package main

import (
	"net/http"

	raptor "github.com/anthdm/raptor/sdk"
	"github.com/go-chi/chi/v5"
)

func handleReq(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hello World"))
}

func main() {
	chi := chi.NewRouter()
	chi.Get("/", handleReq)
	raptor.Handle(chi)
}
