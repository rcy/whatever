package web

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/rcy/whatever/commands"
)

type webhookResponse struct {
	ID string `json:"id"`
}

// postWebhookNote creates a webhook event from an external caller, authenticated
// with a bearer token matching WEBHOOK_TOKEN.
func (s *webservice) postWebhookNote(w http.ResponseWriter, r *http.Request) {
	if s.webhookToken == "" {
		http.NotFound(w, r)
		return
	}
	if s.ownerID == "" {
		http.Error(w, "owner not configured", http.StatusInternalServerError)
		return
	}

	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if subtle.ConstantTimeCompare([]byte(token), []byte(s.webhookToken)) != 1 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	log.Printf("webhook: content-type=%s body=%s", r.Header.Get("Content-Type"), string(body))
	r.Body = io.NopCloser(bytes.NewReader(body))

	fields := make(map[string]string)

	switch r.Header.Get("Content-Type") {
	case "application/json":
		var jsonFields map[string]string
		if err := json.NewDecoder(r.Body).Decode(&jsonFields); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		fields = jsonFields
	default:
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			// fall back to regular form parsing
			if err := r.ParseForm(); err != nil {
				http.Error(w, "failed to parse form", http.StatusBadRequest)
				return
			}
		}
		for key, values := range r.Form {
			if len(values) > 0 {
				fields[key] = values[0]
			}
		}
	}

	webhookID := uuid.New()
	err = s.app.Commander.Send(commands.ReceiveWebhook{
		WebhookID: webhookID,
		Owner:     s.ownerID,
		Fields:    fields,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(webhookResponse{ID: webhookID.String()})
}
