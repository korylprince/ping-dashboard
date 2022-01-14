package main

import (
	"net/http"

	tollbooth "github.com/didip/tollbooth/v6"
	"github.com/didip/tollbooth/v6/limiter"
)

// LimitHandler is a middleware that performs rate-limiting
func LimitHandler(lmt *limiter.Limiter, next http.Handler) http.Handler {
	middle := func(w http.ResponseWriter, r *http.Request) {
		httpError := tollbooth.LimitByRequest(lmt, w, r)
		if httpError != nil {
			l := r.Context().Value(ContextKeyLog).(*Log)
			l.Error = &Error{httpError}

			w.WriteHeader(http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(middle)
}
