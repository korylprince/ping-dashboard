package main

import (
	"bytes"
	"crypto/subtle"
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
)

// BasicAuthHandler is a HTTP Basic Auth handler
func BasicAuthHandler(username, password string, next http.Handler) http.Handler {
	user := []byte(username)
	userLen := int32(len(user))
	pass := []byte(password)
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

		next.ServeHTTP(w, r)
	})
}

// HandleToken returns an http.Handler that returns the service token
func (s *Service) HandleToken() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(s.token))
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
