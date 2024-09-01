package gcrepro

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/mailgun/groupcache/v2"
)

const (
	// defaultTimeout defines the default timeout for crosstalk requests.
	defaultTimeout = 20 * time.Second
)

// crosstalkIngressContextFromRequest returns a context from the given request.
func crosstalkIngressContextFromRequest(r *http.Request) context.Context {
	return r.Context()
}

// crosstalkEgressTransport returns an http.Roundtripper that uses the provided context and wraps http.DefaultTransport.
func crosstalkEgressTransport(ctx context.Context) http.RoundTripper {
	return &crosstalkTransport{ctx: ctx}
}

// crosstalkTransport uses the standard library http.DefaultTransport but adds transactionID headers to all requests.
type crosstalkTransport struct {
	ctx context.Context
}

// RoundTrip adds transaction headers from the request context to the http request.
// It also adds a timeout with a duration of defaultTimeout
func (t *crosstalkTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Important: check the transport context rather than the request context.
	// groupcache passes the context to the transport not the request.
	if isFromCrosstalkIngress(t.ctx) { // not the request context
		return nil, errors.New("stopping multiple chained crosstalk requests")
	}

	ctx, cancel := context.WithTimeout(req.Context(), defaultTimeout)
	defer cancel()

	return http.DefaultTransport.RoundTrip(req.WithContext(ctx))
}

func handleCrosstalk(pool *groupcache.HTTPPool) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ctx = markCrosstalkIngress(ctx)

		// ...

		pool.ServeHTTP(w, r.WithContext(ctx))
	})
}

// unexported type and key for context storage
type crosstalkCtxKeyType string

const crosstalkCtxKey = crosstalkCtxKeyType("crosstalk.context")

func isFromCrosstalkIngress(c context.Context) bool {
	v, ok := c.Value(crosstalkCtxKey).(bool)
	return ok && v
}

func markCrosstalkIngress(c context.Context) context.Context {
	return context.WithValue(c, crosstalkCtxKey, true)
}
