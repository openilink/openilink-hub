package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/openilink/openilink-hub/internal/auth"
	"github.com/openilink/openilink-hub/internal/database"
)

// --- Password auth ---

func (s *Server) handlePasswordRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username    string `json:"username"`
		Password    string `json:"password"`
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.Username == "" || req.Password == "" {
		jsonError(w, "username and password required", http.StatusBadRequest)
		return
	}
	if len(req.Username) < 2 || len(req.Username) > 32 {
		jsonError(w, "username must be 2-32 characters", http.StatusBadRequest)
		return
	}
	if len(req.Password) < 8 {
		jsonError(w, "password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	// Check if username taken
	if _, err := s.DB.GetUserByUsername(req.Username); err == nil {
		jsonError(w, "username already taken", http.StatusConflict)
		return
	}

	displayName := req.DisplayName
	if displayName == "" {
		displayName = req.Username
	}

	// First user becomes admin
	role := database.RoleMember
	count, _ := s.DB.UserCount()
	if count == 0 {
		role = database.RoleSuperAdmin
	}

	hash := auth.HashPassword(req.Password)
	user, err := s.DB.CreateUserFull(req.Username, req.Email, displayName, hash, role)
	if err != nil {
		jsonError(w, "create user failed", http.StatusInternalServerError)
		return
	}

	token, _ := auth.CreateSession(s.DB, user.ID)
	setSessionCookie(w, token)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"ok": true, "user": user})
}

func (s *Server) handlePasswordLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" || req.Password == "" {
		jsonError(w, "username and password required", http.StatusBadRequest)
		return
	}

	user, err := s.DB.GetUserByUsername(req.Username)
	if err != nil {
		jsonError(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	if user.Status != database.StatusActive {
		jsonError(w, "account disabled", http.StatusForbidden)
		return
	}
	if user.PasswordHash == "" || !auth.CheckPassword(req.Password, user.PasswordHash) {
		jsonError(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	token, _ := auth.CreateSession(s.DB, user.ID)
	setSessionCookie(w, token)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"ok": true, "user": user})
}

// --- WebAuthn ---

func (s *Server) handleRegisterBegin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" {
		jsonError(w, "username required", http.StatusBadRequest)
		return
	}
	if len(req.Username) < 2 || len(req.Username) > 32 {
		jsonError(w, "username must be 2-32 characters", http.StatusBadRequest)
		return
	}

	// Check if username already taken
	if _, err := s.DB.GetUserByUsername(req.Username); err == nil {
		jsonError(w, "username already taken", http.StatusConflict)
		return
	}

	displayName := req.DisplayName
	if displayName == "" {
		displayName = req.Username
	}

	// First user becomes admin
	role := database.RoleMember
	count, _ := s.DB.UserCount()
	if count == 0 {
		role = database.RoleSuperAdmin
	}

	user, err := s.DB.CreateUserFull(req.Username, "", displayName, "", role)
	if err != nil {
		jsonError(w, "create user failed", http.StatusInternalServerError)
		return
	}

	waUser, _ := auth.LoadWebAuthnUser(s.DB, user)
	options, session, err := s.WebAuthn.BeginRegistration(waUser)
	if err != nil {
		jsonError(w, "webauthn begin failed", http.StatusInternalServerError)
		return
	}

	s.SessionStore.Set("reg:"+user.ID, session)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(options)
}

func (s *Server) handleRegisterFinish(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")

	user, err := s.DB.GetUserByUsername(username)
	if err != nil {
		jsonError(w, "user not found", http.StatusBadRequest)
		return
	}

	session := s.SessionStore.Get("reg:" + user.ID)
	if session == nil {
		jsonError(w, "no registration session", http.StatusBadRequest)
		return
	}

	waUser, _ := auth.LoadWebAuthnUser(s.DB, user)
	cred, err := s.WebAuthn.FinishRegistration(waUser, *session, r)
	if err != nil {
		jsonError(w, "registration failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	transportsJSON, _ := json.Marshal(cred.Transport)
	if err := s.DB.SaveCredential(&database.Credential{
		ID:              auth.EncodeCredentialID(cred.ID),
		UserID:          user.ID,
		PublicKey:       cred.PublicKey,
		AttestationType: cred.AttestationType,
		Transport:       string(transportsJSON),
		SignCount:       cred.Authenticator.SignCount,
	}); err != nil {
		jsonError(w, "save credential failed", http.StatusInternalServerError)
		return
	}

	token, _ := auth.CreateSession(s.DB, user.ID)
	setSessionCookie(w, token)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"ok": true, "user": user})
}

func (s *Server) handleLoginBegin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" {
		options, session, err := s.WebAuthn.BeginDiscoverableLogin()
		if err != nil {
			jsonError(w, "webauthn begin failed", http.StatusInternalServerError)
			return
		}
		s.SessionStore.Set("login:discoverable", session)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(options)
		return
	}

	user, err := s.DB.GetUserByUsername(req.Username)
	if err != nil {
		jsonError(w, "user not found", http.StatusBadRequest)
		return
	}
	if user.Status != database.StatusActive {
		jsonError(w, "account disabled", http.StatusForbidden)
		return
	}

	waUser, _ := auth.LoadWebAuthnUser(s.DB, user)
	options, session, err := s.WebAuthn.BeginLogin(waUser)
	if err != nil {
		jsonError(w, "webauthn begin failed", http.StatusInternalServerError)
		return
	}

	s.SessionStore.Set("login:"+user.ID, session)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(options)
}

func (s *Server) handleLoginFinish(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")

	var userID string
	if username == "" {
		session := s.SessionStore.Get("login:discoverable")
		if session == nil {
			jsonError(w, "no login session", http.StatusBadRequest)
			return
		}
		parsedResponse, err := protocol.ParseCredentialRequestResponse(r)
		if err != nil {
			jsonError(w, "parse response failed", http.StatusBadRequest)
			return
		}
		_, err = s.WebAuthn.ValidateDiscoverableLogin(
			func(rawID, userHandle []byte) (webauthn.User, error) {
				user, err := s.DB.GetUserByID(string(userHandle))
				if err != nil {
					return nil, err
				}
				return auth.LoadWebAuthnUser(s.DB, user)
			},
			*session, parsedResponse,
		)
		if err != nil {
			jsonError(w, "login failed: "+err.Error(), http.StatusUnauthorized)
			return
		}
		userID = string(parsedResponse.Response.UserHandle)
	} else {
		user, err := s.DB.GetUserByUsername(username)
		if err != nil {
			jsonError(w, "user not found", http.StatusBadRequest)
			return
		}
		session := s.SessionStore.Get("login:" + user.ID)
		if session == nil {
			jsonError(w, "no login session", http.StatusBadRequest)
			return
		}
		waUser, _ := auth.LoadWebAuthnUser(s.DB, user)
		_, err = s.WebAuthn.FinishLogin(waUser, *session, r)
		if err != nil {
			jsonError(w, "login failed: "+err.Error(), http.StatusUnauthorized)
			return
		}
		userID = user.ID
	}

	token, _ := auth.CreateSession(s.DB, userID)
	setSessionCookie(w, token)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"ok": true, "user_id": userID})
}

// --- Session ---

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("session"); err == nil {
		auth.DeleteSession(s.DB, cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name: "session", Value: "", Path: "/",
		HttpOnly: true, MaxAge: -1, Expires: time.Unix(0, 0),
	})
	jsonOK(w)
}

// --- Passkey binding (authenticated) ---

func (s *Server) handleListPasskeys(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	creds, err := s.DB.GetCredentialsByUserID(userID)
	if err != nil {
		jsonError(w, "list failed", http.StatusInternalServerError)
		return
	}
	type passkeyResp struct {
		ID        string `json:"id"`
		CreatedAt int64  `json:"created_at"`
	}
	result := make([]passkeyResp, len(creds))
	for i, c := range creds {
		result[i] = passkeyResp{ID: c.ID, CreatedAt: c.CreatedAt}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) handlePasskeyBindBegin(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	user, err := s.DB.GetUserByID(userID)
	if err != nil {
		jsonError(w, "user not found", http.StatusNotFound)
		return
	}

	waUser, _ := auth.LoadWebAuthnUser(s.DB, user)
	options, session, err := s.WebAuthn.BeginRegistration(waUser)
	if err != nil {
		jsonError(w, "webauthn begin failed", http.StatusInternalServerError)
		return
	}

	s.SessionStore.Set("bind:"+user.ID, session)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(options)
}

func (s *Server) handlePasskeyBindFinish(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	user, err := s.DB.GetUserByID(userID)
	if err != nil {
		jsonError(w, "user not found", http.StatusNotFound)
		return
	}

	session := s.SessionStore.Get("bind:" + user.ID)
	if session == nil {
		jsonError(w, "no registration session", http.StatusBadRequest)
		return
	}

	waUser, _ := auth.LoadWebAuthnUser(s.DB, user)
	cred, err := s.WebAuthn.FinishRegistration(waUser, *session, r)
	if err != nil {
		jsonError(w, "registration failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	transportsJSON, _ := json.Marshal(cred.Transport)
	if err := s.DB.SaveCredential(&database.Credential{
		ID:              auth.EncodeCredentialID(cred.ID),
		UserID:          user.ID,
		PublicKey:       cred.PublicKey,
		AttestationType: cred.AttestationType,
		Transport:       string(transportsJSON),
		SignCount:       cred.Authenticator.SignCount,
	}); err != nil {
		jsonError(w, "save credential failed", http.StatusInternalServerError)
		return
	}

	jsonOK(w)
}

func (s *Server) handleDeletePasskey(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	credID := r.PathValue("id")
	if err := s.DB.DeleteCredential(credID, userID); err != nil {
		jsonError(w, "delete failed", http.StatusInternalServerError)
		return
	}
	jsonOK(w)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	user, err := s.DB.GetUserByID(userID)
	if err != nil {
		jsonError(w, "user not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (s *Server) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	var req struct {
		DisplayName string `json:"display_name"`
		Email       string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	if err := s.DB.UpdateUserProfile(userID, req.DisplayName, req.Email); err != nil {
		jsonError(w, "update failed", http.StatusInternalServerError)
		return
	}
	jsonOK(w)
}

func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.NewPassword == "" {
		jsonError(w, "new_password required", http.StatusBadRequest)
		return
	}
	if len(req.NewPassword) < 8 {
		jsonError(w, "password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	user, err := s.DB.GetUserByID(userID)
	if err != nil {
		jsonError(w, "user not found", http.StatusNotFound)
		return
	}
	// If user already has a password, verify old password
	if user.PasswordHash != "" {
		if req.OldPassword == "" || !auth.CheckPassword(req.OldPassword, user.PasswordHash) {
			jsonError(w, "old password incorrect", http.StatusUnauthorized)
			return
		}
	}

	hash := auth.HashPassword(req.NewPassword)
	if err := s.DB.UpdateUserPassword(userID, hash); err != nil {
		jsonError(w, "update failed", http.StatusInternalServerError)
		return
	}
	jsonOK(w)
}

// --- Helpers ---

func setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name: "session", Value: token, Path: "/",
		HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: 7 * 24 * 3600,
	})
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func jsonOK(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}
