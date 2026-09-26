package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/behaviorengineering/wonderfeed/internal/apperr"
	"github.com/behaviorengineering/wonderfeed/internal/controlplane"
)

// Handler serves parent control-plane routes.
type Handler struct {
	Service       *controlplane.Service
	Auth          ParentAuth
	RequestBudget time.Duration
}

// HandlerConfig constructs a Handler.
type HandlerConfig struct {
	Service       *controlplane.Service
	ParentAuthKey string
	RequestBudget time.Duration
}

// CreateHandler validates dependencies.
func (cfg HandlerConfig) CreateHandler() *Handler {
	if cfg.Service == nil {
		panic("httpapi.CreateHandler: service is nil")
	}
	budget := cfg.RequestBudget
	if budget <= 0 {
		budget = 30 * time.Second
	}
	return &Handler{
		Service:       cfg.Service,
		Auth:          ParentAuth{ParentAuthKey: cfg.ParentAuthKey},
		RequestBudget: budget,
	}
}

// NewHandler is a package-level constructor.
func NewHandler(cfg HandlerConfig) *Handler {
	return cfg.CreateHandler()
}

// Mount registers routes on mux.
func (h *Handler) Mount(mux *http.ServeMux) {
	if mux == nil {
		panic("httpapi.Handler.Mount: mux is nil")
	}
	mux.HandleFunc("GET /health", h.handleHealth)

	api := http.NewServeMux()
	api.HandleFunc("GET /api/v1/parent/children", h.handleListChildren)
	api.HandleFunc("POST /api/v1/parent/children", h.handleCreateChild)
	api.HandleFunc("GET /api/v1/parent/children/{id}", h.handleGetChild)
	api.HandleFunc("PUT /api/v1/parent/children/{id}/policy", h.handleUpdatePolicy)
	api.HandleFunc("GET /api/v1/parent/children/{id}/allowlist", h.handleListAllowlist)
	api.HandleFunc("POST /api/v1/parent/children/{id}/allowlist", h.handleAddAllowlistEntry)
	api.HandleFunc("GET /api/v1/parent/children/{id}/allowlist/{provider}/{external_id}", h.handleGetAllowlistEntry)
	api.HandleFunc("PATCH /api/v1/parent/children/{id}/allowlist/{provider}/{external_id}", h.handlePatchAllowlistEntry)
	api.HandleFunc("DELETE /api/v1/parent/children/{id}/allowlist/{provider}/{external_id}", h.handleDeleteAllowlistEntry)
	api.HandleFunc("POST /api/v1/parent/children/{id}/sync", h.handleForceSync)
	mux.Handle("/", h.Auth.Middleware(api))
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type createChildBody struct {
	Name        string                    `json:"name"`
	AvatarColor string                    `json:"avatar_color"`
	Policy      *controlplane.ChildPolicy `json:"policy,omitempty"`
}

type updatePolicyBody struct {
	ExpectedVersion int64                    `json:"expected_version"`
	Policy          controlplane.ChildPolicy `json:"policy"`
}

type addAllowlistBody struct {
	ExpectedVersion int64 `json:"expected_version"`
	Provider        string `json:"provider"`
	ExternalID      string `json:"external_id"`
	ChannelID       string `json:"channel_id"`
	Title           string `json:"title"`
	URL             string `json:"url"`
}

type patchAllowlistBody struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

func (h *Handler) withBudget(r *http.Request) (*http.Request, func()) {
	ctx, cancel := contextWithTimeout(r, h.RequestBudget)
	return r.WithContext(ctx), cancel
}

func (h *Handler) handleListChildren(w http.ResponseWriter, r *http.Request) {
	r, cancel := h.withBudget(r)
	defer cancel()
	out, err := h.Service.ListChildProfiles(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"children": out})
}

func (h *Handler) handleCreateChild(w http.ResponseWriter, r *http.Request) {
	r, cancel := h.withBudget(r)
	defer cancel()
	var body createChildBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, apperr.New(apperr.CodeInvalid, "httpapi.createChild", "invalid JSON body"))
		return
	}
	out, err := h.Service.CreateChildProfile(r.Context(), controlplane.CreateChildRequest{
		Name:        body.Name,
		AvatarColor: body.AvatarColor,
		Policy:      body.Policy,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	status := http.StatusCreated
	if out.SyncStatus == controlplane.SyncFailed {
		status = http.StatusAccepted
	}
	writeJSON(w, status, out)
}

func (h *Handler) handleGetChild(w http.ResponseWriter, r *http.Request) {
	r, cancel := h.withBudget(r)
	defer cancel()
	id := r.PathValue("id")
	out, err := h.Service.GetChildProfile(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) handleUpdatePolicy(w http.ResponseWriter, r *http.Request) {
	r, cancel := h.withBudget(r)
	defer cancel()
	id := r.PathValue("id")
	var body updatePolicyBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, apperr.New(apperr.CodeInvalid, "httpapi.updatePolicy", "invalid JSON body"))
		return
	}
	out, err := h.Service.UpdateChildPolicy(r.Context(), id, controlplane.UpdatePolicyRequest{
		ExpectedVersion: body.ExpectedVersion,
		Policy:          body.Policy,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	status := http.StatusOK
	if out.SyncStatus == controlplane.SyncFailed || out.SyncStatus == controlplane.SyncPending {
		status = http.StatusAccepted
	}
	writeJSON(w, status, out)
}

func (h *Handler) handleListAllowlist(w http.ResponseWriter, r *http.Request) {
	r, cancel := h.withBudget(r)
	defer cancel()
	allowlist, version, err := h.Service.ListChildAllowlist(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"allowlist": allowlist,
		"version":   version,
	})
}

func (h *Handler) handleAddAllowlistEntry(w http.ResponseWriter, r *http.Request) {
	r, cancel := h.withBudget(r)
	defer cancel()
	var body addAllowlistBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, apperr.New(apperr.CodeInvalid, "httpapi.addAllowlist", "invalid JSON body"))
		return
	}
	externalID := strings.TrimSpace(body.ExternalID)
	if externalID == "" {
		externalID = strings.TrimSpace(body.ChannelID)
	}
	out, err := h.Service.AddChildAllowlistEntry(r.Context(), r.PathValue("id"), controlplane.AddAllowlistEntryRequest{
		ExpectedVersion: body.ExpectedVersion,
		Entry: controlplane.AllowlistChannel{
			Provider:   body.Provider,
			ExternalID: externalID,
			Title:      body.Title,
			URL:        body.URL,
		},
	})
	if err != nil {
		writeError(w, err)
		return
	}
	status := http.StatusCreated
	if out.SyncStatus != controlplane.SyncSynced {
		status = http.StatusAccepted
	}
	writeJSON(w, status, out)
}

func (h *Handler) handleGetAllowlistEntry(w http.ResponseWriter, r *http.Request) {
	r, cancel := h.withBudget(r)
	defer cancel()
	entry, err := h.Service.GetChildAllowlistEntry(r.Context(), r.PathValue("id"), r.PathValue("provider"), r.PathValue("external_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, entry)
}

func (h *Handler) handlePatchAllowlistEntry(w http.ResponseWriter, r *http.Request) {
	r, cancel := h.withBudget(r)
	defer cancel()
	var body patchAllowlistBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, apperr.New(apperr.CodeInvalid, "httpapi.patchAllowlist", "invalid JSON body"))
		return
	}
	entry, err := h.Service.UpdateChildAllowlistEntry(r.Context(), r.PathValue("id"), r.PathValue("provider"), r.PathValue("external_id"), controlplane.UpdateAllowlistEntryRequest{
		Title: body.Title,
		URL:   body.URL,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, entry)
}

func (h *Handler) handleDeleteAllowlistEntry(w http.ResponseWriter, r *http.Request) {
	r, cancel := h.withBudget(r)
	defer cancel()
	expectedVersion, err := parseExpectedVersionQuery(r)
	if err != nil {
		writeError(w, err)
		return
	}
	out, err := h.Service.DeleteChildAllowlistEntry(r.Context(), r.PathValue("id"), r.PathValue("provider"), r.PathValue("external_id"), controlplane.DeleteAllowlistEntryRequest{
		ExpectedVersion: expectedVersion,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	status := http.StatusOK
	if out.SyncStatus != controlplane.SyncSynced {
		status = http.StatusAccepted
	}
	writeJSON(w, status, out)
}

func parseExpectedVersionQuery(r *http.Request) (int64, error) {
	const op = "httpapi.parseExpectedVersionQuery"
	raw := strings.TrimSpace(r.URL.Query().Get("expected_version"))
	if raw == "" {
		return 0, apperr.New(apperr.CodeInvalid, op, "expected_version query parameter is required")
	}
	version, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || version < 1 {
		return 0, apperr.New(apperr.CodeInvalid, op, "expected_version must be a positive integer")
	}
	return version, nil
}

func (h *Handler) handleForceSync(w http.ResponseWriter, r *http.Request) {
	r, cancel := h.withBudget(r)
	defer cancel()
	id := r.PathValue("id")
	if strings.TrimSpace(id) == "" {
		writeError(w, apperr.New(apperr.CodeInvalid, "httpapi.forceSync", "id is required"))
		return
	}
	out, err := h.Service.ForceSync(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	status := http.StatusOK
	if out.SyncStatus != controlplane.SyncSynced {
		status = http.StatusAccepted
	}
	writeJSON(w, status, out)
}
