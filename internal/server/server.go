package server

import (
	"net/http"

	verifier "github.com/zeroverify/verifier-go"
	"github.com/zeroverify/mock-verifier/internal/handlers"
	"github.com/zeroverify/mock-verifier/internal/hub"
	"github.com/zeroverify/mock-verifier/internal/store"
)

type Server struct {
	mux *http.ServeMux
}

func New() *Server {
	s := store.New()
	h := hub.New()
	fetcher := verifier.NewFetcher().Build()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", handlers.Home)
	mux.HandleFunc("POST /verify", handlers.Verify(s, h, fetcher))
	mux.HandleFunc("GET /events", handlers.Events(h))

	return &Server{mux: mux}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-ZeroVerify-Version")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	s.mux.ServeHTTP(w, r)
}
