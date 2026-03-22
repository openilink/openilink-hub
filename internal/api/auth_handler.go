package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/openilink/openilink-hub/internal/auth"
	"github.com/openilink/openilink-hub/internal/database"
)

func (s *Server) handleRegisterBegin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" {
		http.Error(w, `{"error":"username required"}`, http.StatusBadRequest)
		return
	}

	// Create user if not exists
	user, err := s.DB.GetUserByUsername(req.Username)
	if err == sql.ErrNoRows {
		displayName := req.DisplayName
		if displayName == "" {
			displayName = req.Username
		}
		user, err = s.DB.CreateUser(req.Username, displayName)
	}
	if err != nil {
		http.Error(w, `{"error":"create user failed"}`, http.StatusInternalServerError)
		return
	}

	waUser, _ := auth.LoadWebAuthnUser(s.DB, user)
	options, session, err := s.WebAuthn.BeginRegistration(waUser)
	if err != nil {
		http.Error(w, `{"error":"webauthn begin failed"}`, http.StatusInternalServerError)
		return
	}

	s.SessionStore.Set("reg:"+user.ID, session)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(options)
}

func (s *Server) handleRegisterFinish(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
	}
	// Parse username from query or a wrapper — for simplicity, from cookie set during begin
	// Actually we need the username to look up the session. Let's use a query param.
	username := r.URL.Query().Get("username")
	if username == "" {
		json.NewDecoder(r.Body).Decode(&req)
		username = req.Username
	}

	user, err := s.DB.GetUserByUsername(username)
	if err != nil {
		http.Error(w, `{"error":"user not found"}`, http.StatusBadRequest)
		return
	}

	session := s.SessionStore.Get("reg:" + user.ID)
	if session == nil {
		http.Error(w, `{"error":"no registration session"}`, http.StatusBadRequest)
		return
	}

	waUser, _ := auth.LoadWebAuthnUser(s.DB, user)
	cred, err := s.WebAuthn.FinishRegistration(waUser, *session, r)
	if err != nil {
		http.Error(w, `{"error":"registration failed: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	// Save credential
	transportsJSON, _ := json.Marshal(cred.Transport)
	if err := s.DB.SaveCredential(&database.Credential{
		ID:              string(cred.ID),
		UserID:          user.ID,
		PublicKey:       cred.PublicKey,
		AttestationType: cred.AttestationType,
		Transport:       string(transportsJSON),
		SignCount:       cred.Authenticator.SignCount,
	}); err != nil {
		http.Error(w, `{"error":"save credential failed"}`, http.StatusInternalServerError)
		return
	}

	// Create session
	token, _ := auth.CreateSession(s.DB, user.ID)
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   7 * 24 * 3600,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"ok": true, "user_id": user.ID})
}

func (s *Server) handleLoginBegin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" {
		// Discoverable credential flow (no username)
		options, session, err := s.WebAuthn.BeginDiscoverableLogin()
		if err != nil {
			http.Error(w, `{"error":"webauthn begin failed"}`, http.StatusInternalServerError)
			return
		}
		s.SessionStore.Set("login:discoverable", session)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(options)
		return
	}

	user, err := s.DB.GetUserByUsername(req.Username)
	if err != nil {
		http.Error(w, `{"error":"user not found"}`, http.StatusBadRequest)
		return
	}

	waUser, _ := auth.LoadWebAuthnUser(s.DB, user)
	options, session, err := s.WebAuthn.BeginLogin(waUser)
	if err != nil {
		http.Error(w, `{"error":"webauthn begin failed"}`, http.StatusInternalServerError)
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
		// Discoverable login
		session := s.SessionStore.Get("login:discoverable")
		if session == nil {
			http.Error(w, `{"error":"no login session"}`, http.StatusBadRequest)
			return
		}

		parsedResponse, err := protocol.ParseCredentialRequestResponse(r)
		if err != nil {
			http.Error(w, `{"error":"parse response failed"}`, http.StatusBadRequest)
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
			*session,
			parsedResponse,
		)
		if err != nil {
			http.Error(w, `{"error":"login failed: `+err.Error()+`"}`, http.StatusUnauthorized)
			return
		}
		userID = string(parsedResponse.Response.UserHandle)
	} else {
		user, err := s.DB.GetUserByUsername(username)
		if err != nil {
			http.Error(w, `{"error":"user not found"}`, http.StatusBadRequest)
			return
		}

		session := s.SessionStore.Get("login:" + user.ID)
		if session == nil {
			http.Error(w, `{"error":"no login session"}`, http.StatusBadRequest)
			return
		}

		waUser, _ := auth.LoadWebAuthnUser(s.DB, user)
		_, err = s.WebAuthn.FinishLogin(waUser, *session, r)
		if err != nil {
			http.Error(w, `{"error":"login failed: `+err.Error()+`"}`, http.StatusUnauthorized)
			return
		}
		userID = user.ID
	}

	token, _ := auth.CreateSession(s.DB, userID)
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   7 * 24 * 3600,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"ok": true, "user_id": userID})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("session"); err == nil {
		auth.DeleteSession(s.DB, cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	user, err := s.DB.GetUserByID(userID)
	if err != nil {
		http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}
