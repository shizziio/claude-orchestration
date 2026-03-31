package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/nextlevelbuilder/goclaw/internal/store"
)

// DevflowGitCredentialsHandler handles git credential CRUD endpoints.
type DevflowGitCredentialsHandler struct {
	creds store.GitCredentialStore
}

func NewDevflowGitCredentialsHandler(creds store.GitCredentialStore) *DevflowGitCredentialsHandler {
	return &DevflowGitCredentialsHandler{creds: creds}
}

func (h *DevflowGitCredentialsHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/devflow/git-credentials", requireAuth("", h.handleList))
	mux.HandleFunc("POST /v1/devflow/git-credentials", requireAuth("", h.handleCreate))
	mux.HandleFunc("GET /v1/devflow/git-credentials/{id}", requireAuth("", h.handleGet))
	mux.HandleFunc("PUT /v1/devflow/git-credentials/{id}", requireAuth("", h.handleUpdate))
	mux.HandleFunc("DELETE /v1/devflow/git-credentials/{id}", requireAuth("", h.handleDelete))
}

func (h *DevflowGitCredentialsHandler) handleList(w http.ResponseWriter, r *http.Request) {
	userID := store.UserIDFromContext(r.Context())
	creds, err := h.creds.List(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if creds == nil {
		creds = []*store.GitCredential{}
	}
	writeJSON(w, http.StatusOK, creds)
}

func (h *DevflowGitCredentialsHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Label      string  `json:"label"`
		Provider   string  `json:"provider"`
		Host       *string `json:"host"`
		AuthType   string  `json:"auth_type"`
		PublicKey  *string `json:"public_key"`
		PrivateKey *string `json:"private_key"` // PEM text
		Token      *string `json:"token"`       // PAT
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if body.Label == "" || body.AuthType == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "label and auth_type are required"})
		return
	}
	if body.AuthType != "ssh_key" && body.AuthType != "token" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "auth_type must be ssh_key or token"})
		return
	}
	provider := body.Provider
	if provider == "" {
		provider = "custom"
	}

	var privateKeyBytes, tokenBytes []byte
	if body.PrivateKey != nil {
		// Normalize line endings — clipboard paste often introduces \r\n
		// which causes OpenSSH to reject the key file as "invalid format".
		key := strings.ReplaceAll(*body.PrivateKey, "\r\n", "\n")
		key = strings.ReplaceAll(key, "\r", "\n")
		// Ensure the key ends with a newline as OpenSSH expects
		key = strings.TrimRight(key, "\n") + "\n"
		privateKeyBytes = []byte(key)
	}
	if body.Token != nil {
		tokenBytes = []byte(strings.TrimSpace(*body.Token))
	}

	userID := store.UserIDFromContext(r.Context())
	cred, err := h.creds.Create(r.Context(), store.CreateGitCredentialInput{
		UserID:     userID,
		Label:      body.Label,
		Provider:   provider,
		Host:       body.Host,
		AuthType:   body.AuthType,
		PublicKey:  body.PublicKey,
		PrivateKey: privateKeyBytes,
		Token:      tokenBytes,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, cred)
}

func (h *DevflowGitCredentialsHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var body struct {
		Label      *string `json:"label"`
		Host       *string `json:"host"`
		PublicKey  *string `json:"public_key"`
		PrivateKey *string `json:"private_key"`
		Token      *string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	in := store.UpdateGitCredentialInput{
		Label:     body.Label,
		Host:      body.Host,
		PublicKey: body.PublicKey,
	}
	if body.PrivateKey != nil && *body.PrivateKey != "" {
		key := strings.ReplaceAll(*body.PrivateKey, "\r\n", "\n")
		key = strings.ReplaceAll(key, "\r", "\n")
		key = strings.TrimRight(key, "\n") + "\n"
		in.PrivateKey = []byte(key)
	}
	if body.Token != nil && *body.Token != "" {
		in.Token = []byte(strings.TrimSpace(*body.Token))
	}

	cred, err := h.creds.Update(r.Context(), id, in)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, cred)
}

func (h *DevflowGitCredentialsHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	cred, err := h.creds.Get(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "credential not found"})
		return
	}
	writeJSON(w, http.StatusOK, cred)
}

func (h *DevflowGitCredentialsHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := h.creds.Delete(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
