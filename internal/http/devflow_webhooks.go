package http

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"github.com/nextlevelbuilder/goclaw/internal/store"
)

// DevflowWebhookHandler handles webhook CRUD and incoming git events.
type DevflowWebhookHandler struct {
	webhooks store.DevflowWebhookStore
	projects store.ProjectStore
	runs     store.DevflowRunStore
}

func NewDevflowWebhookHandler(webhooks store.DevflowWebhookStore, projects store.ProjectStore, runs store.DevflowRunStore) *DevflowWebhookHandler {
	return &DevflowWebhookHandler{webhooks: webhooks, projects: projects, runs: runs}
}

func (h *DevflowWebhookHandler) RegisterRoutes(mux *http.ServeMux) {
	// CRUD (authenticated)
	mux.HandleFunc("GET /v1/devflow/projects/{projectId}/webhooks", requireAuth("", h.handleList))
	mux.HandleFunc("POST /v1/devflow/projects/{projectId}/webhooks", requireAuth("", h.handleCreate))
	mux.HandleFunc("GET /v1/devflow/projects/{projectId}/webhooks/{id}", requireAuth("", h.handleGet))
	mux.HandleFunc("PUT /v1/devflow/projects/{projectId}/webhooks/{id}", requireAuth("", h.handleUpdate))
	mux.HandleFunc("DELETE /v1/devflow/projects/{projectId}/webhooks/{id}", requireAuth("", h.handleDelete))
	// Incoming webhook (no auth — validated by secret)
	mux.HandleFunc("POST /v1/devflow/webhooks/incoming/{projectId}", h.handleIncoming)
}

func (h *DevflowWebhookHandler) handleList(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(r.PathValue("projectId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid projectId"})
		return
	}
	hooks, err := h.webhooks.ListByProject(r.Context(), projectID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if hooks == nil {
		hooks = []*store.DevflowWebhook{}
	}
	writeJSON(w, http.StatusOK, hooks)
}

func (h *DevflowWebhookHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(r.PathValue("projectId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid projectId"})
		return
	}
	var body struct {
		EventType    string  `json:"event_type"`
		BranchFilter *string `json:"branch_filter"`
		TaskTemplate string  `json:"task_template"`
		Secret       *string `json:"secret"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if body.TaskTemplate == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "task_template is required"})
		return
	}
	hook, err := h.webhooks.Create(r.Context(), store.CreateWebhookInput{
		ProjectID:    projectID,
		EventType:    body.EventType,
		BranchFilter: body.BranchFilter,
		TaskTemplate: body.TaskTemplate,
		Secret:       body.Secret,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, hook)
}

func (h *DevflowWebhookHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	hook, err := h.webhooks.Get(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, hook)
}

func (h *DevflowWebhookHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var body struct {
		EventType    *string `json:"event_type"`
		BranchFilter *string `json:"branch_filter"`
		TaskTemplate *string `json:"task_template"`
		Enabled      *bool   `json:"enabled"`
		Secret       *string `json:"secret"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	hook, err := h.webhooks.Update(r.Context(), id, store.UpdateWebhookInput{
		EventType:    body.EventType,
		BranchFilter: body.BranchFilter,
		TaskTemplate: body.TaskTemplate,
		Enabled:      body.Enabled,
		Secret:       body.Secret,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, hook)
}

func (h *DevflowWebhookHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := h.webhooks.Delete(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleIncoming receives webhook payloads from GitHub/GitLab/Bitbucket.
// No auth middleware — validates via HMAC secret per webhook.
func (h *DevflowWebhookHandler) handleIncoming(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(r.PathValue("projectId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid projectId"})
		return
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1MB limit
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to read body"})
		return
	}

	// Detect event type from headers
	eventType := detectEventType(r)

	// Parse common payload fields
	payload := parseWebhookPayload(bodyBytes)

	// Inject tenant context for DB queries (webhooks are project-scoped)
	project, err := h.projects.Get(store.WithTenantID(r.Context(), uuid.Nil), projectID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project not found"})
		return
	}
	ctx := store.WithTenantID(r.Context(), project.TenantID)
	ctx = store.WithUserID(ctx, "webhook")

	// Find matching enabled webhooks
	hooks, err := h.webhooks.ListEnabled(ctx, projectID, eventType)
	if err != nil || len(hooks) == 0 {
		writeJSON(w, http.StatusOK, map[string]string{"status": "no matching webhooks"})
		return
	}

	var created int
	for _, hook := range hooks {
		// Validate secret if configured
		if hook.Secret != nil && *hook.Secret != "" {
			sig := r.Header.Get("X-Hub-Signature-256")
			if sig == "" {
				sig = r.Header.Get("X-Gitlab-Token")
			}
			if !validateSignature(bodyBytes, *hook.Secret, sig) {
				slog.Warn("devflow.webhook.invalid_signature", "hook_id", hook.ID, "project_id", projectID)
				continue
			}
		}

		// Check branch filter
		if hook.BranchFilter != nil && *hook.BranchFilter != "" && payload.Branch != "" {
			matched, _ := regexp.MatchString(*hook.BranchFilter, payload.Branch)
			if !matched {
				continue
			}
		}

		// Render task from template
		task := renderTaskTemplate(hook.TaskTemplate, payload)

		// Create run
		run, err := h.runs.Create(ctx, store.CreateRunInput{
			ProjectID:       projectID,
			TaskDescription: task,
			Branch:          &payload.Branch,
			CreatedBy:       "webhook",
		})
		if err != nil {
			slog.Error("devflow.webhook.create_run_failed", "hook_id", hook.ID, "error", err)
			continue
		}
		created++
		slog.Info("devflow.webhook.run_created",
			"hook_id", hook.ID,
			"run_id", run.ID,
			"project_id", projectID,
			"branch", payload.Branch,
			"event", eventType,
		)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":   "ok",
		"runs_created": created,
	})
}

// --- webhook payload helpers ---

type webhookPayload struct {
	Branch    string
	CommitMsg string
	Author    string
	RepoURL   string
}

func detectEventType(r *http.Request) string {
	// GitHub
	if ev := r.Header.Get("X-GitHub-Event"); ev != "" {
		switch ev {
		case "push":
			return "push"
		case "pull_request":
			return "pull_request"
		}
		return ev
	}
	// GitLab
	if ev := r.Header.Get("X-Gitlab-Event"); ev != "" {
		if strings.Contains(ev, "Push") {
			return "push"
		}
		if strings.Contains(ev, "Merge Request") {
			return "pull_request"
		}
		return strings.ToLower(ev)
	}
	// Bitbucket — no standard header, check body
	return "push" // default
}

func parseWebhookPayload(body []byte) webhookPayload {
	var raw map[string]json.RawMessage
	json.Unmarshal(body, &raw)

	p := webhookPayload{}

	// GitHub/GitLab push: "ref" = "refs/heads/main"
	if ref, ok := raw["ref"]; ok {
		var refStr string
		json.Unmarshal(ref, &refStr)
		p.Branch = strings.TrimPrefix(refStr, "refs/heads/")
	}

	// Bitbucket push
	if push, ok := raw["push"]; ok {
		var pushData struct {
			Changes []struct {
				New struct {
					Name string `json:"name"`
				} `json:"new"`
			} `json:"changes"`
		}
		json.Unmarshal(push, &pushData)
		if len(pushData.Changes) > 0 {
			p.Branch = pushData.Changes[0].New.Name
		}
	}

	// Last commit message
	if commits, ok := raw["commits"]; ok {
		var commitList []struct {
			Message string `json:"message"`
			Author  struct {
				Name string `json:"name"`
			} `json:"author"`
		}
		json.Unmarshal(commits, &commitList)
		if len(commitList) > 0 {
			p.CommitMsg = commitList[len(commitList)-1].Message
			p.Author = commitList[len(commitList)-1].Author.Name
		}
	}

	return p
}

func renderTaskTemplate(tmpl string, p webhookPayload) string {
	r := strings.NewReplacer(
		"{{.Branch}}", p.Branch,
		"{{.CommitMsg}}", p.CommitMsg,
		"{{.Author}}", p.Author,
	)
	return r.Replace(tmpl)
}

func validateSignature(body []byte, secret, signature string) bool {
	if signature == "" {
		return false
	}
	// GitHub: X-Hub-Signature-256 = "sha256=hex"
	if strings.HasPrefix(signature, "sha256=") {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
		return hmac.Equal([]byte(expected), []byte(signature))
	}
	// GitLab: X-Gitlab-Token = plain secret
	return signature == secret
}

