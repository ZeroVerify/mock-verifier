package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/zeroverify/mock-verifier/internal/hub"
	"github.com/zeroverify/mock-verifier/internal/store"
	verifier "github.com/zeroverify/verifier-go"
)

func post(t *testing.T, challenges *store.Challenges, s *store.Store, body string) *store.ProofResult {
	t.Helper()
	// The fetcher points at an unreachable address: these tests must not need the network, and a request that
	// reaches the SDK would surface as an error result.
	client := verifier.NewClient(verifier.NewFetcher().WithVKeyURL("http://127.0.0.1:1/%s").Build())
	h := Verify(s, hub.New(), client, challenges)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/verify", strings.NewReader(body)).WithContext(context.Background())
	h(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]string
	json.Unmarshal(rec.Body.Bytes(), &resp)
	r, ok := s.Get(resp["id"])
	if !ok {
		t.Fatal("result not stored")
	}
	return r
}

const proofJSON = `{"pi_a":["0","0","1"],"pi_b":[["0","0"],["0","0"],["1","0"]],"pi_c":["0","0","1"],"protocol":"groth16","curve":"bn128"}`

func submission(signals ...string) string {
	b, _ := json.Marshal(signals)
	return `{"proof":` + proofJSON + `,"publicSignals":` + string(b) + `}`
}

func TestVerifyRejectsANonceThisVerifierNeverIssued(t *testing.T) {
	r := post(t, store.NewChallenges(time.Minute), store.New(), submission("1", "2", "3", "4", "5", "999", "7"))
	if r.Valid || r.Error != "unknown_challenge" {
		t.Fatalf("expected unknown_challenge, got valid=%v error=%q", r.Valid, r.Error)
	}
}

func TestVerifyRejectsAWrongSignalCount(t *testing.T) {
	c := store.NewChallenges(time.Minute)
	c.Issue("999")
	r := post(t, c, store.New(), submission("1", "2", "3", "999", "5"))
	if r.Valid || r.Error != "unknown_challenge" {
		t.Fatalf("expected unknown_challenge for a 5-signal (old circuit) submission, got valid=%v error=%q", r.Valid, r.Error)
	}
}

func TestVerifyConsumesTheNonce(t *testing.T) {
	c := store.NewChallenges(time.Minute)
	c.Issue("999")
	sig := []string{"999", "2", "3", "4", "5", "999", "7"}
	first := post(t, c, store.New(), submission(sig...))
	if first.Error == "unknown_challenge" {
		t.Fatalf("an issued nonce must reach the SDK, got %q", first.Error)
	}
	second := post(t, c, store.New(), submission(sig...))
	if second.Valid || second.Error != "unknown_challenge" {
		t.Fatalf("a replayed nonce must be rejected, got valid=%v error=%q", second.Valid, second.Error)
	}
}

func TestVerifyRejectsInvalidJSON(t *testing.T) {
	client := verifier.NewClient(verifier.NewFetcher().Build())
	h := Verify(store.New(), hub.New(), client, store.NewChallenges(time.Minute))
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodPost, "/verify", strings.NewReader("not json")))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
