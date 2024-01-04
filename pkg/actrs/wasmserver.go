package actrs

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/cluster"
	"github.com/anthdm/run/pkg/storage"
	"github.com/anthdm/run/pkg/types"
	"github.com/anthdm/run/proto"
	"github.com/google/uuid"
)

const KindWasmServer = "wasm_server"

type requestWithResponse struct {
	request  *proto.HTTPRequest
	response chan *proto.HTTPResponse
}

func newRequestWithResponse(request *proto.HTTPRequest) requestWithResponse {
	return requestWithResponse{
		request:  request,
		response: make(chan *proto.HTTPResponse, 1),
	}
}

type startTaskRequest struct {
	ID uuid.UUID
}

// WasmServer is an HTTP server that will proxy and route the request to the corresponding function.
type WasmServer struct {
	server      *http.Server
	self        *actor.PID
	store       storage.Store
	metricStore storage.MetricStore
	cache       storage.ModCacher
	cluster     *cluster.Cluster
	responses   map[string]chan *proto.HTTPResponse
}

// NewWasmServer return a new wasm server given a storage and a mod cache.
func NewWasmServer(addr string, cluster *cluster.Cluster, store storage.Store, metricStore storage.MetricStore, cache storage.ModCacher) actor.Producer {
	return func() actor.Receiver {
		s := &WasmServer{
			store:       store,
			metricStore: metricStore,
			cache:       cache,
			cluster:     cluster,
			responses:   make(map[string]chan *proto.HTTPResponse),
		}
		server := &http.Server{
			Handler: s,
			Addr:    addr,
		}
		s.server = server
		return s
	}
}

func (s *WasmServer) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
		s.initialize(c)

	case actor.Stopped:

	case requestWithResponse:
		s.responses[msg.request.ID] = msg.response
		s.sendEndpointRequestToRuntime(msg.request)

	case *proto.StartStopResponse:
		if resp, ok := s.responses[msg.RequestID]; ok {
			r := &proto.HTTPResponse{}
			if msg.Err != "" {
				r.Response = []byte(msg.Err)
				r.StatusCode = http.StatusInternalServerError
			} else {
				r.Response = []byte("ok")
				r.StatusCode = http.StatusOK
			}
			resp <- r
			delete(s.responses, msg.RequestID)
		}

	case *proto.HTTPResponse:
		if resp, ok := s.responses[msg.RequestID]; ok {
			resp <- msg
			delete(s.responses, msg.RequestID)
		}

	}
}

func (s *WasmServer) initialize(c *actor.Context) {
	s.self = c.PID()
	go func() {
		log.Fatal(s.server.ListenAndServe())
	}()
}

func (s *WasmServer) sendEndpointRequestToRuntime(req *proto.HTTPRequest) {
	pid := s.cluster.Activate(KindEndpointRuntime, &cluster.ActivationConfig{})
	s.cluster.Engine().SendWithSender(pid, req, s.self)
}

func (s *WasmServer) sendTaskStartRequestToRuntime(req *proto.StartRequest) {
	pid := s.cluster.Activate(KindTaskRuntime, &cluster.ActivationConfig{})
	s.cluster.Engine().SendWithSender(pid, req, s.self)
}

func (s *WasmServer) sendTaskStopRequestToRuntime(req *proto.StopRequest) {
	// todo make sure it sends to the runtime with the task
	pid := s.cluster.Activate(KindTaskRuntime, &cluster.ActivationConfig{})
	s.cluster.Engine().SendWithSender(pid, req, s.self)
}

func (s *WasmServer) sendProcessStartRequestToRuntime(req *proto.StartRequest) {
	pid := s.cluster.Activate(KindProcessRuntime, &cluster.ActivationConfig{})
	s.cluster.Engine().SendWithSender(pid, req, s.self)
}

func (s *WasmServer) sendProcessStopRequestToRuntime(req *proto.StopRequest) {
	// todo make sure it sends to the runtime with the task
	pid := s.cluster.Activate(KindProcessRuntime, &cluster.ActivationConfig{})
	s.cluster.Engine().SendWithSender(pid, req, s.self)
}

func (s *WasmServer) sendServeHTTPRequestToRuntime(req *proto.HTTPRequest) {
	pid := s.cluster.Activate(KindEndpointRuntime, &cluster.ActivationConfig{})
	s.cluster.Engine().SendWithSender(pid, req, s.self)
}

// TODO handle stop and start task and processes
func (s *WasmServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	pathParts := strings.Split(path, "/")
	if len(pathParts) == 0 {
		writeResponse(w, http.StatusBadRequest, []byte("invalid application id given"))
		return
	}
	id := pathParts[0]
	endpointID, err := uuid.Parse(id)
	if err != nil {
		writeResponse(w, http.StatusBadRequest, []byte(err.Error()))
		return
	}
	a, err := s.store.GetApp(endpointID)
	if err != nil {
		writeResponse(w, http.StatusNotFound, []byte(err.Error()))
		return
	}

	if !a.HasActiveDeploy() {
		writeResponse(w, http.StatusNotFound, []byte("application does not have an active deploy yet"))
		return
	}

	endpoint, ok := a.(*types.Endpoint)
	if !ok {
		writeResponse(w, http.StatusInternalServerError, []byte("could not cast endpoint type"))
		return
	}

	req, err := makeEndpointProtoRequest(r)
	if err != nil {
		writeResponse(w, http.StatusInternalServerError, []byte(err.Error()))
		return
	}
	req.Runtime = endpoint.Runtime
	req.EndpointID = endpointID.String()
	req.ActiveDeployID = endpoint.ActiveDeployID.String()
	req.Env = endpoint.Environment

	reqres := newRequestWithResponse(req)

	s.cluster.Engine().Send(s.self, reqres)

	resp := <-reqres.response
	w.WriteHeader(int(resp.StatusCode))
	w.Write(resp.Response)
}

func makeEndpointProtoRequest(r *http.Request) (*proto.HTTPRequest, error) {
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	return &proto.HTTPRequest{
		ID:     uuid.NewString(),
		Body:   b,
		Method: r.Method,
		URL:    trimmedEndpointFromURL(r.URL),
	}, nil
}

func makeStartProtoRequest(task types.App) *proto.StartRequest {
	return &proto.StartRequest{
		ID:        task.GetID().String(),
		RequestID: uuid.NewString(),
	}
}

func makeStopProtoRequest(task *types.Task) *proto.StopRequest {
	return &proto.StopRequest{
		ID:        task.ID.String(),
		RequestID: uuid.NewString(),
	}
}

func writeResponse(w http.ResponseWriter, code int, b []byte) {
	w.WriteHeader(http.StatusNotFound)
	w.Write(b)
}

func trimmedEndpointFromURL(url *url.URL) string {
	path := strings.TrimPrefix(url.Path, "/")
	pathParts := strings.Split(path, "/")
	return "/" + strings.Join(pathParts[1:], "/")
}
