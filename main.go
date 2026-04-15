package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/zeroverify/mock-verifier/internal/server"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	slog.Info("mock verifier starting", "port", port)
	if err := http.ListenAndServe(":"+port, server.New()); err != nil {
		slog.Error("server failed", "err", err)
		os.Exit(1)
	}
}
