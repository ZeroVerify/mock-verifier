package handlers

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/iden3/go-rapidsnark/types"
	"github.com/zeroverify/mock-verifier/internal/hub"
	"github.com/zeroverify/mock-verifier/internal/store"
	verifier "github.com/zeroverify/verifier-go"
)

type incomingProof struct {
	Proof         types.ProofData `json:"proof"`
	PublicSignals []string        `json:"publicSignals"`
}

type eventPayload struct {
	ID            string   `json:"id"`
	Valid         bool     `json:"valid"`
	Error         string   `json:"error,omitempty"`
	ProofJSON     string   `json:"proof_json"`
	PublicSignals []string `json:"public_signals"`
	ReceivedAt    string   `json:"received_at"`
}

// challengeSignalIndex is the position of challenge_nonce in the student_status public signals:
// [out_nonce, pseudonym_hash, revocation_index, Ax, Ay, challenge_nonce, now].
const (
	challengeSignalIndex = 5
	studentSignalCount   = 7
)

// Verify checks a proof the way a real relying party should: the nonce must be one this verifier issued (once), and the
// SDK then checks the issuer key, the clock, the revocation bit and the Groth16 proof. Checking only the Groth16 proof
// would accept a credential the prover signed themselves.
func Verify(s *store.Store, h *hub.Hub, client *verifier.Client, challenges *store.Challenges) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
		if err != nil {
			http.Error(w, "request too large or unreadable", http.StatusBadRequest)
			return
		}
		var payload incomingProof
		if err := json.Unmarshal(body, &payload); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		id := randomHex(8)
		proofJSON, err := json.Marshal(payload.Proof)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		result := &store.ProofResult{
			ID:            id,
			ProofJSON:     proofJSON,
			PublicSignals: payload.PublicSignals,
			ReceivedAt:    time.Now(),
		}

		challenge := ""
		if len(payload.PublicSignals) == studentSignalCount {
			challenge = payload.PublicSignals[challengeSignalIndex]
		}

		switch {
		case os.Getenv("SKIP_VERIFICATION") == "1":
			result.Valid = true
		case challenge == "" || !challenges.Consume(challenge):
			result.Error = "unknown_challenge"
		default:
			ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
			defer cancel()
			vr, err := client.VerifySubmission(ctx, body, "student_status", challenge)
			if err != nil {
				slog.Error("verifying submission", "err", err)
				result.Error = err.Error()
			} else {
				result.Valid = vr.Valid
				if !vr.Valid {
					result.Error = vr.Reason
				}
			}
		}

		s.Save(result)

		event := eventPayload{
			ID:            result.ID,
			Valid:         result.Valid,
			Error:         result.Error,
			ProofJSON:     string(result.ProofJSON),
			PublicSignals: result.PublicSignals,
			ReceivedAt:    result.ReceivedAt.Format(time.RFC3339),
		}
		eventJSON, _ := json.Marshal(event)
		h.Broadcast(eventJSON)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"id": id})
	}
}
