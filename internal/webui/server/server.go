package server

import (
	"fmt"

	"github.com/anthdm/raptor/internal/webui/handler"
	"github.com/labstack/echo/v4"
)

type Server struct {
	WebUI  *echo.Echo
	config *WebUiConfig
}

type WebUiConfig struct {
	HostAddr string
	WebUiURL string
	ApiAddr  string
	WasmAddr string
}

func New(config *WebUiConfig) *Server {
	return &Server{
		WebUI:  echo.New(),
		config: config,
	}
}

func (s *Server) AddRoutes() {
	s.WebUI.GET("/", handler.HomeHandler{}.HandleHomeShow)
}

func (s *Server) Start() error {
	s.WebUI.HideBanner = true
	s.WebUI.HidePort = true

	s.AddRoutes()

	fmt.Printf("webui server running\t%s\n", s.config.WebUiURL)
	return s.WebUI.Start(s.config.HostAddr)
}

func (s *Server) Close() error {
	return s.WebUI.Close()
}
