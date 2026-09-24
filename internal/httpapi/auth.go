package httpapi

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/behaviorengineering/wonderfeed/internal/apperr"
)

// ParentAuth middleware requires a bearer token when ParentAuthKey is non-empty.
// When ParentAuthKey is empty (loopback-only mode), requests are allowed without a token.
type ParentAuth struct {
	ParentAuthKey string
}

func (a ParentAuth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.ParentAuthKey == "" {
			next.ServeHTTP(w, r)
			return
		}
		auth := r.Header.Get("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(auth, prefix) {
			writeError(w, apperr.New(apperr.CodeUnauthorized, "httpapi.ParentAuth", "missing bearer token"))
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(auth, prefix))
		if subtle.ConstantTimeCompare([]byte(token), []byte(a.ParentAuthKey)) != 1 {
			writeError(w, apperr.New(apperr.CodeUnauthorized, "httpapi.ParentAuth", "invalid bearer token"))
			return
		}
		next.ServeHTTP(w, r)
	})
}
