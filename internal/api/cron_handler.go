package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/openilink/openilink-hub/internal/auth"
	"github.com/openilink/openilink-hub/internal/cron"
	"github.com/openilink/openilink-hub/internal/store"
)

// GET /api/bots/{id}/cron-jobs
func (s *Server) handleListCronJobs(w http.ResponseWriter, r *http.Request) {
	botID := r.PathValue("id")
	userID := auth.UserIDFromContext(r.Context())

	bot, err := s.Store.GetBot(botID)
	if err != nil || bot.UserID != userID {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	jobs, err := s.Store.ListCronJobsByBot(botID)
	if err != nil {
		jsonError(w, "list failed", http.StatusInternalServerError)
		return
	}
	if jobs == nil {
		jobs = []store.CronJob{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jobs)
}

// POST /api/bots/{id}/cron-jobs
func (s *Server) handleCreateCronJob(w http.ResponseWriter, r *http.Request) {
	botID := r.PathValue("id")
	userID := auth.UserIDFromContext(r.Context())

	bot, err := s.Store.GetBot(botID)
	if err != nil || bot.UserID != userID {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	var req struct {
		Name      string `json:"name"`
		CronExpr  string `json:"cron_expr"`
		Message   string `json:"message"`
		Recipient string `json:"recipient"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if req.CronExpr == "" || req.Message == "" {
		jsonError(w, "cron_expr and message are required", http.StatusBadRequest)
		return
	}
	if err := cron.Validate(req.CronExpr); err != nil {
		jsonError(w, "invalid cron expression: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Calculate first next_run_at — fail if the expression cannot be scheduled.
	next, err := cron.NextAfter(req.CronExpr, time.Now())
	if err != nil {
		jsonError(w, "unable to schedule cron job: "+err.Error(), http.StatusBadRequest)
		return
	}
	v := next.Unix()
	nextRunAt := &v

	job := &store.CronJob{
		BotID:     botID,
		UserID:    userID,
		Name:      req.Name,
		CronExpr:  req.CronExpr,
		Message:   req.Message,
		Recipient: req.Recipient,
		Enabled:   true,
		NextRunAt: nextRunAt,
	}
	created, err := s.Store.CreateCronJob(job)
	if err != nil {
		jsonError(w, "create failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

// PUT /api/bots/{id}/cron-jobs/{jid}
func (s *Server) handleUpdateCronJob(w http.ResponseWriter, r *http.Request) {
	botID := r.PathValue("id")
	jobID := r.PathValue("jid")
	userID := auth.UserIDFromContext(r.Context())

	bot, err := s.Store.GetBot(botID)
	if err != nil || bot.UserID != userID {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	existing, err := s.Store.GetCronJob(jobID)
	if err != nil || existing.BotID != botID {
		jsonError(w, "job not found", http.StatusNotFound)
		return
	}

	var req struct {
		Name      *string `json:"name"`
		CronExpr  *string `json:"cron_expr"`
		Message   *string `json:"message"`
		Recipient *string `json:"recipient"`
		Enabled   *bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	name := existing.Name
	cronExpr := existing.CronExpr
	message := existing.Message
	recipient := existing.Recipient
	enabled := existing.Enabled
	scheduleChanged := false

	if req.Name != nil {
		name = *req.Name
	}
	if req.CronExpr != nil {
		if err := cron.Validate(*req.CronExpr); err != nil {
			jsonError(w, "invalid cron expression: "+err.Error(), http.StatusBadRequest)
			return
		}
		scheduleChanged = *req.CronExpr != existing.CronExpr
		cronExpr = *req.CronExpr
	}
	if req.Message != nil {
		if *req.Message == "" {
			jsonError(w, "message cannot be empty", http.StatusBadRequest)
			return
		}
		message = *req.Message
	}
	if req.Recipient != nil {
		recipient = *req.Recipient
	}
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	// Recalculate next_run_at only when schedule changed or job was re-enabled.
	nextRunAt := existing.NextRunAt
	reEnabled := req.Enabled != nil && *req.Enabled && !existing.Enabled
	if !enabled {
		nextRunAt = nil
	} else if scheduleChanged || reEnabled || existing.NextRunAt == nil {
		next, err := cron.NextAfter(cronExpr, time.Now())
		if err != nil {
			jsonError(w, "unable to schedule cron job: "+err.Error(), http.StatusBadRequest)
			return
		}
		v := next.Unix()
		nextRunAt = &v
	}

	if err := s.Store.UpdateCronJob(jobID, name, cronExpr, message, recipient, enabled, nextRunAt); err != nil {
		jsonError(w, "update failed", http.StatusInternalServerError)
		return
	}

	updated, err := s.Store.GetCronJob(jobID)
	if err != nil {
		jsonError(w, "update readback failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}

// DELETE /api/bots/{id}/cron-jobs/{jid}
func (s *Server) handleDeleteCronJob(w http.ResponseWriter, r *http.Request) {
	botID := r.PathValue("id")
	jobID := r.PathValue("jid")
	userID := auth.UserIDFromContext(r.Context())

	bot, err := s.Store.GetBot(botID)
	if err != nil || bot.UserID != userID {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	existing, err := s.Store.GetCronJob(jobID)
	if err != nil || existing.BotID != botID {
		jsonError(w, "job not found", http.StatusNotFound)
		return
	}

	if err := s.Store.DeleteCronJob(jobID); err != nil {
		jsonError(w, "delete failed", http.StatusInternalServerError)
		return
	}
	jsonOK(w)
}
