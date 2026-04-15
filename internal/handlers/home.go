package handlers

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/zeroverify/mock-verifier/internal/templates"
)

var walletBaseURL = func() string {
	if u := os.Getenv("WALLET_URL"); u != "" {
		return strings.TrimRight(u, "/")
	}
	return "https://wallet.zeroverify.net"
}()

func Home(w http.ResponseWriter, r *http.Request) {
	challenge := randomDecimalChallenge()
	baseURL := publicBaseURL(r)

	verifyURL := fmt.Sprintf(
		"%s/prove?proof_type=student_status&verifier_id=Mock+Merchant&challenge=%s&callback=%s/verify",
		walletBaseURL, challenge, baseURL,
	)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.Home.Execute(w, struct{ VerifyURL string }{verifyURL})
}

func publicBaseURL(r *http.Request) string {
	if base := os.Getenv("PUBLIC_BASE_URL"); base != "" {
		return strings.TrimRight(base, "/")
	}
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, r.Host)
}
