package main

import (
	"bytes"
	"crypto/subtle"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

const cookieName = "auth"

// AuthHandler is a HTTP Basic Auth handler
func (s *Service) AuthHandler() http.Handler {
	user := []byte(s.Config.Username)
	userLen := int32(len(user))
	pass := []byte(s.Config.Password)
	passLen := int32(len(pass))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok ||
			subtle.ConstantTimeEq(userLen, int32(len([]byte(u)))) != 1 ||
			subtle.ConstantTimeCompare(user, []byte(u)) != 1 ||
			subtle.ConstantTimeEq(passLen, int32(len([]byte(p)))) != 1 ||
			subtle.ConstantTimeCompare(pass, []byte(p)) != 1 {
			w.Header().Set("WWW-Authenticate", "Basic")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     cookieName,
			Value:    s.token,
			Domain:   r.Host,
			Path:     "/",
			Expires:  time.Now().Add(s.Config.SessionDuration),
			HttpOnly: true,
		})
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	})
}

// RequireAuth is an HTTP middleware that verifies authentication and uses the unauth handler if authentication fails
func (s *Service) RequireAuth(next, unauth http.Handler) http.Handler {
	token := []byte(s.token)
	tokenLen := int32(len(s.token))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(cookieName)
		if err != nil {
			unauth.ServeHTTP(w, r)
			return
		}
		if subtle.ConstantTimeEq(tokenLen, int32(len([]byte(c.Value)))) != 1 ||
			subtle.ConstantTimeCompare(token, []byte(c.Value)) != 1 {
			unauth.ServeHTTP(w, r)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     cookieName,
			Value:    s.token,
			Domain:   r.Host,
			Path:     "/",
			Expires:  time.Now().Add(s.Config.SessionDuration),
			HttpOnly: true,
		})

		next.ServeHTTP(w, r)
	})
}

// RejectAuthRedirect redirects the client to the authentication handler
func (s *Service) RejectAuthRedirect() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/auth", http.StatusTemporaryRedirect)
	})
}

// RejectAuthWebsocket notifies the client via the websocket that authentication failed
func (s *Service) RejectAuthWebsocket() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := r.Context().Value(ContextKeyLog).(*Log)
		c, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			l.Error = &Error{fmt.Errorf("could not start websocket conn: %w", err)}
			return
		}
		if err = c.WriteJSON(map[string]string{"t": "u"}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			l.Error = &Error{fmt.Errorf("could not write unauthenticated message: %w", err)}
			return
		}
	})
}

// HandlePing returns an http.Handler that pings hosts and returns the information via a websocket
func (s *Service) HandlePing() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := r.Context().Value(ContextKeyLog).(*Log)

		buf, err := os.ReadFile(s.Config.HostsPath)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			l.Error = &Error{fmt.Errorf("could not read hosts file: %w", err)}
			return
		}

		schema, err := UnmarshalSchema(bytes.NewBuffer(buf))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			l.Error = &Error{fmt.Errorf("could not parse hosts file: %w", err)}
			return
		}

		c, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			l.Error = &Error{fmt.Errorf("could not start websocket conn: %w", err)}
			return
		}

		if err = s.HandleConn(c, schema); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			l.Error = &Error{fmt.Errorf("could not finish websocket conn: %w", err)}
			return
		}
	})
}
