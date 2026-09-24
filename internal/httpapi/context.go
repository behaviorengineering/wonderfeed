package httpapi

import (
	"context"
	"net/http"
	"time"
)

func contextWithTimeout(r *http.Request, budget time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), budget)
}
