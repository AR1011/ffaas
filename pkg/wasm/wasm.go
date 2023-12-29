package wasm

import (
	"encoding/json"
	"net/http"

	"github.com/anthdm/ffaas/pkg/api"
	"github.com/anthdm/ffaas/pkg/runtime"
	"github.com/anthdm/ffaas/pkg/storage"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/tetratelabs/wazero"
)

// Server is an HTTP server that will proxy and route the request to the corresponding function.
type Server struct {
	router *chi.Mux
	store  storage.Store
	cache  storage.ModCacher
}

// NewServer return a new wasm server given a storage and a mod cache.
func NewServer(store storage.Store, cache storage.ModCacher) *Server {
	return &Server{
		router: chi.NewRouter(),
		store:  store,
		cache:  cache,
	}
}

// Listen starts listening on the given address.
func (s *Server) Listen(addr string) error {
	s.initRoutes()
	return http.ListenAndServe(addr, s.router)
}

func (s *Server) initRoutes() {
	s.router.Handle("/{appID}", api.UseTimerMiddleware(http.HandlerFunc(s.handleRequest)))
}

// temp
type handleRequestResponse struct {
	Error string `json:"error"`
}

func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
	requestID := uuid.New()
	w.Header().Set("X-Request-ID", requestID.String())

	appID, err := uuid.Parse(chi.URLParam(r, ("appID")))
	if err != nil {
		writeJson(w, http.StatusNotFound, handleRequestResponse{
			Error: err.Error(),
		})
		return
	}
	app, err := s.store.GetApplication(appID)
	if err != nil {
		writeJson(w, http.StatusNotFound, handleRequestResponse{
			Error: err.Error(),
		})
		return
	}
	if !app.HasActiveDeploy() {
		writeJson(w, http.StatusNotFound, handleRequestResponse{
			Error: "no active deploy",
		})
		return

	}
	deploy, err := s.store.GetDeploy(app.ActiveDeployID)
	if err != nil {
		writeJson(w, http.StatusNotFound, handleRequestResponse{
			Error: err.Error(),
		})
		return
	}
	compCache, ok := s.cache.Get(app.ID)
	if !ok {
		compCache = wazero.NewCompilationCache()
		s.cache.Put(app.ID, compCache)
	}
	reqPlugin, err := runtime.NewRequestModule(r)
	if err != nil {
		writeJson(w, http.StatusInternalServerError, handleRequestResponse{
			Error: err.Error(),
		})
		return
	}

	args := runtime.Args{
		Blob:          deploy.Blob,
		Cache:         compCache,
		RequestPlugin: reqPlugin,
		Env:           app.Environment,
		RequestID:     requestID,
		AppID:         app.ID,
		Store:         s.store,
	}
	if err := runtime.Run(r.Context(), args); err != nil {
		writeJson(w, http.StatusInternalServerError, handleRequestResponse{
			Error: err.Error(),
		})
		return
	}
	if _, err := reqPlugin.WriteResponse(w); err != nil {
		writeJson(w, http.StatusInternalServerError, handleRequestResponse{
			Error: err.Error(),
		})
		return
	}
}
func writeJson(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	err := json.NewEncoder(w).Encode(v)
	if err != nil {
		panic(err)
	}
}
