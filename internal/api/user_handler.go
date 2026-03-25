package api

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/openilink/openilink-hub/internal/auth"
	"github.com/openilink/openilink-hub/internal/store"
)

// requireAdmin is a middleware that rejects non-admin users.
func (s *Server) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := auth.UserIDFromContext(r.Context())
		user, err := s.Store.GetUserByID(userID)
		if err != nil || !store.IsAdmin(user.Role) {
			jsonError(w, "admin required", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.Store.ListUsers()
	if err != nil {
		jsonError(w, "list failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username    string `json:"username"`
		Password    string `json:"password"`
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
		Role        string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" || req.Password == "" {
		jsonError(w, "username and password required", http.StatusBadRequest)
		return
	}
	if len(req.Password) < 8 {
		jsonError(w, "password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	role := req.Role
	if role != store.RoleAdmin && role != store.RoleMember {
		role = store.RoleMember
	}
	displayName := req.DisplayName
	if displayName == "" {
		displayName = req.Username
	}

	hash := auth.HashPassword(req.Password)
	user, err := s.Store.CreateUserFull(req.Username, req.Email, displayName, hash, role)
	if err != nil {
		jsonError(w, "create user failed: "+err.Error(), http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func (s *Server) handleUpdateUserRole(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.Role != store.RoleAdmin && req.Role != store.RoleMember {
		jsonError(w, "role must be admin or member", http.StatusBadRequest)
		return
	}

	// Protect superadmin
	target, err := s.Store.GetUserByID(id)
	if err != nil {
		jsonError(w, "user not found", http.StatusNotFound)
		return
	}
	if target.Role == store.RoleSuperAdmin {
		jsonError(w, "cannot change superadmin role", http.StatusForbidden)
		return
	}

	// Prevent self-demotion
	currentUserID := auth.UserIDFromContext(r.Context())
	if id == currentUserID && !store.IsAdmin(req.Role) {
		jsonError(w, "cannot demote yourself", http.StatusBadRequest)
		return
	}

	if err := s.Store.UpdateUserRole(id, req.Role); err != nil {
		jsonError(w, "update failed", http.StatusInternalServerError)
		return
	}
	jsonOK(w)
}

func (s *Server) handleUpdateUserStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.Status != store.StatusActive && req.Status != store.StatusDisabled {
		jsonError(w, "status must be active or disabled", http.StatusBadRequest)
		return
	}

	// Protect superadmin
	target, err := s.Store.GetUserByID(id)
	if err != nil {
		jsonError(w, "user not found", http.StatusNotFound)
		return
	}
	if target.Role == store.RoleSuperAdmin {
		jsonError(w, "cannot disable superadmin", http.StatusForbidden)
		return
	}

	// Prevent self-disable
	currentUserID := auth.UserIDFromContext(r.Context())
	if id == currentUserID {
		jsonError(w, "cannot disable yourself", http.StatusBadRequest)
		return
	}

	if err := s.Store.UpdateUserStatus(id, req.Status); err != nil {
		jsonError(w, "update failed", http.StatusInternalServerError)
		return
	}

	// Invalidate all sessions for disabled user
	if req.Status == store.StatusDisabled {
		s.Store.DeleteSessionsByUserID(id)
	}
	jsonOK(w)
}

func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	// Protect superadmin
	target, err := s.Store.GetUserByID(id)
	if err != nil {
		jsonError(w, "user not found", http.StatusNotFound)
		return
	}
	if target.Role == store.RoleSuperAdmin {
		jsonError(w, "cannot delete superadmin", http.StatusForbidden)
		return
	}

	currentUserID := auth.UserIDFromContext(r.Context())
	if id == currentUserID {
		jsonError(w, "cannot delete yourself", http.StatusBadRequest)
		return
	}

	if err := s.Store.DeleteUser(id); err != nil {
		jsonError(w, "delete failed", http.StatusInternalServerError)
		return
	}
	jsonOK(w)
}

func (s *Server) handleResetUserPassword(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	// Generate random password
	b := make([]byte, 12)
	rand.Read(b)
	password := base64.RawURLEncoding.EncodeToString(b)[:16]

	hash := auth.HashPassword(password)
	if err := s.Store.UpdateUserPassword(id, hash); err != nil {
		jsonError(w, "update failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"password": password})
}
