package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/anthdm/run/pkg/config"
	"github.com/anthdm/run/pkg/storage"
	"github.com/anthdm/run/pkg/types"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Server serves the public run API.
type Server struct {
	router      *chi.Mux
	store       storage.Store
	metricStore storage.MetricStore
	cache       storage.ModCacher
}

// NewServer returns a new server given a Store interface.
func NewServer(store storage.Store, metricStore storage.MetricStore, cache storage.ModCacher) *Server {
	return &Server{
		store:       store,
		cache:       cache,
		metricStore: metricStore,
	}
}

// Listen starts listening on the given address.
func (s *Server) Listen(addr string) error {
	s.initRouter()
	return http.ListenAndServe(addr, s.router)
}

func (s *Server) initRouter() {
	s.router = chi.NewRouter()
	if config.Get().Authorization {
		s.router.Use(s.withAPIToken)
	}
	s.router.Get("/status", handleStatus)
	s.router.Get("/application/{id}", makeAPIHandler(s.handleGetApplication))
	s.router.Get("/application/{id}/metrics", makeAPIHandler(s.handleGetEndpointMetrics))
	s.router.Post("/application", makeAPIHandler(s.handleCreateApplication))
	s.router.Post("/application/{id}/deploy", makeAPIHandler(s.handleCreateDeploy))
	s.router.Post("/application/{id}/rollback", makeAPIHandler(s.handleCreateRollback))
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	status := map[string]string{
		"status": "ok",
	}
	json.NewEncoder(w).Encode(status)
}

func (s *Server) handleCreateApplication(w http.ResponseWriter, r *http.Request) error {
	var (
		params CreateParams
		body   []byte
		err    error
	)

	body, err = io.ReadAll(r.Body)
	if err != nil {
		return writeJSON(w, http.StatusBadRequest, ErrorResponse(err))
	}

	params, err = DecodeParams(body)

	var app types.App

	switch params.getType() {
	case types.AppTypeEndpoint:
		params := params.(CreateEndpointParams)
		endpoint := types.NewEndpoint(params.Name, params.Runtime, params.Environment)
		endpoint.URL = config.GetWasmUrl() + "/" + endpoint.ID.String()
		app = endpoint

	case types.AppTypeCron:
		params := params.(CreateCronParams)
		cron := types.NewCron(params.Name, params.Runtime, params.Interval, params.Environment)
		app = cron

	case types.AppTypeProcess:
		params := params.(CreateProcessparams)
		process := types.NewProcess(params.Name, params.Runtime, params.Environment)
		app = process

	default:
		return writeJSON(w, http.StatusBadRequest, ErrorResponse(fmt.Errorf("invalid application type (%T)", params.getType())))
	}

	if err := s.store.CreateApp(app); err != nil {
		return writeJSON(w, http.StatusBadRequest, ErrorResponse(err))
	}
	return writeJSON(w, http.StatusOK, app)
}

// CreateDeployParams holds all the necessary fields to deploy a new function.
type CreateDeployParams struct{}

func (s *Server) handleCreateDeploy(w http.ResponseWriter, r *http.Request) error {
	appID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return writeJSON(w, http.StatusBadRequest, ErrorResponse(err))
	}
	app, err := s.store.GetApp(appID)
	if err != nil {
		return writeJSON(w, http.StatusNotFound, ErrorResponse(err))
	}

	b, err := io.ReadAll(r.Body)
	if err != nil {
		return writeJSON(w, http.StatusNotFound, ErrorResponse(err))
	}

	switch a := app.(type) {
	case *types.Endpoint:
		return s.handleCreateEndpointDeploy(w, r, a, b)
	case *types.Cron:
		return s.handleCreateCronDeploy(w, r, a, b)
	case *types.Process:
		return s.handleCreateProcessDeploy(w, r, a, b)
	default:
		return writeJSON(w, http.StatusBadRequest, ErrorResponse(fmt.Errorf("unknown app type (%T)", app)))
	}
}

func (s *Server) handleCreateEndpointDeploy(w http.ResponseWriter, r *http.Request, endpoint *types.Endpoint, b []byte) error {
	deploy := types.NewEndpointDeploy(endpoint, b)
	if err := s.store.CreateDeploy(deploy); err != nil {
		return writeJSON(w, http.StatusUnprocessableEntity, ErrorResponse(err))
	}
	// Each new deploy will be the endpoint's active deploy
	err := s.store.UpdateApp(endpoint.ID, types.EndpointUpdateParams{
		ActiveDeployID: deploy.ID,
		Deploys:        []*types.EndpointDeploy{deploy},
	})

	if err != nil {
		return writeJSON(w, http.StatusUnprocessableEntity, ErrorResponse(err))
	}
	return writeJSON(w, http.StatusOK, deploy)
}

func (s *Server) handleCreateCronDeploy(w http.ResponseWriter, r *http.Request, cron *types.Cron, b []byte) error {
	deploy := types.NewCronDeploy(cron, b)
	if err := s.store.CreateDeploy(deploy); err != nil {
		return writeJSON(w, http.StatusUnprocessableEntity, ErrorResponse(err))
	}

	// Each new deploy will be the cron's active deploy
	err := s.store.UpdateApp(cron.ID, types.CronUpdateParams{
		ActiveDeployID: deploy.ID,
		Deploys:        []*types.CronDeploy{deploy},
	})

	if err != nil {
		return writeJSON(w, http.StatusUnprocessableEntity, ErrorResponse(err))
	}

	// todo start cron
	return writeJSON(w, http.StatusOK, deploy)
}

func (s *Server) handleCreateProcessDeploy(w http.ResponseWriter, r *http.Request, process *types.Process, b []byte) error {
	deploy := types.NewProcessDeploy(process, b)
	if err := s.store.CreateDeploy(deploy); err != nil {
		return writeJSON(w, http.StatusUnprocessableEntity, ErrorResponse(err))
	}

	// Each new deploy will be the process's active deploy
	err := s.store.UpdateApp(process.ID, types.ProcessUpdateParams{
		ActiveDeployID: deploy.ID,
		Deploys:        []*types.ProcessDeploy{deploy},
	})

	if err != nil {
		return writeJSON(w, http.StatusUnprocessableEntity, ErrorResponse(err))
	}

	// todo start process
	return writeJSON(w, http.StatusOK, deploy)
}

func (s *Server) handleGetApplication(w http.ResponseWriter, r *http.Request) error {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return writeJSON(w, http.StatusBadRequest, ErrorResponse(err))
	}
	app, err := s.store.GetApp(id)
	if err != nil {
		return writeJSON(w, http.StatusNotFound, ErrorResponse(err))
	}
	return writeJSON(w, http.StatusOK, app)
}

// CreateRollbackParams holds all the necessary fields to rollback your application
// to a specific deploy id (version).
type CreateRollbackParams struct {
	DeployID uuid.UUID `json:"deploy_id"`
}

func (s *Server) handleCreateRollback(w http.ResponseWriter, r *http.Request) error {
	appID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return writeJSON(w, http.StatusBadRequest, ErrorResponse(err))
	}
	app, err := s.store.GetApp(appID)
	if err != nil {
		return writeJSON(w, http.StatusBadRequest, ErrorResponse(err))
	}

	currentDeployID := app.GetActiveDeployID()

	var params CreateRollbackParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		return writeJSON(w, http.StatusBadRequest, ErrorResponse(err))
	}

	deploy, err := s.store.GetDeploy(params.DeployID)
	if err != nil {
		return writeJSON(w, http.StatusNotFound, ErrorResponse(err))
	}

	var updateParams types.AppUpdateParams
	switch d := deploy.(type) {
	case *types.EndpointDeploy:
		updateParams = &types.EndpointUpdateParams{
			ActiveDeployID: d.ID,
		}
	case *types.CronDeploy:
		updateParams = &types.CronUpdateParams{
			ActiveDeployID: d.ID,
		}
	case *types.ProcessDeploy:
		updateParams = &types.ProcessUpdateParams{
			ActiveDeployID: d.ID,
		}
	default:
		return writeJSON(w, http.StatusBadRequest, ErrorResponse(fmt.Errorf("unknown app type (%T)", app)))
	}

	if err := s.store.UpdateApp(appID, updateParams); err != nil {
		return writeJSON(w, http.StatusBadRequest, ErrorResponse(err))
	}

	s.cache.Delete(currentDeployID)

	return writeJSON(w, http.StatusOK, deploy)
}

func (s *Server) handleGetEndpointMetrics(w http.ResponseWriter, r *http.Request) error {
	endpointID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return writeJSON(w, http.StatusBadRequest, ErrorResponse(err))
	}
	metrics, err := s.metricStore.GetMetrics(endpointID)
	if err != nil {
		return writeJSON(w, http.StatusNotFound, ErrorResponse(err))
	}
	return writeJSON(w, http.StatusOK, metrics)
}

var errUnauthorized = errors.New("unauthorized")

func (s *Server) withAPIToken(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if len(authHeader) < 10 {
			writeJSON(w, http.StatusUnauthorized, ErrorResponse(errUnauthorized))
			return
		}
		apiToken := strings.TrimPrefix(authHeader, "Bearer ")
		if apiToken != config.Get().APIToken {
			writeJSON(w, http.StatusUnauthorized, ErrorResponse(errUnauthorized))
			return
		}
		h.ServeHTTP(w, r)
	})
}
