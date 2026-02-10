package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mklimuk/vault-pilot/pkg/automation"
	"github.com/mklimuk/vault-pilot/pkg/db"
)

type createAutomationRequest struct {
	Name         string          `json:"name"`
	ActionType   string          `json:"action_type"`
	ScheduleKind string          `json:"schedule_kind"`
	ScheduleExpr string          `json:"schedule_expr"`
	Timezone     string          `json:"timezone"`
	Payload      json.RawMessage `json:"payload"`
	Enabled      *bool           `json:"enabled"`
}

type updateAutomationRequest struct {
	Name         *string          `json:"name"`
	ActionType   *string          `json:"action_type"`
	ScheduleKind *string          `json:"schedule_kind"`
	ScheduleExpr *string          `json:"schedule_expr"`
	Timezone     *string          `json:"timezone"`
	Payload      *json.RawMessage `json:"payload"`
	Enabled      *bool            `json:"enabled"`
}

func (h *Handler) HandleCreateAutomation(w http.ResponseWriter, r *http.Request) {
	var req createAutomationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.ActionType = strings.TrimSpace(req.ActionType)
	req.ScheduleKind = strings.TrimSpace(strings.ToLower(req.ScheduleKind))
	req.ScheduleExpr = strings.TrimSpace(req.ScheduleExpr)
	if req.Name == "" || req.ActionType == "" || req.ScheduleKind == "" || req.ScheduleExpr == "" {
		http.Error(w, "name, action_type, schedule_kind and schedule_expr are required", http.StatusBadRequest)
		return
	}

	tz := strings.TrimSpace(req.Timezone)
	if tz == "" {
		tz = "UTC"
	}
	nextRun, err := automation.NextRun(req.ScheduleKind, req.ScheduleExpr, tz, time.Now().UTC())
	if err != nil {
		http.Error(w, "invalid schedule: "+err.Error(), http.StatusBadRequest)
		return
	}

	payload := []byte("{}")
	if len(req.Payload) > 0 {
		payload = req.Payload
		if !json.Valid(payload) {
			http.Error(w, "payload must be valid JSON", http.StatusBadRequest)
			return
		}
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	def := &db.AutomationDefinition{
		Name:         req.Name,
		ActionType:   req.ActionType,
		ScheduleKind: req.ScheduleKind,
		ScheduleExpr: req.ScheduleExpr,
		Timezone:     tz,
		PayloadJSON:  string(payload),
		Enabled:      enabled,
		NextRunAt:    nextRun,
	}
	id, err := h.Repo.CreateAutomation(def)
	if err != nil {
		http.Error(w, "failed to create automation: "+err.Error(), http.StatusInternalServerError)
		return
	}
	created, err := h.Repo.GetAutomationByID(id)
	if err != nil {
		http.Error(w, "failed to fetch created automation: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) HandleListAutomations(w http.ResponseWriter, r *http.Request) {
	defs, err := h.Repo.ListAutomations()
	if err != nil {
		http.Error(w, "failed to list automations: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"automations": defs})
}

func (h *Handler) HandleUpdateAutomation(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDPath(w, r)
	if !ok {
		return
	}
	current, err := h.Repo.GetAutomationByID(id)
	if err != nil {
		http.Error(w, "failed to load automation: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if current == nil {
		http.Error(w, "automation not found", http.StatusNotFound)
		return
	}

	var req updateAutomationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name != nil {
		current.Name = strings.TrimSpace(*req.Name)
	}
	if req.ActionType != nil {
		current.ActionType = strings.TrimSpace(*req.ActionType)
	}
	if req.ScheduleKind != nil {
		current.ScheduleKind = strings.TrimSpace(strings.ToLower(*req.ScheduleKind))
	}
	if req.ScheduleExpr != nil {
		current.ScheduleExpr = strings.TrimSpace(*req.ScheduleExpr)
	}
	if req.Timezone != nil {
		tz := strings.TrimSpace(*req.Timezone)
		if tz == "" {
			tz = "UTC"
		}
		current.Timezone = tz
	}
	if req.Payload != nil {
		if !json.Valid(*req.Payload) {
			http.Error(w, "payload must be valid JSON", http.StatusBadRequest)
			return
		}
		current.PayloadJSON = string(*req.Payload)
	}
	if req.Enabled != nil {
		current.Enabled = *req.Enabled
	}

	if current.Name == "" || current.ActionType == "" || current.ScheduleKind == "" || current.ScheduleExpr == "" {
		http.Error(w, "name, action_type, schedule_kind and schedule_expr are required", http.StatusBadRequest)
		return
	}

	nextRun, err := automation.NextRun(current.ScheduleKind, current.ScheduleExpr, current.Timezone, time.Now().UTC())
	if err != nil {
		http.Error(w, "invalid schedule: "+err.Error(), http.StatusBadRequest)
		return
	}
	current.NextRunAt = nextRun

	if err := h.Repo.UpdateAutomation(current); err != nil {
		http.Error(w, "failed to update automation: "+err.Error(), http.StatusInternalServerError)
		return
	}
	updated, err := h.Repo.GetAutomationByID(current.ID)
	if err != nil {
		http.Error(w, "failed to fetch automation: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) HandleRunAutomationNow(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDPath(w, r)
	if !ok {
		return
	}
	current, err := h.Repo.GetAutomationByID(id)
	if err != nil {
		http.Error(w, "failed to load automation: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if current == nil {
		http.Error(w, "automation not found", http.StatusNotFound)
		return
	}
	if err := h.Repo.TriggerAutomationNow(id, time.Now().UTC()); err != nil {
		http.Error(w, "failed to trigger automation: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "scheduled"})
}

func parseIDPath(w http.ResponseWriter, r *http.Request) (int64, bool) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
