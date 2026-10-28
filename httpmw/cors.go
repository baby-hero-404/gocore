package httpmw

import (
	"net/http"
	"strings"

	"github.com/baby-hero-404/gocore/stdlog"
)

// CORSConfig configures CORS. The fields match httpserver.Config so both stacks take the same values.
type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	AllowCredentials bool
}

// CORSPolicy decides which CORS headers a request gets. It is shared by the net/http middleware and
// the Gin adapter so the security rules live in one place.
//
// Only requests carrying an Origin header get CORS headers. An origin listed explicitly in
// AllowOrigins is answered with that origin (plus Vary: Origin, and Access-Control-Allow-Credentials
// when AllowCredentials is set). A "*" entry allows any other origin with the literal "*" and never
// with credentials, so a wildcard cannot be combined with credentialed cross-origin access.
// Origins that match nothing receive no CORS headers, which makes browsers block the call.
type CORSPolicy struct {
	listed      map[string]struct{}
	allowAny    bool
	credentials bool
	methods     string
	headers     string
}

// NewCORSPolicy builds a policy from cfg. It logs a warning when credentials are requested together
// with a wildcard origin, because the wildcard never carries credentials.
func NewCORSPolicy(cfg CORSConfig) *CORSPolicy {
	p := &CORSPolicy{
		listed:      make(map[string]struct{}, len(cfg.AllowOrigins)),
		credentials: cfg.AllowCredentials,
		methods:     strings.Join(cfg.AllowMethods, ", "),
		headers:     strings.Join(cfg.AllowHeaders, ", "),
	}
	for _, o := range cfg.AllowOrigins {
		if o == "*" {
			p.allowAny = true
			continue
		}
		p.listed[o] = struct{}{}
	}
	if p.allowAny && p.credentials {
		stdlog.Warn("cors: AllowCredentials is ignored for the wildcard origin; list the origins explicitly to allow credentials")
	}
	return p
}

// Apply writes the CORS response headers for a request with the given Origin header value.
func (p *CORSPolicy) Apply(h http.Header, origin string) {
	if origin == "" {
		return
	}
	if len(p.listed) > 0 {
		h.Add("Vary", "Origin")
	}
	_, isListed := p.listed[origin]
	if !isListed && !p.allowAny {
		return
	}
	if isListed {
		h.Set("Access-Control-Allow-Origin", origin)
		if p.credentials {
			h.Set("Access-Control-Allow-Credentials", "true")
		}
	} else {
		h.Set("Access-Control-Allow-Origin", "*")
	}
	h.Set("Access-Control-Allow-Methods", p.methods)
	h.Set("Access-Control-Allow-Headers", p.headers)
	h.Set("Access-Control-Expose-Headers", HeaderRequestID)
}

// CORS applies the policy and answers OPTIONS requests with 204 without calling the next handler.
func CORS(cfg CORSConfig) Middleware {
	policy := NewCORSPolicy(cfg)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			policy.Apply(w.Header(), r.Header.Get("Origin"))
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
