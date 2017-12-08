package loglevel

import (
	"code.cloudfoundry.org/lager"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"service-discovery-controller/config"
)

type Server struct {
	config *config.Config
	sink   *lager.ReconfigurableSink
	logger lager.Logger
}

func NewServer(config *config.Config, sink *lager.ReconfigurableSink, logger lager.Logger) *Server {
	return &Server{
		config: config,
		sink:   sink,
		logger: logger,
	}
}

func (s *Server) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/log-level", s.handleRequest)

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", s.config.LogLevelAddress, s.config.LogLevelPort),
		Handler: mux,
	}
	server.SetKeepAlivesEnabled(false)

	exited := make(chan error)
	go func() {
		err := server.ListenAndServe()
		if err != nil {
			s.logger.Error("Listen and serve exited with error:", err)
			exited <- err
		}
	}()

	close(ready)

	for {
		select {
		case err := <-exited:
			server.Close()
			return err
		case <-signals:
			server.Close()
			return nil
		}
	}
}

func (s *Server) handleRequest(resp http.ResponseWriter, req *http.Request) {
	bytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		s.logger.Info("Unable to read request body")
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	body := string(bytes)

	var returnStatus int

	switch body {
	case "info":
		s.sink.SetMinLevel(lager.INFO)
		s.logger.Info("Log level set to INFO")
		returnStatus = http.StatusNoContent
	case "debug":
		s.sink.SetMinLevel(lager.DEBUG)
		s.logger.Info("Log level set to DEBUG")
		returnStatus = http.StatusNoContent
	default:
		s.logger.Info(fmt.Sprintf("Invalid log level requested: `%s`. Skipping.", body))
		returnStatus = http.StatusBadRequest
	}

	resp.WriteHeader(returnStatus)
}
