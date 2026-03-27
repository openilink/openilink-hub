package api

import (
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/openilink/openilink-hub/internal/auth"
	"github.com/openilink/openilink-hub/internal/store"
)

// --- WebAuthn registration ---

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
	if _, err := s.Store.GetUserByUsername(req.Username); err == nil {
		jsonError(w, "username already taken", http.StatusConflict)
		return
	}

	displayName := req.DisplayName
	if displayName == "" {
		displayName = req.Username
	}

	// First user becomes admin; subsequent registrations respect the registration gate
	count, _ := s.Store.UserCount()
	if count > 0 && !s.registrationEnabled() {
		jsonError(w, "registration is disabled", http.StatusForbidden)
		return
	}

	role := store.RoleMember
	if count == 0 {
		role = store.RoleSuperAdmin
	}

	user, err := s.Store.CreateUserFull(req.Username, "", displayName, "", role)
	if err != nil {
		jsonError(w, "create user failed", http.StatusInternalServerError)
		return
	}

	waUser, _ := auth.LoadWebAuthnUser(s.Store, user)
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

	user, err := s.Store.GetUserByUsername(username)
	if err != nil {
		jsonError(w, "user not found", http.StatusBadRequest)
		return
	}

	session := s.SessionStore.Get("reg:" + user.ID)
	if session == nil {
		jsonError(w, "no registration session", http.StatusBadRequest)
		return
	}

	waUser, _ := auth.LoadWebAuthnUser(s.Store, user)
	cred, err := s.WebAuthn.FinishRegistration(waUser, *session, r)
	if err != nil {
		jsonError(w, "registration failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	transportsJSON, _ := json.Marshal(cred.Transport)
	if err := s.Store.SaveCredential(&store.Credential{
		ID:              base64.RawURLEncoding.EncodeToString(cred.ID),
		UserID:          user.ID,
		PublicKey:       cred.PublicKey,
		AttestationType: cred.AttestationType,
		Transport:       string(transportsJSON),
		SignCount:       cred.Authenticator.SignCount,
		BackupEligible:  cred.Flags.BackupEligible,
		BackupState:     cred.Flags.BackupState,
	}); err != nil {
		jsonError(w, "save credential failed", http.StatusInternalServerError)
		return
	}

	token, _ := auth.CreateSession(s.Store, user.ID)
	setSessionCookie(w, token)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"ok": true, "user": user})
}

// --- WebAuthn login ---

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

	user, err := s.Store.GetUserByUsername(req.Username)
	if err != nil {
		jsonError(w, "user not found", http.StatusBadRequest)
		return
	}
	if user.Status != store.StatusActive {
		jsonError(w, "account disabled", http.StatusForbidden)
		return
	}

	waUser, _ := auth.LoadWebAuthnUser(s.Store, user)
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
				user, err := s.Store.GetUserByID(string(userHandle))
				if err != nil {
					return nil, err
				}
				return auth.LoadWebAuthnUser(s.Store, user)
			},
			*session, parsedResponse,
		)
		if err != nil {
			jsonError(w, "login failed: "+err.Error(), http.StatusUnauthorized)
			return
		}
		userID = string(parsedResponse.Response.UserHandle)
	} else {
		user, err := s.Store.GetUserByUsername(username)
		if err != nil {
			jsonError(w, "user not found", http.StatusBadRequest)
			return
		}
		session := s.SessionStore.Get("login:" + user.ID)
		if session == nil {
			jsonError(w, "no login session", http.StatusBadRequest)
			return
		}
		waUser, _ := auth.LoadWebAuthnUser(s.Store, user)
		_, err = s.WebAuthn.FinishLogin(waUser, *session, r)
		if err != nil {
			jsonError(w, "login failed: "+err.Error(), http.StatusUnauthorized)
			return
		}
		userID = user.ID
	}

	token, _ := auth.CreateSession(s.Store, userID)
	setSessionCookie(w, token)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"ok": true, "user_id": userID})
}

// --- Passkey binding (authenticated) ---

func (s *Server) handleListPasskeys(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	creds, err := s.Store.GetCredentialsByUserID(userID)
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
	user, err := s.Store.GetUserByID(userID)
	if err != nil {
		jsonError(w, "user not found", http.StatusNotFound)
		return
	}

	waUser, _ := auth.LoadWebAuthnUser(s.Store, user)
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
	user, err := s.Store.GetUserByID(userID)
	if err != nil {
		jsonError(w, "user not found", http.StatusNotFound)
		return
	}

	session := s.SessionStore.Get("bind:" + user.ID)
	if session == nil {
		jsonError(w, "no registration session", http.StatusBadRequest)
		return
	}

	waUser, _ := auth.LoadWebAuthnUser(s.Store, user)
	cred, err := s.WebAuthn.FinishRegistration(waUser, *session, r)
	if err != nil {
		jsonError(w, "registration failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	transportsJSON, _ := json.Marshal(cred.Transport)
	if err := s.Store.SaveCredential(&store.Credential{
		ID:              base64.RawURLEncoding.EncodeToString(cred.ID),
		UserID:          user.ID,
		PublicKey:       cred.PublicKey,
		AttestationType: cred.AttestationType,
		Transport:       string(transportsJSON),
		SignCount:       cred.Authenticator.SignCount,
		BackupEligible:  cred.Flags.BackupEligible,
		BackupState:     cred.Flags.BackupState,
	}); err != nil {
		jsonError(w, "save credential failed", http.StatusInternalServerError)
		return
	}

	jsonOK(w)
}

func (s *Server) handleDeletePasskey(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	credID := r.PathValue("id")
	if err := s.Store.DeleteCredential(credID, userID); err != nil {
		jsonError(w, "delete failed", http.StatusInternalServerError)
		return
	}
	jsonOK(w)
}
