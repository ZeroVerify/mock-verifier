package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/iden3/go-rapidsnark/types"
	verifier "github.com/zeroverify/verifier-go"
	"github.com/zeroverify/mock-verifier/internal/hub"
	"github.com/zeroverify/mock-verifier/internal/store"
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

func Verify(s *store.Store, h *hub.Hub, fetcher *verifier.Fetcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload incomingProof
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
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

		if os.Getenv("SKIP_VERIFICATION") == "1" {
			result.Valid = true
		} else {
			ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
			defer cancel()

			vkJSON, err := fetcher.VerificationKey(ctx, "student_status")
			if err != nil {
				slog.Error("fetching vkey", "err", err)
				result.Error = fmt.Sprintf("failed to fetch verification key: %v", err)
			} else {
				vr, err := verifier.VerifyProof(proofJSON, vkJSON, payload.PublicSignals)
				if err != nil {
					result.Error = err.Error()
				} else {
					result.Valid = vr.Valid
					if !vr.Valid {
						result.Error = vr.Reason
					}
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
